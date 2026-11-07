// Package extract finds I/O call sites in Go source and attributes them to
// durable symbol anchors (pkg.Func / pkg.Type.Method / pkg.Var).
//
// A hermetic, stdlib-only AST walker (go/parser + ast.Inspect, NO go/packages,
// NO module downloads): it runs anywhere a source tree exists, including a Nix
// check sandbox with GOPROXY=off.
//
// What it detects:
//   - package-qualified calls through resolved imports (os.*, io.*, net.*,
//     tls.*, quic.*, http3.*, http.*, rand.*, rsa.*, x509.*, sql.Open,
//     exec.*, syscall.*, path/filepath walks and globs, otel exporters,
//     oidc, cqrs-lite engines)
//   - method-name tokens on unresolvable receivers (ListenAndServe, Dial,
//     RoundTrip, ExecContext, ...) — the honest weak tier, attributed to the
//     enclosing symbol
//   - same-package seam identifiers (atomicWriteFile, writeCacheFileAtomic,
//     EnqueueOp, EnqueueCommand, AppendBatch, SaveQuery, ...)
//   - //go:embed directives (comment -> following var)
//   - net.Resolver Dial-field assignments (func literals are not calls)
//
// What it deliberately does NOT do: type-check the world. Type-fact resolution
// (interface dispatch, cross-package method sets) would need go/packages plus
// the full module graph; sites invisible to the AST tier are marked
// facts.extractable=false in the manifest instead.
//
// Confidence model: package-qualified = ConfidenceResolved; method-token /
// ident-seam = ConfidenceToken. Only resolved sites may fail a check; token
// sites are advisory.
package extract

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Confidence levels, strongest first.
const (
	ConfidenceResolved = "resolved"
	ConfidenceToken    = "token"
)

// Site is one extracted I/O surface entry: a durable symbol anchor with every
// watched call location inside it.
type Site struct {
	ID         string   `json:"id"`
	Locations  []string `json:"locations"`
	Calls      []string `json:"calls"`
	Families   []string `json:"families"`
	Via        string   `json:"via,omitempty"`
	Testinfra  bool     `json:"testinfra"`
	Confidence string   `json:"confidence"`
}

type extractor struct {
	root  string
	fset  token.FileSet
	sites map[string]*Site
	seams map[string]string
	// seamDecls maps package dir -> seam ident, collected in pass A of the
	// walk (every FuncDecl named in the seam vocabulary). Pass B resolves
	// cross-package calls `pkg.Seam` by suffix-matching the caller's import
	// path against the declaring dirs — the hermetic substitute for type
	// resolution (no go/packages).
	seamDecls map[string]map[string]string
	// currentImports is the alias list of the file being parsed (per-file,
	// unlike the watched-package imports map).
	currentImports []importAlias
	// observations/captures are the detector channel (D1): non-nil only when
	// ExtractWithObservations runs, never touched by plain Extract. The
	// ignore maps ride the same lifecycle (suppression is observation-time
	// work, T5).
	observations   []Observation
	captures       map[ast.Node]*capture
	captureList    []*capture
	fnCaptureMark  int
	ignores        map[string][]ignoreDirective
	invalidIgnores []ignoreDirective
}

// importAlias pairs a file's local package alias with its full import path.
type importAlias struct {
	local string
	path  string
}

func (e *extractor) walk(dir string, opts Options, collectOnly bool) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}

			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") && !opts.IncludeTests {
			return nil
		}
		if ignored(path, e.root, opts.Ignore) {
			return nil
		}

		if collectOnly {
			e.collectSeamDecl(path)

			return nil
		}

		return e.parseFile(path)
	})
}

// ignored reports whether the file matches any --ignore glob, against the
// repo-relative slash path and the base name (so `zz_generated*` matches at
// any depth without a leading `**`).
func ignored(path, root string, patterns []string) bool {
	rel := path
	if r, err := filepath.Rel(root, path); err == nil {
		rel = r
	}
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	for _, p := range patterns {
		if ok, _ := filepath.Match(p, rel); ok {
			return true
		}
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}

	return false
}

// pkgID computes the manifest symbol prefix for a file path, relative to the
// REPOSITORY root (internal/server/x.go -> "internal/server").
func pkgID(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return ""
	}

	return filepath.ToSlash(dir)
}

// collectSeamDecl records, for the seam vocabulary only, which package dirs
// declare them: pass A of the two-pass walk. Cheap (parse only when the file
// could declare a watched ident — decided by a substring prefilter).
func (e *extractor) collectSeamDecl(path string) {
	watched := make([]string, 0, len(e.seams))
	for ident := range e.seams {
		watched = append(watched, ident)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return
	}
	containsAny := false
	for _, ident := range watched {
		if strings.Contains(string(src), "func "+ident+"(") {
			containsAny = true

			break
		}
	}
	if !containsAny {
		return
	}
	file, err := goparser.ParseFile(&e.fset, path, src, goparser.SkipObjectResolution)
	if err != nil {
		return
	}
	pkg := pkgID(e.root, path)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if fam, watchedIdent := e.seams[fn.Name.Name]; watchedIdent {
			if e.seamDecls[pkg] == nil {
				e.seamDecls[pkg] = map[string]string{}
			}
			e.seamDecls[pkg][fn.Name.Name] = fam
		}
	}
}

// resolveCrossPkgSeam matches a caller's `local.Seam` call against the seam
// declaration index: the import that `local` aliases must suffix-match a
// declaring dir (longest match wins, mirroring how Go import paths nest).
// Returns "" when local is a watched stdlib package or no seam matches.
func (e *extractor) resolveCrossPkgSeam(local, ident string) string {
	for _, imp := range e.currentImports {
		if imp.local != local {
			continue
		}
		best, bestLen := "", 0
		for dir, idents := range e.seamDecls {
			fam, ok := idents[ident]
			if !ok {
				continue
			}
			if strings.HasSuffix(imp.path, "/"+dir) || imp.path == dir && dir != "" {
				if len(dir) > bestLen {
					best, bestLen = fam, len(dir)
				}
			}
		}

		return best
	}

	return ""
}

func (e *extractor) parseFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	file, err := goparser.ParseFile(&e.fset, path, src, goparser.ParseComments)
	if err != nil {
		return nil // unparsable file: report-free skip (generated/vendored)
	}

	pkg := pkgID(e.root, path)
	testinfra := strings.Contains(filepath.ToSlash(path), "/testutil/")

	imports := map[string]string{}
	e.currentImports = e.currentImports[:0]
	for _, imp := range file.Imports {
		ipath := strings.Trim(imp.Path.Value, `"`)
		localName := filepath.Base(ipath)
		if imp.Name != nil {
			localName = imp.Name.Name
		}
		if localName != "_" && localName != "." {
			e.currentImports = append(e.currentImports, importAlias{local: localName, path: ipath})
		}
		key, ok := pkgOf[ipath]
		if !ok {
			for frag, k := range importSubstring {
				if strings.Contains(ipath, frag) {
					key, ok = k, true

					break
				}
			}
		}
		if !ok || key == "" {
			continue
		}
		local := filepath.Base(ipath)
		if imp.Name != nil {
			local = imp.Name.Name
		}
		if local == "_" || local == "." {
			continue
		}
		imports[local] = key
	}

	e.scanEmbeds(file, pkg, testinfra)
	e.collectIgnores(file)

	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		e.scanFunc(fn, imports, pkg, testinfra)

		return true
	})

	return nil
}

// symbolID builds pkg.Func or pkg.Type.Method (receiver base type).
func symbolID(pkg string, fn *ast.FuncDecl) string {
	name := fn.Name.Name
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		if _, recv := receiverType(fn.Recv.List[0].Type); recv != "" {
			return pkg + "." + recv + "." + name
		}
	}

	return pkg + "." + name
}

func (e *extractor) scanFunc(fn *ast.FuncDecl, imports map[string]string, pkg string, testinfra bool) {
	if fn.Body == nil {
		return
	}
	id := symbolID(pkg, fn)
	var fnCaptures []*capture
	if e.captures != nil {
		e.captureContexts(fn, imports, id)
		fnCaptures = e.captureList[e.fnCaptureMark:]
		e.fnCaptureMark = len(e.captureList)
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			call, ok := node.Fun.(*ast.SelectorExpr)
			if !ok {
				if ident, ok := node.Fun.(*ast.Ident); ok {
					if fam, isSeam := e.seams[ident.Name]; isSeam {
						e.record(id, node, testinfra, ident.Name, fam, ConfidenceToken, ident.Name)
					}
				}

				return true
			}
			if xIdent, ok := call.X.(*ast.Ident); ok {
				if key, imported := imports[xIdent.Name]; imported {
					if fam, watched := callFamily[key][call.Sel.Name]; watched {
						e.record(id, node, testinfra, key+"."+call.Sel.Name, fam, ConfidenceResolved, "")
						if c, ok := e.captures[node]; ok {
							c.sel, c.family = key+"."+call.Sel.Name, fam
						}

						return true
					}
				} else if fam := e.resolveCrossPkgSeam(xIdent.Name, call.Sel.Name); fam != "" {
					e.record(id, node, testinfra, xIdent.Name+"."+call.Sel.Name, fam, ConfidenceToken, call.Sel.Name)

					return true
				}
			}
			if fam, isToken := methodToken[call.Sel.Name]; isToken {
				e.record(id, node, testinfra, "."+call.Sel.Name, fam, ConfidenceToken, "")
			}
		case *ast.KeyValueExpr:
			if sel, ok := node.Key.(*ast.SelectorExpr); ok && sel.Sel.Name == "Dial" {
				if _, isFn := node.Value.(*ast.FuncLit); isFn {
					e.record(id, node, testinfra, ".Dial=", "outbound", ConfidenceToken, "")
				}
			}
		}

		return true
	})
	if e.captures != nil {
		e.runDetectors(id, fn, imports, fnCaptures)
	}
}

func (e *extractor) record(id string, n ast.Node, testinfra bool, call, family, confidence, via string) {
	s := e.site(id, testinfra)
	loc := e.locOf(n)
	if !contains(s.Locations, loc) {
		s.Locations = append(s.Locations, loc)
	}
	if !contains(s.Calls, call) {
		s.Calls = append(s.Calls, call)
	}
	if family != "" && !contains(s.Families, family) {
		s.Families = append(s.Families, family)
	}
	if confidence == ConfidenceToken {
		s.Confidence = ConfidenceToken
	}
	if via != "" && s.Via == "" {
		s.Via = via
	}
}

func (e *extractor) site(id string, testinfra bool) *Site {
	s, ok := e.sites[id]
	if !ok {
		s = &Site{ID: id, Testinfra: testinfra, Confidence: ConfidenceResolved}
		e.sites[id] = s
	}

	return s
}

// scanEmbeds finds //go:embed directives and attributes them to the following
// var declaration (the embed.FS / []byte site).
func (e *extractor) scanEmbeds(file *ast.File, pkg string, testinfra bool) {
	for _, group := range file.Comments {
		for _, c := range group.List {
			directive, ok := strings.CutPrefix(c.Text, "//go:embed ")
			if !ok {
				continue
			}
			directive = strings.TrimSpace(directive)
			varName := embedVarName(file, c.Pos())
			if varName == "" {
				continue
			}
			pos := e.fset.Position(c.Pos())
			rel := pos.Filename
			if r, err := filepath.Rel(e.root, pos.Filename); err == nil {
				rel = filepath.ToSlash(r)
			}
			s := e.site(pkg+"."+varName, testinfra)
			loc := rel + ":" + strconv.Itoa(pos.Line)
			if !contains(s.Locations, loc) {
				s.Locations = append(s.Locations, loc)
			}
			if !contains(s.Calls, "go:embed "+directive) {
				s.Calls = append(s.Calls, "go:embed "+directive)
			}
			if !contains(s.Families, "embed") {
				s.Families = append(s.Families, "embed")
			}
		}
	}
}

// embedVarName finds the first var declared after a //go:embed comment.
func embedVarName(file *ast.File, pos token.Pos) string {
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		if gd.Pos() <= pos || gd.Pos()-pos >= 200 {
			continue
		}
		if len(gd.Specs) == 0 {
			continue
		}
		if vs, ok := gd.Specs[0].(*ast.ValueSpec); ok && len(vs.Names) > 0 {
			return vs.Names[0].Name
		}
	}

	return ""
}

func contains(list []string, v string) bool {
	return slices.Contains(list, v)
}

// sortedSites returns sites ordered by ID with deterministic inner slices.
func (e *extractor) sortedSites() []*Site {
	out := make([]*Site, 0, len(e.sites))
	for _, s := range e.sites {
		sort.Strings(s.Locations)
		sort.Strings(s.Calls)
		sort.Strings(s.Families)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}
