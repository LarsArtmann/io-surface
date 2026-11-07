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
// A forced refresh (--force) rewrites facts.cue only; the other three files
// survive once they exist.
//
// Green-by-construction: the snapshot IS the current tree, so the first
// `check` passes and can only fail on I/O added afterwards.
package scaffold

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/larsartmann/go-atomic-write"

	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
)

//go:embed schema.cue
var schemaCUE []byte

// Dir is the conventional manifest location inside a repository.
// Dir is the conventional manifest location — aliased from the manifest
// package, which owns the fact (one constant, one home).
const Dir = manifest.Dir

// Files returns the scaffold file set as path->content (paths relative to the
// manifest dir). root names the repository the snapshot is taken from and only
// feeds the audit provenance line.
func Files(root string, sites []*extract.Site, date string) map[string][]byte {
	return map[string][]byte{
		"schema.cue":       schemaCUE,
		"facts.cue":        factsCUE(sites, date, auditCommit(root)),
		"claims.cue":       claimsCUE(),
		"baseline.cue":     baselineCUE(),
		"capabilities.cue": capabilitiesCUE(),
	}
}

// Write writes the scaffold for root into <root>/<Dir>. On a manifest dir
// without schema.cue it writes the full four-file set (or errors without
// force). With force it refreshes ONLY facts.cue: claims.cue, baseline.cue
// and schema.cue are never overwritten once present, because they carry human
// judgment and the standard schema — regeneration is for the mechanical file.
func Write(root string, sites []*extract.Site, date string, force bool) (string, error) {
	dir := filepath.Join(root, Dir)
	if _, err := os.Stat(filepath.Join(dir, "schema.cue")); err == nil && !force {
		return dir, fmt.Errorf("%w: %s (use --force to overwrite facts.cue)", ErrExists, dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dir, err
	}
	files := Files(root, sites, date)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		if name != "facts.cue" {
			if _, err := os.Stat(path); err == nil {
				continue // judgment/standard file already present: never clobber
			}
		}
		if err := atomicwrite.WriteWithPerm(path, files[name], 0o644); err != nil {
			return dir, err
		}
	}

	return dir, nil
}

// factsCUE renders the committed extraction snapshot.
func factsCUE(sites []*extract.Site, date, commit string) []byte {
	var b strings.Builder
	b.WriteString(`// facts.cue — extracted from code by ` + "`io-surface init`" + `.
//
// Mechanical file: regenerate with ` + "`io-surface init --force`" + ` after an
// I/O-adding change; never hand-edit locations. Judgment lives in claims.cue.
package io_surface

`)
	fmt.Fprintf(&b, "audit: {\n\tcommit: %q\n\tdate:   %q\n\tclaims: %d\n\tmethod: %q\n}\n\n",
		commit, date, countLocations(sites), "io-surface extract (AST, stdlib-only)")

	b.WriteString("enforcement: \"gate\"\n\n")

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

// capabilitiesCUE renders the empty capability-facts file. It is the one file
// `absorb` owns: reachability facts are mechanical (engine-derived), so they
// live apart from the human-owned claims and baseline.
func capabilitiesCUE() []byte {
	return []byte(`// capabilities.cue — reachability facts observed by an external engine.
//
// Written ONLY by ` + "`io-surface absorb`" + ` (from pinned Capslock output);
// never hand-edit locations, and never let another tool touch this file.
// Advisory tier: capability findings never fail the gate by themselves.
package io_surface

capabilities: []
`)
}

// auditCommit resolves the commit the facts were extracted at: the short
// HEAD hash of the repo containing root, suffixed -dirty when the worktree
// carries uncommitted changes (the snapshot then spans unextracted edits).
// Outside a git repo — a tarball, a sandbox checkout without .git — it falls
// back to the literal "working-tree" so scaffolded manifests stay valid.
func auditCommit(root string) string {
	head, err := git(root, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "working-tree"
	}
	dirty, err := git(root, "status", "--porcelain")
	if err != nil {
		return head
	}
	if strings.TrimSpace(dirty) != "" {
		return head + "-dirty"
	}

	return head
}

// git runs one git invocation in dir and returns trimmed stdout.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
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
