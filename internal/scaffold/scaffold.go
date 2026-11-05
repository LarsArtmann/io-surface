// Package scaffold writes a starter io-surface/v1 package into a repo, so
// adopting the gate costs one command: `io-surface init`.
//
// What it writes (in <root>/docs/architecture/io-surface/):
//
//	schema.cue    the standard (embedded copy; do not edit per repo)
//	facts.cue     every extracted site, as a committed snapshot
//	claims.cue    an empty, documented place for human judgment
//	baseline.cue  empty; add entries only for findings you refuse to declare
//
// Green-by-construction: the snapshot IS the current tree, so the first
// `check` passes and can only fail on I/O added afterwards.
package scaffold

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/larsartmann/io-surface/internal/extract"
)

//go:embed schema.cue
var schemaCUE []byte

// Dir is the conventional manifest location inside a repository.
const Dir = "docs/architecture/io-surface"

// Files returns the scaffold file set as path->content (paths relative to the
// manifest dir).
func Files(sites []*extract.Site, date string) map[string][]byte {
	return map[string][]byte{
		"schema.cue":   schemaCUE,
		"facts.cue":    factsCUE(sites, date),
		"claims.cue":   claimsCUE(),
		"baseline.cue": baselineCUE(),
	}
}

// Write writes the scaffold for root into <root>/<Dir>, unless it already
// exists (in which case it returns ErrExists and touches nothing).
func Write(root string, sites []*extract.Site, date string, force bool) (string, error) {
	dir := filepath.Join(root, Dir)
	if _, err := os.Stat(filepath.Join(dir, "schema.cue")); err == nil && !force {
		return dir, fmt.Errorf("%w: %s (use --force to overwrite facts.cue)", ErrExists, dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dir, err
	}
	files := Files(sites, date)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), files[name], 0o644); err != nil {
			return dir, err
		}
	}

	return dir, nil
}

// factsCUE renders the committed extraction snapshot.
func factsCUE(sites []*extract.Site, date string) []byte {
	var b strings.Builder
	b.WriteString(`// facts.cue — extracted from code by ` + "`io-surface init`" + `.
//
// Mechanical file: regenerate with ` + "`io-surface init --force`" + ` after an
// I/O-adding change; never hand-edit locations. Judgment lives in claims.cue.
package io_surface

`)
	fmt.Fprintf(&b, "audit: {\n\tcommit: %q\n\tdate:   %q\n\tclaims: %d\n\tmethod: %q\n}\n\n",
		"working-tree", date, countLocations(sites), "io-surface extract (AST, stdlib-only)")

	b.WriteString(`scope: {
	in: [
		"file system (reads, writes, embeds)",
		"databases (SQLite/engine handles)",
		"network listeners",
		"outbound network",
		"stream copies",
		"entropy",
		"subprocess / local IPC",
		"environment reads",
	]
	out: [
		"timers and tickers (scheduling)",
		"memory pooling",
		"kernel-internal I/O (epoll, page cache)",
	]
}

sites: {
`)
	for _, s := range sites {
		family := s.Family()
		if family == "" {
			continue // weak-tier token with no family hint; advisory only
		}
		fmt.Fprintf(&b, "\t%q: {\n", s.ID)
		b.WriteString("\t\tlocation: [\n")
		for _, loc := range s.Locations {
			fmt.Fprintf(&b, "\t\t\t%q,\n", loc)
		}
		b.WriteString("\t\t]\n")
		fmt.Fprintf(&b, "\t\tfamily: %q\n", family)
		b.WriteString("\t\tfacts: {\n")
		if len(s.Calls) > 0 {
			quoted := make([]string, len(s.Calls))
			for i, c := range s.Calls {
				quoted[i] = fmt.Sprintf("%q", c)
			}
			fmt.Fprintf(&b, "\t\t\tcall: [%s]\n", strings.Join(quoted, ", "))
		}
		if s.Testinfra {
			b.WriteString("\t\t\ttestinfra: true\n")
		}
		b.WriteString("\t\t}\n\t}\n")
	}
	b.WriteString("}\n")

	return []byte(b.String())
}

// claimsCUE renders the empty, documented judgment file.
func claimsCUE() []byte {
	return []byte(`// claims.cue — human judgment. Tooling never overwrites this file.
//
// Both files contribute to the same ` + "`sites`" + ` value (CUE unification), so an entry
// here only needs the claim fields:
//
//	sites: "internal/server.AllowlistManager.Save": claims: {
//		resource:   "temp_allowlist_path JSON"
//		trigger:    "every grant/revoke/expiry sweep/shutdown"
//		discipline: ["atomic-write"]
//		rationale:  "why this posture is right, in prose"
//	}
//
// A decided exception is data, not a comment:
//
//	sites: "internal/dns.timedCacheWrite": claims: {
//		exception: "no-fsync"
//		rationale: "the cache is re-fetched unconditionally after a crash, so a torn file is never parsed"
//	}
package io_surface
`)
}

// baselineCUE renders the empty baseline file.
func baselineCUE() []byte {
	return []byte(`// baseline.cue — grandfathered undeclared sites (the adoption escape hatch).
//
// The baseline only shrinks: every entry needs a note and a date, and a new
// entry needs a reason in the commit message. Never use it for a site you are
// about to fix "later" without a date.
package io_surface

baseline: []
`)
}

func countLocations(sites []*extract.Site) int {
	seen := map[string]bool{}
	for _, s := range sites {
		for _, loc := range s.Locations {
			seen[loc] = true
		}
	}

	return len(seen)
}
