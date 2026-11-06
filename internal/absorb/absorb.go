// Package absorb turns pinned Capslock JSON output into the manifest's
// capability facts (io-surface/v1.1).
//
// Absorb is the ONLY writer of capabilities.cue, and it never touches
// facts.cue, claims.cue, baseline.cue or schema.cue: reachability facts are
// mechanical (engine-derived), human judgment is not. The hermetic gate never
// runs Capslock — absorb is where the external engine's output enters, and
// its drift report is advisory by contract (capability findings never exit 1;
// promotion to a gate needs the manifest's enforcement policy, not this
// command).
//
// The engine output is trusted only as far as its schema: unknown capability
// types fail the run, unmapped capability NAMES are counted and reported
// loudly (they mean the vocabulary moved — probably a schema-version bump —
// never silently dropped).
package absorb

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/larsartmann/io-surface/internal/capability"
	"github.com/larsartmann/io-surface/internal/manifest"
)

// RuleCapabilityDrift is the advisory rule id for committed capabilities that
// disagree with a fresh engine run. It is emitted by `absorb --check`, not by
// the hermetic `check` command — the gate never reads engine artifacts.
const RuleCapabilityDrift = "io-surface/capability-drift"

// caplockEntry is the slice of the Capslock report absorb understands. The
// full report carries call paths and sites; absorb intentionally ignores them
// and stores per-package sets, not edges.
type caplockEntry struct {
	CapabilityName string `json:"capabilityName"`
	CapabilityType string `json:"capabilityType"`
	PackageDir     string `json:"packageDir"`
}

type caplockReport struct {
	CapabilityInfo []caplockEntry `json:"capabilityInfo"`
}

// Audit is the provenance recorded alongside the capability sets.
type Audit struct {
	Engine  string
	Version string
	Commit  string
	Date    string
}

// Options tunes one absorb run. Empty Audit fields are filled with defaults
// (engine "capslock", today's UTC date, git HEAD of root).
type Options struct {
	// ModulePath overrides the module path parsed from <root>/go.mod.
	ModulePath string
	Audit      Audit
}

// Result is one absorb run's computed state.
type Result struct {
	Sets    []capability.Set
	Dropped map[string]int
	Pkgs    int
}

var moduleLine = regexp.MustCompile(`(?m)^module\s+(\S+)`)

// ModulePath parses the module directive from <root>/go.mod. Hermetic: one
// file read, no tool invocations.
func ModulePath(root string) (string, error) {
	data, err := os.ReadFile(goModPath(root))
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	m := moduleLine.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("no module directive in %s", goModPath(root))
	}

	return string(m[1]), nil
}

func goModPath(root string) string {
	if root == "" {
		return "go.mod"
	}

	return root + "/go.mod"
}

// Run decodes a Capslock JSON report and folds it into per-package capability
// sets for the repo at root.
func Run(root string, report []byte, opts Options) (*Result, error) {
	var raw caplockReport
	if err := json.Unmarshal(report, &raw); err != nil {
		return nil, fmt.Errorf("decoding Capslock report: %w", err)
	}

	module := opts.ModulePath
	if module == "" {
		var err error
		if module, err = ModulePath(root); err != nil {
			return nil, err
		}
	}

	edges := make([]capability.Edge, 0, len(raw.CapabilityInfo))
	for _, e := range raw.CapabilityInfo {
		depth, err := normalizeDepth(e.CapabilityType)
		if err != nil {
			return nil, err
		}
		edges = append(edges, capability.Edge{
			PackageDir:     e.PackageDir,
			CapabilityName: e.CapabilityName,
			Depth:          depth,
		})
	}

	folded := capability.Sets(module, edges)
	res := &Result{Sets: folded.Sets, Dropped: folded.Dropped, Pkgs: len(folded.Sets)}

	return res, nil
}

func normalizeDepth(t string) (string, error) {
	switch t {
	case "CAPABILITY_TYPE_DIRECT":
		return "DIRECT", nil
	case "CAPABILITY_TYPE_TRANSITIVE":
		return "TRANSITIVE", nil
	case "":
		return "", fmt.Errorf("capability entry without a type")
	default:
		return "", fmt.Errorf("unknown capability type %q (engine newer than this schema?)", t)
	}
}

// ResolveAudit fills the audit provenance: git HEAD of root (± -dirty) and
// today's UTC date for whatever is missing.
func ResolveAudit(root string, audit Audit) Audit {
	if audit.Engine == "" {
		audit.Engine = "capslock"
	}
	if audit.Date == "" {
		audit.Date = time.Now().UTC().Format("2006-01-02")
	}
	if audit.Commit == "" {
		audit.Commit = gitHead(root)
	}

	return audit
}

// gitHead mirrors scaffold's audit provenance: short HEAD, -dirty when the
// worktree carries changes, literal fallback outside a git repo.
func gitHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "working-tree"
	}
	head := strings.TrimSpace(string(out))
	status := exec.Command("git", "status", "--porcelain")
	status.Dir = root
	if dirty, err := status.Output(); err == nil && strings.TrimSpace(string(dirty)) != "" {
		return head + "-dirty"
	}

	return head
}

// CUE renders the capabilities.cue body for a run. This is the byte contract
// `absorb --write` puts on disk; the file carries ONLY the capabilities
// section and its provenance.
func CUE(r *Result, audit Audit) []byte {
	var b strings.Builder
	b.WriteString(`// capabilities.cue — reachability facts observed by an external engine.
//
// Written ONLY by ` + "`io-surface absorb`" + ` (from pinned Capslock output);
// never hand-edit locations, and never let another tool touch this file.
// Advisory tier: capability findings never fail the gate by themselves.
package io_surface

capabilityAudit: {
	engine:  ` + quote(audit.Engine) + `
	version: ` + quote(audit.Version) + `
	commit:  ` + quote(audit.Commit) + `
	date:    ` + quote(audit.Date) + `
}

capabilities: [
`)
	for _, s := range r.Sets {
		fmt.Fprintf(&b, "\t{pkg: %s, direct: [%s], deep: [%s]},\n",
			quote(s.Pkg), names(s.Direct), names(s.Deep))
	}
	b.WriteString("]\n")

	return []byte(b.String())
}

func quote(s string) string { return "\"" + s + "\"" }

func names(ns []capability.Name) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = quote(string(n))
	}

	return strings.Join(parts, ", ")
}

// Drift compares the committed manifest's capabilities against a fresh run
// and returns human-readable advisory lines (empty = in sync). Both
// directions count: a capability that disappeared and one that appeared are
// equally stale facts.
func Drift(m *manifest.Manifest, r *Result) []string {
	var out []string
	committed := map[string]manifest.Capability{}
	for _, c := range m.Capabilities {
		committed[c.Pkg] = c
	}
	computed := map[string]capability.Set{}
	for _, s := range r.Sets {
		computed[s.Pkg] = s
	}

	pkgs := map[string]bool{}
	for pkg := range committed {
		pkgs[pkg] = true
	}
	for pkg := range computed {
		pkgs[pkg] = true
	}
	sorted := make([]string, 0, len(pkgs))
	for pkg := range pkgs {
		sorted = append(sorted, pkg)
	}
	sort.Strings(sorted)

	for _, pkg := range sorted {
		have, ok := committed[pkg]
		if !ok {
			out = append(out, fmt.Sprintf("%s: capability facts missing (fresh run reaches %s)", pkg, summarize(computed[pkg])))

			continue
		}
		want, ok := computed[pkg]
		if !ok {
			out = append(out, fmt.Sprintf("%s: committed capability facts no longer observed (committed %s)", pkg, setSummary(have)))

			continue
		}
		if !sameSet(have.Direct, want.Direct) || !sameSet(have.Deep, want.Deep) {
			out = append(out, fmt.Sprintf("%s: committed %s ≠ fresh %s", pkg, setSummary(have), summarize(want)))
		}
	}

	return out
}

func summarize(s capability.Set) string {
	if len(s.Deep) == 0 {
		return "nothing mapped"
	}

	return "deep [" + joinNames(s.Deep) + "]"
}

func setSummary(c manifest.Capability) string {
	return "deep [" + strings.Join(c.Deep, ", ") + "]"
}

func joinNames(ns []capability.Name) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = string(n)
	}

	return strings.Join(parts, ", ")
}

func sameSet(a []string, b []capability.Name) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != string(b[i]) {
			return false
		}
	}

	return true
}
