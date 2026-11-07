package extract

// Declaration-shaped detector rules and their helpers (the fnDetectors
// registry half): rules whose subject is the whole function — client
// literals, stat-then-open pairs, final-path writes — plus the small helpers
// they share. Split from detectors.go for the 650-line file gate.

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// detectClientNoTimeout flags http.Client composite literals without a
// Timeout field — declaration-shaped, run once per function.
func detectClientNoTimeout(e *extractor, fn *ast.FuncDecl, imports map[string]string, id string) []Observation {
	var obs []Observation
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok || imports[x.Name] != "http" || sel.Sel.Name != "Client" {
			return true
		}
		for _, elt := range lit.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Timeout" {
					return true
				}
			}
		}
		obs = append(obs, Observation{
			SiteID:   id,
			Rule:     RuleClientNoTimeout,
			Call:     "http.Client{}",
			Location: e.locOf(lit),
			Evidence: evidenceOf(&e.fset, lit) + " — no Timeout field; a hung server hangs this client forever",
		})

		return true
	})

	return obs
}

// statOpeners pairs a preceding stat with the calls stat-then-open guards:
// identical path expressions mean a double syscall (and a TOCTOU window).
var statOpeners = map[string]bool{
	"os.Open": true, "os.Create": true, "os.OpenFile": true,
	"os.ReadFile": true, "os.WriteFile": true,
}

// detectStatThenOpen pairs os.Stat/os.Lstat with an open/read/write of the
// SAME path expression within one function (printer-normalized comparison).
// Only exact expression matches fire — anything looser would be guesswork.
func detectStatThenOpen(e *extractor, fn *ast.FuncDecl, imports map[string]string, id string) []Observation {
	statPaths := map[string]string{} // printed path -> first stat location
	var obs []Observation
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		key := imports[x.Name]
		name := sel.Sel.Name
		if len(call.Args) == 0 {
			return true
		}
		printed := ""
		switch {
		case (key == "os" && (name == "Stat" || name == "Lstat")):
			printed = evidenceOf(&e.fset, call.Args[0])
			if _, exists := statPaths[printed]; !exists {
				statPaths[printed] = e.locOf(call)
			}
		case statOpeners[key+"."+name]:
			printed = evidenceOf(&e.fset, call.Args[0])
			if statLoc, exists := statPaths[printed]; exists && statLoc != e.locOf(call) {
				obs = append(obs, Observation{
					SiteID:   id,
					Rule:     RuleStatThenOpen,
					Call:     key + "." + name,
					Location: e.locOf(call),
					Evidence: fmt.Sprintf("%s.%s of %s (stat'd at %s) — double syscall and a TOCTOU window; open and handle the error instead", key, name, printed, statLoc),
				})
			}
		}

		return true
	})

	return obs
}

// tinyBufferThreshold is the page-size bar: reads below it mean syscall
// amplification (one Read syscall per 512 bytes instead of per 4096).
const tinyBufferThreshold = 4096

// detectTinyBuffer flags sub-page buffers that feed read/write helpers. The
// observation anchors at the call (the capture point) but names the make
// site, so the fix lands where the allocation is.
func detectTinyBuffer(e *extractor, c *capture) []Observation {
	idxs, watched := tinyBufferConsumers[c.sel]
	// The amplification only matters under repetition: a once-called tiny
	// buffer is not the bug shape (plan T22: ".Read in loop").
	if !watched || !c.inLoop || len(c.node.Args) <= idxs[0] {
		return nil
	}
	arg, ok := c.node.Args[idxs[0]].(*ast.Ident)
	if !ok {
		return nil
	}
	binding, size := makeBufferBinding(c.fn, arg.Name)
	if binding == nil || size < 0 || size >= tinyBufferThreshold {
		return nil
	}

	return []Observation{e.emit(c, RuleTinyBuffer,
		fmt.Sprintf("%s = make([]byte, %d) feeds %s — %d-byte reads amplify syscalls; size to >= %d (or use bufio.Scanner)",
			arg.Name, size, c.sel, size, tinyBufferThreshold))}
}

// makeBufferBinding finds `name := make([]byte, N)` in fn and returns the
// make call plus its literal size (-1 when the size is not a plain literal).
func makeBufferBinding(fn *ast.FuncDecl, name string) (*ast.CallExpr, int64) {
	var found *ast.CallExpr
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 || len(assign.Lhs) != 1 {
			return true
		}
		lhs, ok := assign.Lhs[0].(*ast.Ident)
		if !ok || lhs.Name != name {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		mk, ok := call.Fun.(*ast.Ident)
		if !ok || mk.Name != "make" {
			return true
		}
		arrayType, ok := call.Args[0].(*ast.ArrayType)
		if !ok {
			return true
		}
		if elt, ok := arrayType.Elt.(*ast.Ident); !ok || elt.Name != "byte" {
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok || lit.Kind != token.INT {
			return true
		}
		if _, err := strconv.ParseInt(lit.Value, 0, 64); err != nil {
			return true
		}
		found = call

		return true
	})
	if found == nil {
		return nil, -1
	}
	lit := found.Args[1].(*ast.BasicLit)
	size, _ := strconv.ParseInt(lit.Value, 0, 64)

	return found, size
}

// detectWriteNonAtomic flags direct final-path writes in functions that
// never rename a temp into place (plan T21). Functions that ARE sanctioned
// atomic-write seams are exempt — the wrapper exists precisely to own the
// raw write. The observation names the write; lint weighs durability claims.
func detectWriteNonAtomic(e *extractor, fn *ast.FuncDecl, imports map[string]string, id string) []Observation {
	if isSeamSymbol(e, id) {
		return nil
	}
	hasRename := false
	var writes []*ast.CallExpr
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		key := imports[x.Name] + "." + sel.Sel.Name
		if key == "os.Rename" {
			hasRename = true

			return true
		}
		if writeCreators[key] && (key != "os.OpenFile" || openFileCreates(e, call)) {
			writes = append(writes, call)
		}

		return true
	})
	if hasRename || len(writes) == 0 {
		return nil
	}
	obs := make([]Observation, 0, len(writes))
	for _, w := range writes {
		obs = append(obs, Observation{
			SiteID:   id,
			Rule:     RuleWriteNonAtomic,
			Call:     callKey(imports, w),
			Location: e.locOf(w),
			Evidence: evidenceOf(&e.fset, w) + " — direct final-path write without a temp+rename in this function; a crash mid-write tears the file",
		})
	}

	return obs
}

// isSeamSymbol reports whether the enclosing symbol's base name is a
// registered seam (atomicWriteFile et al.) — the sanctioned raw-write site.
func isSeamSymbol(e *extractor, id string) bool {
	if i := strings.LastIndex(id, "."); i >= 0 {
		if _, isSeam := e.seams[id[i+1:]]; isSeam {
			return true
		}
	}

	return false
}

// openFileCreates reports whether an os.OpenFile call passes creation flags
// (a name-based match on the O_* flag idents — the hermetic stand-in for
// flag-constant resolution).
func openFileCreates(e *extractor, call *ast.CallExpr) bool {
	if len(call.Args) < 2 {
		return false
	}
	printed := evidenceOf(&e.fset, call.Args[1])
	if printed == "" {
		return false
	}

	return strings.Contains(printed, "O_CREATE") || strings.Contains(printed, "O_WRONLY") ||
		strings.Contains(printed, "O_RDWR") || strings.Contains(printed, "O_APPEND")
}

// callKey renders the resolved package-qualified call name for an observation.
func callKey(imports map[string]string, call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return imports[x.Name] + "." + sel.Sel.Name
}
