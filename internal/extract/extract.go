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

// familyPrecedence orders the families by "how much this tells you about the
// site". A function that stats a file and then renames it is a file WRITER, not
// a reader; one that listens and also dials is a listener. The first family in
// this list that a site touches becomes its dominant family.
var familyPrecedence = []string{
	"file-write",
	"database",
	"listener",
	"outbound",
	"ipc-subprocess",
	"embed",
	"entropy",
	"stream-copy",
	"env",
	"file-read",
}

// Family returns the dominant family hint for the site (the highest-precedence
// family it touches), or "" when the extraction could not classify it.
func (s *Site) Family() string {
	for _, fam := range familyPrecedence {
		if contains(s.Families, fam) {
			return fam
		}
	}
	if len(s.Families) > 0 {
		return s.Families[0]
	}

	return ""
}

// Options controls an extraction run.
type Options struct {
	// IncludeTests parses *_test.go files too. Test files are never manifest
	// sites; this exists for counting/reporting only.
	IncludeTests bool
	// Dirs limits the walk to these subdirectories (default: cmd, internal).
	Dirs []string
	// Ignore skips files whose repo-relative slash path — or base name —
	// matches any glob (filepath.Match syntax: `*_templ.go`, `zz_generated*`).
	// Generated code is not authored I/O; declaring it would be inventory
	// noise the generator rewrites anyway.
	Ignore []string
	// Seams extends the built-in wrapper vocabulary with manifest-derived
	// seams: ident → family learned from declared sites' facts.via values.
	// check.Run builds it from the manifest, so a repo teaches the extractor
	// its own wrapper names (dnsblockd's writeMapping) without an extractor
	// edit; init keeps the built-in table so first scaffolds still see the
	// common wrappers.
	Seams map[string]string
}

// Extract walks root and returns sites ordered by ID.
func Extract(root string, opts Options) ([]*Site, error) {
	seams := make(map[string]string, len(seamIdent)+len(opts.Seams))
	for ident, fam := range seamIdent {
		seams[ident] = fam
	}
	for ident, fam := range opts.Seams {
		seams[ident] = fam
	}
	e := &extractor{root: root, sites: map[string]*Site{}, seams: seams}
	dirs := opts.Dirs
	if len(dirs) == 0 {
		dirs = []string{"cmd", "internal"}
	}
	for _, dir := range dirs {
		path := filepath.Join(root, dir)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		err = e.walk(path, opts)
		if err != nil {
			return nil, err
		}
	}

	return e.sortedSites(), nil
}

// pkgOf maps an import path to the canonical package key we classify by.
var pkgOf = map[string]string{
	"os":                         "os",
	"io":                         "io",
	"bufio":                      "bufio",
	"net":                        "net",
	"crypto":                     "",
	"net/http":                   "http",
	"net/http/httputil":          "httputil",
	"crypto/rand":                "rand",
	"crypto/rsa":                 "rsa",
	"crypto/x509":                "x509",
	"crypto/tls":                 "tls",
	"database/sql":               "sql",
	"os/exec":                    "exec",
	"path/filepath":              "filepath",
	"syscall":                    "syscall",
	"github.com/quic-go/quic-go": "quic",
	"github.com/quic-go/http3":   "http3",
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp": "otlptracehttp",
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp":     "otlploghttp",
	"github.com/coreos/go-oidc/v3/oidc":                               "oidc",
	"github.com/larsartmann/go-cqrs-lite/system/v4":                   "cqrs-system",
}

// importSubstring matches cqrs engine imports by path fragment (several
// sub-modules: metaengine, metaengine/sqliteengine, projectionhost...).
var importSubstring = map[string]string{
	"go-cqrs-lite/metaengine": "cqrs-engine",
	"sqliteengine":            "cqrs-engine",
}

// callFamily classifies a package-qualified call; "" = not watched.
var callFamily = map[string]map[string]string{
	"os": {
		"Open": "file-read", "ReadFile": "file-read", "Stat": "file-read", "Lstat": "file-read",
		"ReadDir": "file-read", "TempDir": "file-read",
		"Getenv": "env", "LookupEnv": "env", "Setenv": "env", "Unsetenv": "env", "Environ": "env",
		"Create": "file-write", "WriteFile": "file-write", "OpenFile": "file-write", "CreateTemp": "file-write",
		"Truncate": "file-write", "Rename": "file-write", "Remove": "file-write", "RemoveAll": "file-write",
		"Mkdir": "file-write", "MkdirAll": "file-write", "Sync": "file-write",
	},
	"io": {
		"Copy": "stream-copy", "CopyBuffer": "stream-copy", "CopyN": "stream-copy",
		"ReadAll": "stream-copy", "ReadFull": "stream-copy", "TeeReader": "stream-copy",
		"LimitReader": "stream-copy", "Pipe": "stream-copy",
	},
	"net": {
		"Listen": "listener", "ListenPacket": "listener", "ListenUDP": "listener", "ListenTCP": "listener",
		"ListenUnix": "listener", "ListenUnixgram": "listener",
		"Dial": "outbound", "DialContext": "outbound", "DialTimeout": "outbound",
		"DialTCP": "outbound", "DialUDP": "outbound",
		"DialUnix":   "ipc-subprocess",
		"LookupHost": "outbound", "LookupIP": "outbound", "Resolver": "outbound",
	},
	"http": {
		"NewRequest": "outbound", "NewRequestWithContext": "outbound",
		"Get": "outbound", "Post": "outbound", "Head": "outbound",
		"MaxBytesReader": "stream-copy", "Transport": "outbound",
		"ServeContent": "stream-copy", "ServeFile": "stream-copy",
	},
	"httputil":      {"ReverseProxy": "outbound", "NewSingleHostReverseProxy": "outbound"},
	"rand":          {"Read": "entropy", "Int": "entropy", "Reader": "entropy", "Text": "entropy"},
	"rsa":           {"GenerateKey": "entropy"},
	"x509":          {"CreateCertificate": "entropy"},
	"tls":           {"Listen": "listener", "Dial": "outbound", "LoadX509KeyPair": "file-read", "Server": "listener"},
	"sql":           {"Open": "database", "OpenDB": "database"},
	"filepath":      {"Walk": "file-read", "WalkDir": "file-read", "Glob": "file-read"},
	"exec":          {"Command": "ipc-subprocess", "CommandContext": "ipc-subprocess", "LookPath": "ipc-subprocess"},
	"syscall":       {"SetsockoptInt": "listener", "RawConn": "listener", "Mmap": "file-read", "Munmap": "file-read"},
	"bufio":         {"NewReader": "stream-copy", "NewReaderSize": "stream-copy", "NewWriter": "stream-copy", "NewWriterSize": "stream-copy", "NewReadWriter": "stream-copy"},
	"quic":          {"Listen": "listener", "Dial": "outbound", "Accept": "listener", "AcceptStream": "listener"},
	"http3":         {"Server": "listener"},
	"otlptracehttp": {"New": "outbound"},
	"otlploghttp":   {"New": "outbound"},
	"oidc":          {"NewProvider": "outbound"},
	"cqrs-system":   {"New": "database"},
	"cqrs-engine":   {"New": "database", "Open": "database"},
}

// methodToken classifies calls on unresolvable receivers by method name. ""
// means the token is watched but carries no family (it still marks the symbol
// as an I/O site; the manifest's family is the claim).
var methodToken = map[string]string{
	"ActivateAndServe": "listener", "ListenAndServe": "listener", "ListenAndServeTLS": "listener",
	"Serve": "listener", "ServeTLS": "listener", "Accept": "listener", "AcceptStream": "listener",
	"AcceptUnix": "listener", "ReadFromUDP": "listener",
	"Dial": "outbound", "DialContext": "outbound", "RoundTrip": "outbound",
	"LookupHost": "outbound", "FinishLogin": "outbound", "WriteToUDP": "outbound",
	// .Do is ambiguous by design: every http.Client.Do, sql.Tx.Do-style
	// helper and task-runner shares the name, so the token records the site
	// but claims no family (the manifest's family is the claim).
	"Do":          "",
	"ExecContext": "database", "QueryContext": "database", "QueryRowContext": "database",
	"BeginTx": "database", "PingContext": "database", "PrepareContext": "database",
	"AppendBatch": "database", "SaveQuery": "database", "EnqueueOp": "database", "EnqueueCommand": "database",
	"ReadAll": "stream-copy",
}

// seamIdent: same-module wrapper identifiers — the raw call lives in the seam,
// callers get via=seam attribution instead of a raw-family hit.
var seamIdent = map[string]string{
	"atomicWriteFile":         "file-write",
	"writeCacheFileAtomic":    "file-write",
	"writeCacheFile":          "file-write",
	"EnqueueOp":               "database",
	"EnqueueCommand":          "database",
	"AppendBatch":             "database",
	"SaveQuery":               "database",
	"FetchBlocklistWithLimit": "outbound",
	"FetchBlocklistWithRetry": "outbound",
}

type extractor struct {
	root  string
	fset  token.FileSet
	sites map[string]*Site
	seams map[string]string
}

func (e *extractor) walk(dir string, opts Options) error {
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
	for _, imp := range file.Imports {
		ipath := strings.Trim(imp.Path.Value, `"`)
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

						return true
					}
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
}

func (e *extractor) record(id string, n ast.Node, testinfra bool, call, family, confidence, via string) {
	s := e.site(id, testinfra)
	pos := e.fset.Position(n.Pos())
	rel := pos.Filename
	if r, err := filepath.Rel(e.root, pos.Filename); err == nil {
		rel = filepath.ToSlash(r)
	}
	loc := rel + ":" + strconv.Itoa(pos.Line)
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
