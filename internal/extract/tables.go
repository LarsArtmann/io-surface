package extract

// Detection tables: the single place the extractor "knows" anything about
// specific packages, calls, or wrapper idioms (AGENTS.md "Where detection
// rules live"). Pure data on purpose — teaching the extractor a new call is
// editing these maps plus a testdata fixture row, never control flow.

import (
	"os"
	"path/filepath"
)

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
	e := &extractor{root: root, sites: map[string]*Site{}, seams: seams, seamDecls: map[string]map[string]string{}}
	dirs := opts.Dirs
	if len(dirs) == 0 {
		dirs = []string{"cmd", "internal"}
	}
	// Two passes: pass A indexes which dirs declare seam idents (the
	// caller may sit in a file walked BEFORE the declaring file, so the
	// index must be complete before any scanFunc resolves). Pass B scans.
	for _, dir := range dirs {
		path := filepath.Join(root, dir)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		if err = e.walk(path, opts, true); err != nil {
			return nil, err
		}
	}
	for _, dir := range dirs {
		path := filepath.Join(root, dir)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		if err = e.walk(path, opts, false); err != nil {
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
