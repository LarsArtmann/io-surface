package extract

// Detector rules: pure functions over one resolved call capture that emit
// observations. Every rule is data-driven (tables in tables.go, D6) and fires
// at info severity until fleet evidence promotes it (D4). The registry and
// the metadata table below are the two halves of the rule contract; a parity
// test pins them together.

import (
	"go/ast"
	"go/token"
	"slices"
)

// detector is one registered rule: name + the function that produces its
// observations for a single capture.
type detector struct {
	rule string
	run  func(e *extractor, c *capture) []Observation
}

// fnDetector is the declaration-shaped half of the registry: rules whose
// subject is the whole function (client literals, stat-then-open pairs),
// run once per function so per-capture execution cannot duplicate them.
type fnDetector struct {
	rule string
	run  func(e *extractor, fn *ast.FuncDecl, imports map[string]string, id string) []Observation
}

// RuleDef is the published half of the rule contract: the stable name, the
// current severity, and the finding message builder the lint face uses.
type RuleDef struct {
	Name        string
	Severity    string // "info" until T17 evidence-gated promotion (D4)
	Description string
	Message     func(o Observation) string
}

// detectors is the execution registry (order = observation emission order per
// capture; global ordering is imposed by sortedObservations).
var detectors = []detector{
	{rule: RuleErrDropped, run: detectErrDropped},
	{rule: RuleCloserMissing, run: detectCloserMissing},
	{rule: RuleCloserMissing, run: detectBodyUnclosed},
	{rule: RuleCtxDropped, run: detectCtxDropped},
	{rule: RuleDeferInLoop, run: detectDeferInLoop},
	{rule: RuleIOInLoop, run: detectIOInLoop},
	{rule: RuleClientNoTimeout, run: detectBareHTTPHelper},
	{rule: RuleTinyBuffer, run: detectTinyBuffer},
}

// fnDetectors is the declaration-shaped registry (once per function).
var fnDetectors = []fnDetector{
	{rule: RuleClientNoTimeout, run: detectClientNoTimeout},
	{rule: RuleStatThenOpen, run: detectStatThenOpen},
	{rule: RuleWriteNonAtomic, run: detectWriteNonAtomic},
}

// detectorRules is the metadata registry. Message receives the observation's
// Evidence clause and returns the complete human-facing finding message.
var detectorRules = []RuleDef{
	{
		Name:        RuleErrDropped,
		Severity:    "info",
		Description: "a watched call's error return is discarded",
		Message: func(o Observation) string {
			return o.Evidence + " — the error return is unchecked; handle it or ignore deliberately with `_ =`"
		},
	},
	{
		Name:        RuleCloserMissing,
		Severity:    "info",
		Description: "a value from a closable constructor is never closed or handed off",
		Message: func(o Observation) string {
			return o.Evidence + " — nothing closes it in this function; add `defer <name>.Close()` or hand off ownership"
		},
	},
	{
		Name:        RuleCtxDropped,
		Severity:    "info",
		Description: "a context.Context is in scope but the call uses the context-blind variant",
		Message: func(o Observation) string {
			return o.Evidence
		},
	},
	{
		Name:        RuleDeferInLoop,
		Severity:    "info",
		Description: "a deferred close or I/O call runs inside a loop but executes at function exit",
		Message: func(o Observation) string {
			return o.Evidence
		},
	},
	{
		Name:        RuleClientNoTimeout,
		Severity:    "info",
		Description: "an HTTP client or helper without a timeout",
		Message: func(o Observation) string {
			return o.Evidence
		},
	},
	{
		Name:        RuleStatThenOpen,
		Severity:    "info",
		Description: "the same path expression is stat'd and then opened (double syscall, TOCTOU)",
		Message: func(o Observation) string {
			return o.Evidence
		},
	},
	{
		Name:        RuleIOInLoop,
		Severity:    "info",
		Description: "a watched I/O call sits inside a loop body (N+1 shape)",
		Message: func(o Observation) string {
			return o.Evidence
		},
	},
	{
		Name:        RuleWriteNonAtomic,
		Severity:    "info",
		Description: "a direct final-path write without a temp+rename in the same function",
		Message: func(o Observation) string {
			return o.Evidence
		},
	},
	{
		Name:        RuleTinyBuffer,
		Severity:    "info",
		Description: "a sub-page buffer feeds a read/write helper (syscall amplification)",
		Message: func(o Observation) string {
			return o.Evidence
		},
	},
}

// DetectorRule returns the published metadata for a detector rule name —
// the lookup the lint face uses to build stable rule IDs, severities, and
// messages (D7: "io-surface/" + Name is the finding contract).
func DetectorRule(name string) (RuleDef, bool) {
	for _, r := range detectorRules {
		if r.Name == name {
			return r, true
		}
	}

	return RuleDef{}, false
}

// DetectorRules lists every rule's published metadata, in registry order —
// the vocabulary source for docs and SARIF rule descriptions.
func DetectorRules() []RuleDef {
	out := make([]RuleDef, len(detectorRules))
	copy(out, detectorRules)

	return out
}

// detectErrDropped implements D8 exactly: a blank error slot in a named
// assignment is flagged (`x, _ := os.ReadFile(p)`), a fully blank assignment
// is a deliberate ignore (`_ = os.Remove(p)`) and stays silent, bare
// statement calls drop everything and are flagged, while returned, deferred
// (cleanup defers legitimately ignore), and expression-embedded calls never
// fire.
func detectErrDropped(e *extractor, c *capture) []Observation {
	errPos, watched := errResults[c.sel]
	if !watched {
		return nil
	}
	switch c.shape {
	case shapeBare:
		return []Observation{e.emit(c, RuleErrDropped, evidenceOf(&e.fset, c.node)+" (results discarded)")}
	case shapeAssign:
		if len(c.blanks) == c.slots {
			return nil // D8: every slot blank is a deliberate ignore
		}
		for _, p := range errPos {
			if slices.Contains(c.blanks, c.slotIdx+p) {
				return []Observation{e.emit(c, RuleErrDropped, evidenceOf(&e.fset, c.parent))}
			}
		}
	}

	return nil
}

// detectCloserMissing flags closable constructor results that are never
// closed AND never handed off in the enclosing function. The ownership
// classifier is deliberately conservative: any return, call argument, or
// assignment counts as a hand-off and silences the rule — a false negative
// costs less than a false positive at info tier (D4). Method receivers other
// than Close (f.Read) do NOT transfer ownership, so the classic
// used-but-never-closed leak still fires.
func detectCloserMissing(e *extractor, c *capture) []Observation {
	if !closableCtors[c.sel] {
		return nil
	}
	if c.shape != shapeAssign || c.slotIdx != 0 {
		return nil
	}
	assign, ok := c.parent.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) == 0 {
		return nil
	}
	binding, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || binding.Name == "_" || slices.Contains(c.blanks, 0) {
		return nil
	}
	scan := closerScan{bindingName: binding.Name, bindingPos: binding.Pos()}
	ast.Walk(&scan, c.fn.Body)
	if scan.closed || scan.transferred {
		return nil
	}

	return []Observation{e.emit(c, RuleCloserMissing, evidenceOf(&e.fset, c.parent))}
}

// closerScan classifies every mention of the constructor-bound identifier.
type closerScan struct {
	bindingName string
	bindingPos  token.Pos
	stack       []ast.Node
	closed      bool
	transferred bool
	used        bool
}

func (s *closerScan) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		s.stack = s.stack[:len(s.stack)-1]

		return nil
	}
	var parent ast.Node
	if len(s.stack) > 0 {
		parent = s.stack[len(s.stack)-1]
	}
	s.stack = append(s.stack, n)
	if id, ok := n.(*ast.Ident); ok && id.Name == s.bindingName && id.Pos() != s.bindingPos {
		s.classify(id, parent)
	}

	return s
}

func (s *closerScan) classify(id *ast.Ident, parent ast.Node) {
	switch p := parent.(type) {
	case *ast.SelectorExpr:
		if p.X == id && p.Sel.Name == "Close" {
			s.closed = true
		} else {
			s.used = true
		}
	case *ast.ReturnStmt:
		s.transferred = true
	case *ast.CallExpr:
		if p.Fun == id {
			s.used = true
		} else {
			s.transferred = true
		}
	case *ast.AssignStmt, *ast.KeyValueExpr, *ast.SendStmt:
		s.transferred = true
	default:
		s.used = true
	}
}

// detectBodyUnclosed is the closer-missing http variant: resp.Body carries
// the close duty. Field accesses (resp.StatusCode, resp.Body reads) do not
// transfer ownership — io.ReadAll(resp.Body) without a Close is the classic
// leak and still fires — while returning or passing the whole resp silences
// the rule (ownership moved).
func detectBodyUnclosed(e *extractor, c *capture) []Observation {
	if !httpBodyCtors[c.sel] || c.shape != shapeAssign || c.slotIdx != 0 {
		return nil
	}
	assign, ok := c.parent.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) == 0 {
		return nil
	}
	binding, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || binding.Name == "_" || slices.Contains(c.blanks, 0) {
		return nil
	}
	scan := bodyScan{bindingName: binding.Name, bindingPos: binding.Pos()}
	ast.Inspect(c.fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		outer, ok := sel.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := outer.X.(*ast.Ident)
		if ok && x.Name == scan.bindingName && outer.Sel.Name == "Body" && sel.Sel.Name == "Close" {
			scan.closed = true
		}

		return true
	})
	if !scan.closed {
		ast.Walk(&scan, c.fn.Body)
	}
	if scan.closed || scan.transferred {
		return nil
	}

	return []Observation{e.emit(c, RuleCloserMissing, evidenceOf(&e.fset, c.parent))}
}

// bodyScan classifies mentions of the response binding: field accesses are
// neutral, whole-response returns/arguments/storage transfer ownership.
type bodyScan struct {
	bindingName string
	bindingPos  token.Pos
	stack       []ast.Node
	closed      bool
	transferred bool
}

func (s *bodyScan) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		s.stack = s.stack[:len(s.stack)-1]

		return nil
	}
	var parent ast.Node
	if len(s.stack) > 0 {
		parent = s.stack[len(s.stack)-1]
	}
	s.stack = append(s.stack, n)
	if id, ok := n.(*ast.Ident); ok && id.Name == s.bindingName && id.Pos() != s.bindingPos {
		switch p := parent.(type) {
		case *ast.SelectorExpr:
			if p.X == id {
				return s // field access: neutral
			}
			s.transferred = true
		case *ast.ReturnStmt:
			s.transferred = true
		case *ast.CallExpr:
			if p.Fun != id {
				s.transferred = true
			}
		case *ast.AssignStmt, *ast.KeyValueExpr, *ast.SendStmt:
			s.transferred = true
		}
	}

	return s
}

// detectCtxDropped fires when a context.Context is in scope (the enclosing
// function takes one) but the call uses the context-blind variant. The
// documented FP class: functions that deliberately detach work from the
// caller's cancellation (background cleanup) still fire at info tier — the
// finding is the nudge, not the verdict.
func detectCtxDropped(e *extractor, c *capture) []Observation {
	variant, watched := ctxVariantPairs[c.sel]
	if !watched || !c.ctxParam {
		return nil
	}

	return []Observation{e.emit(c, RuleCtxDropped, evidenceOf(&e.fset, c.node)+" — the enclosing function takes a context.Context; prefer "+variant)}
}

// detectDeferInLoop flags deferred calls inside loop bodies: defers run at
// FUNCTION exit, so every iteration's handle stays open until then. Fires on
// Close-method defers and on deferred watched I/O calls alike; plain closes
// in loops are the correct pattern and never fire.
func detectDeferInLoop(e *extractor, c *capture) []Observation {
	if !c.inLoop || c.shape != shapeDeferred {
		return nil
	}
	isClose := false
	if sel, ok := c.node.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Close" {
		isClose = true
	}
	if !isClose && c.sel == "" {
		return nil // unwatched non-Close defers are out of the I/O scope
	}

	return []Observation{e.emit(c, RuleDeferInLoop, evidenceOf(&e.fset, c.node)+" — defers run at function exit, so earlier loop iterations' resources stay open until then")}
}

// detectIOInLoop is the N+1 shape: a watched resolved I/O call inside a loop
// body. Info only — hoisting is context-dependent and the fix may be wrong
// for streamed workloads.
func detectIOInLoop(e *extractor, c *capture) []Observation {
	if !c.inLoop || c.sel == "" || c.shape == shapeDeferred {
		return nil
	}

	return []Observation{e.emit(c, RuleIOInLoop, evidenceOf(&e.fset, c.node)+" — I/O inside a loop (N+1 shape); consider hoisting or batching")}
}

// detectBareHTTPHelper flags http.Get/Post/PostForm/Head: they use the
// package default client, which has NO timeout — a hung server hangs the
// caller forever.
func detectBareHTTPHelper(e *extractor, c *capture) []Observation {
	if !httpBodyCtors[c.sel] {
		return nil
	}

	return []Observation{e.emit(c, RuleClientNoTimeout, evidenceOf(&e.fset, c.node)+" — the default client has no timeout; use a client with Timeout set (or a request with context)")}
}
