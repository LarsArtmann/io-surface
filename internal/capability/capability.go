// Package capability is the single source of truth for the Capslock
// vocabulary the unified manifest understands (io-surface/v1.1; the closed
// ten since v1.3).
//
// io-surface answers where I/O sits (per-call-site, hermetic); Capslock
// answers what is reachable (SSA callgraph, transitive, noisy). This package
// maps the second onto the first: every capability name absorb may store, and
// the site families that legitimately produce it. It exists so docs, rules and
// the CUE schema all derive from ONE table — a second mapping is a split
// brain.
//
// v1.3 (ROADMAP Q5 first-class decision) stores the runtime-power set at
// PACKAGE level only — as reachability facts in capabilities.cue, never as
// per-site disciplines (transitively they reach into every package, so as
// site disciplines they would be pure noise). RuntimePower reports which
// half of the closed ten a name belongs to; the advisory shadow tier builds
// on it. Extending either half is a schema version bump (taxonomy freeze),
// same as adding a family.
//
// The closed enum is duplicated as #CapabilityName in internal/scaffold/
// schema.cue (the standard copy). Keep the two in sync; the scaffold test
// round-trips a manifest through both.
package capability

import (
	"slices"
	"sort"
	"strings"
)

// Name is a Capslock capability io-surface stores. The set is closed.
type Name string

// The mapped capability names (v1.1) and the runtime-power set (v1.3,
// ROADMAP Q5): exact Capslock names from the 2026-11-06 blind-spot run.
// MODIFY_SYSTEM_STATE/* sub-variants normalize to the base name at absorb.
const (
	Files   Name = "FILES"
	Network Name = "NETWORK"
	Exec    Name = "EXEC"

	Reflect           Name = "REFLECT"
	Runtime           Name = "RUNTIME"
	SystemCalls       Name = "SYSTEM_CALLS"
	UnsafePointer     Name = "UNSAFE_POINTER"
	OperatingSystem   Name = "OPERATING_SYSTEM"
	ReadSystemState   Name = "READ_SYSTEM_STATE"
	ModifySystemState Name = "MODIFY_SYSTEM_STATE"
)

// Names is the closed enum (ten, v1.3), sorted for deterministic iteration.
var Names = []Name{
	Exec, Files, Network,
	ModifySystemState, OperatingSystem, ReadSystemState, Reflect, Runtime, SystemCalls, UnsafePointer,
}

// runtimePower is the v1.3 half of the closed ten: reachability facts stored
// at package level, outside the site-family mapping.
var runtimePower = map[Name]bool{
	Reflect:           true,
	Runtime:           true,
	SystemCalls:       true,
	UnsafePointer:     true,
	OperatingSystem:   true,
	ReadSystemState:   true,
	ModifySystemState: true,
}

// RuntimePower reports whether name belongs to the runtime-power half of the
// closed ten (v1.3). False for the three site-mapped caps.
func RuntimePower(name Name) bool {
	return runtimePower[name]
}

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
		slices.Sort(set.Direct)
		slices.Sort(set.Deep)
		out = append(out, set)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pkg < out[j].Pkg })

	return Result{Sets: out, Dropped: dropped}
}
