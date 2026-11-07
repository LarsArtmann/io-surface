package manifest

// T11 tests moved out of manifest_test.go for the 650-line file gate: the
// constraint-error contract (CUE paths, phantom list.Concat entries).

import (
	"errors"
	"strings"
	"testing"
)

// TestLoad_BadDisciplineNamesCUEPath pins the T11 contract: a per-site
// validation failure reports the CUE path (sites."X".field), not just a
// bare disjunction count.
func TestLoad_BadDisciplineNamesCUEPath(t *testing.T) {
	closedSchema := `package io_surface

sites: {[string]: {
	location: [...string]
	family:   string
	claims: {resource: string, trigger: string, discipline: [...#Discipline]}
}}

#Discipline: "atomic-write" | "fsync" | "no-fsync"
`
	dir := writeManifest(t, map[string]string{
		"schema.cue": closedSchema,
		"facts.cue": `package io_surface

sites: "a.B": {location: ["a.go:1"], family: "file-read", facts: {call: []}}
`,
		"claims.cue": `package io_surface

sites: "a.B": claims: {resource: "config file", trigger: "on startup", discipline: ["no-such-discipline"]}
`,
	})
	_, err := Load(dir)
	if err == nil {
		t.Fatal("load succeeded on a bad discipline — constraint enforcement lost")
	}
	if !strings.Contains(err.Error(), `sites."a.B".claims.discipline`) {
		t.Errorf("error does not name the CUE path (T11): %v", err)
	}
}

// TestLoad_ConcatViolations_PhantomProof pins the dnsblockd bug found by
// T11: a violations value computed by list.Concat over empty comprehensions
// exposed its operand sub-lists to the entry recovery, which decoded them
// into contentless "violations" that failed the gate with opaque ` - .`
// entries. Contentless entries are decode artifacts, never findings.
func TestLoad_ConcatViolations_PhantomProof(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "a.B": {location: ["a.go:1"], family: "file-read", facts: {call: []}}
`,
		"claims.cue": `package io_surface

sites: "a.B": claims: {resource: "config file", trigger: "on startup"}
`,
		"disciplines.cue": `package io_surface

import "list"

violations: list.Concat([
	[for siteID, site in sites if site.claims.resource == "" {
		id:  siteID
		msg: "no claims.resource",
	}],
])
`,
	})
	if _, err := Load(dir); err != nil {
		t.Fatalf("phantom violations failed the gate (T11 regression): %v", err)
	}
}

// TestLoad_ConcatViolations_RealEntryFailsWithPaths pins the positive half of
// the phantom guard: a genuinely firing concat rule reports id, msg, AND the
// CUE element path.
func TestLoad_ConcatViolations_RealEntryFailsWithPaths(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "a.B": {location: ["a.go:1"], family: "file-read", facts: {call: []}}
`,
		"claims.cue": `package io_surface

sites: "a.B": claims: {resource: "", trigger: "on startup"}
`,
		"disciplines.cue": `package io_surface

import "list"

violations: list.Concat([
	[for siteID, site in sites if site.claims.resource == "" {
		id:  siteID
		msg: "no claims.resource",
	}],
])
`,
	})
	_, err := Load(dir)
	if !errors.Is(err, ErrConstraints) {
		t.Fatalf("want ErrConstraints, got %v", err)
	}
	for _, want := range []string{"a.B", "no claims.resource", "violations"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("violation report lost %q (T11 path contract): %v", want, err)
		}
	}
}
