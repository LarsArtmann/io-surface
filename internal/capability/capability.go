// Package capability is the single source of truth for the Capslock
// vocabulary the unified manifest understands (io-surface/v1.1).
//
// io-surface answers where I/O sits (per-call-site, hermetic); Capslock
// answers what is reachable (SSA callgraph, transitive, noisy). This package
// maps the second onto the first: every capability name absorb may store, and
// the site families that legitimately produce it. It exists so docs, rules and
// the CUE schema all derive from ONE table — a second mapping is a split
// brain.
//
// Capslock capabilities outside this table (CGO, UNSAFE_POINTER, REFLECT,
// RUNTIME, SYSTEM_CALLS, ARBITRARY_EXECUTION, READ/MODIFY_SYSTEM_STATE,
// OPERATING_SYSTEM, UNANALYZED, ...) are out-of-scope by decision Q5
// (ROADMAP): transitively they reach into every package, so as per-site
// disciplines they would be pure noise. Extending the table is a schema
// version bump (taxonomy freeze), same as adding a family.
//
// The closed enum is duplicated as #CapabilityName in internal/scaffold/
// schema.cue (the standard copy). Keep the two in sync; the scaffold test
// round-trips a manifest through both.
package capability

import (
	"fmt"
	"sort"
	"strings"
)

// Name is a Capslock capability io-surface stores. The set is closed.
type Name string

// The mapped capability names (v1.1).
const (
	Files   Name = "FILES"
	Network Name = "NETWORK"
	Exec    Name = "EXEC"
)

// Names is the closed enum, sorted for deterministic iteration.
var Names = []Name{Exec, Files, Network}

// Valid reports whether name is part of the closed enum.
func Valid(name string) bool {
	for _, n := range Names {
		if string(n) == name {
			return true
		}
	}

	return false
}

// familiesFor maps a capability to the site families that legitimately
// produce it. This is THE mapping; shadow-capability derives from it.
var familiesFor = map[Name][]string{
	Files:   {"file-read", "file-write", "embed", "database"},
	Network: {"listener", "outbound"},
	Exec:    {"ipc-subprocess"},
}

// FamiliesFor returns the mapped families for one capability.
func FamiliesFor(name Name) []string {
	out := append([]string(nil), familiesFor[name]...)
	sort.Strings(out)

	return out
}

// Families returns the deduplicated union of families for a capability set,
// sorted. An unknown name is skipped by Valid's contract upstream; this
// function treats unknown names as an error to keep call sites honest.
func Families(names []Name) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, n := range names {
		fs, ok := familiesFor[n]
		if !ok {
			return nil, fmt.Errorf("capability %q is not in the closed enum", n)
		}
		for _, f := range fs {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	sort.Strings(out)

	return out, nil
}

// Edge is one Capslock observation: packageDir reached capabilityName with
// the given analysis depth ("DIRECT" or "TRANSITIVE").
type Edge struct {
	PackageDir     string
	CapabilityName string
	Depth          string
}

// Set is the per-package capability fact stored in the manifest.
// Deep always contains Direct: deep is reachability including dependencies,
// direct is reachability through the package's own code.
type Set struct {
	Pkg    string
	Direct []Name
	Deep   []Name
}

// Result is the outcome of folding Capslock edges into per-package sets.
// Dropped counts unmapped capability names per name — absorb reports them
// loudly instead of silently discarding vocabulary it does not know.
type Result struct {
	Sets    []Set
	Dropped map[string]int
}

// Sets folds first-party edges (packageDir under modulePath) into per-package
// direct/deep sets. Packages with no mapped capability are omitted. Output is
// sorted by package for deterministic writes.
func Sets(modulePath string, edges []Edge) Result {
	prefix := modulePath + "/"
	type acc struct {
		direct map[Name]bool
		deep   map[Name]bool
	}
	byPkg := map[string]*acc{}
	dropped := map[string]int{}

	add := func(pkg string) *acc {
		a := byPkg[pkg]
		if a == nil {
			a = &acc{direct: map[Name]bool{}, deep: map[Name]bool{}}
			byPkg[pkg] = a
		}

		return a
	}

	for _, e := range edges {
		if e.PackageDir == "" || !strings.HasPrefix(e.PackageDir, prefix) {
			continue // third-party package: not a fact about this repo's tree
		}
		pkg := strings.TrimPrefix(e.PackageDir, prefix)
		name := Name(e.CapabilityName)
		if !Valid(string(name)) {
			dropped[e.CapabilityName]++

			continue
		}
		a := add(pkg)
		a.deep[name] = true
		if e.Depth == "DIRECT" {
			a.direct[name] = true
		}
	}

	var out []Set
	for pkg, a := range byPkg {
		set := Set{Pkg: pkg}
		for n := range a.direct {
			set.Direct = append(set.Direct, n)
		}
		for n := range a.deep {
			set.Deep = append(set.Deep, n)
		}
		sort.Slice(set.Direct, func(i, j int) bool { return set.Direct[i] < set.Direct[j] })
		sort.Slice(set.Deep, func(i, j int) bool { return set.Deep[i] < set.Deep[j] })
		out = append(out, set)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pkg < out[j].Pkg })

	return Result{Sets: out, Dropped: dropped}
}
