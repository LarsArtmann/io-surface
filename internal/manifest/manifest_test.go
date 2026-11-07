package manifest

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeManifest scaffolds a CUE package in a temp dir from the given files.
func writeManifest(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return dir
}

const schemaBody = `package io_surface

sites: {[string]: _}
`

func TestLoad_MissingDir(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nowhere"))
	if !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("want ErrNotInitialized, got %v", err)
	}
}

func TestLoad_EmptySites(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue":  "package io_surface\n\nsites: {}\n",
	})
	_, err := Load(dir)
	if !errors.Is(err, ErrEmpty) {
		t.Fatalf("want ErrEmpty, got %v", err)
	}
}

func TestLoad_DecodesFullManifest(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "internal/store.LoadConfig": {
	location: ["internal/store/store.go:9"]
	family:   "file-read"
	facts: {
		call:        ["os.ReadFile"]
		via:         "loadConfig"
		extractable: true
		testinfra:   false
	}
}
`,
		"claims.cue": `package io_surface

sites: "internal/store.LoadConfig": claims: {
	resource:    "service config file"
	trigger:     "process start"
	discipline:  ["bounded"]
	exception:   "wrapped"
	rationale:   "read once at boot"
	criticality: "critical"
	durability:  "durable"
}
`,
		"baseline.cue": `package io_surface

baseline: [{id: "legacy.Site", note: "grandfathered", date: "2026-01-01"}]
`,
		"meta.cue": `package io_surface

audit: {commit: "abc1234", date: "2026-11-06", claims: 1, method: "test audit"}
scope: {in: ["file reads"], out: ["timers"]}
`,
	})
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	s, ok := m.Sites["internal/store.LoadConfig"]
	if !ok {
		t.Fatal("site missing from decoded manifest")
	}
	if s.Family != "file-read" || len(s.Location) != 1 || s.Location[0] != "internal/store/store.go:9" {
		t.Errorf("site facts decoded wrong: %+v", s)
	}
	if s.Facts.Call[0] != "os.ReadFile" || s.Facts.Via != "loadConfig" || !s.Facts.Extractable || s.Facts.Testinfra {
		t.Errorf("facts block decoded wrong: %+v", s.Facts)
	}
	if s.Claims.Resource != "service config file" || s.Claims.Rationale != "read once at boot" {
		t.Errorf("claims decoded wrong: %+v", s.Claims)
	}
	if s.Claims.Criticality != "critical" || s.Claims.Durability != "durable" {
		t.Errorf("v1.2 severity-modulation claims decoded wrong: %+v", s.Claims)
	}
	if len(s.Claims.Discipline) != 1 || s.Claims.Discipline[0] != "bounded" {
		t.Errorf("discipline decoded wrong: %+v", s.Claims.Discipline)
	}
	if m.Audit.Commit != "abc1234" || m.Audit.Claims != 1 {
		t.Errorf("audit decoded wrong: %+v", m.Audit)
	}
	if len(m.Scope.Out) != 1 || m.Scope.Out[0] != "timers" {
		t.Errorf("scope decoded wrong: %+v", m.Scope)
	}
	if len(m.Baseline) != 1 || m.Baseline[0].ID != "legacy.Site" {
		t.Errorf("baseline decoded wrong: %+v", m.Baseline)
	}
}

func TestLoad_OptionalSectionsDefault(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "a.B": {location: ["a.go:1"], family: "env", facts: {call: ["os.Getenv"]}}
`,
	})
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("freshly scaffolded manifest must load: %v", err)
	}
	if m.Audit != (Audit{}) || len(m.Scope.In) != 0 || len(m.Scope.Out) != 0 || m.Baseline != nil {
		t.Errorf("optional sections must default to zero, got audit=%+v scope=%+v baseline=%+v", m.Audit, m.Scope, m.Baseline)
	}
}

// TestLoad_EnforcementPolicy pins the D4 field: absent normalizes to "gate",
// "audit" decodes, an unknown value is a validation error (not a silent
// default), and the report-facing policy accessor never returns "".
func TestLoad_EnforcementPolicy(t *testing.T) {
	site := `sites: "a.B": {location: ["a.go:1"], family: "env", facts: {call: ["os.Getenv"]}}`

	t.Run("absent means gate", func(t *testing.T) {
		dir := writeManifest(t, map[string]string{
			"schema.cue": schemaBody,
			"facts.cue":  "package io_surface\n\n" + site,
		})
		m, err := Load(dir)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if m.Enforcement != "" {
			t.Errorf("raw Enforcement = %q, want \"\" (field optional)", m.Enforcement)
		}
		if got := m.EnforcementPolicy(); got != EnforcementGate {
			t.Errorf("EnforcementPolicy = %q, want %q", got, EnforcementGate)
		}
	})

	t.Run("audit decodes", func(t *testing.T) {
		dir := writeManifest(t, map[string]string{
			"schema.cue": schemaBody,
			"facts.cue":  "package io_surface\n\nenforcement: \"audit\"\n\n" + site,
		})
		m, err := Load(dir)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if got := m.EnforcementPolicy(); got != EnforcementAudit {
			t.Errorf("EnforcementPolicy = %q, want %q", got, EnforcementAudit)
		}
	})

	t.Run("unknown value is a validation error", func(t *testing.T) {
		dir := writeManifest(t, map[string]string{
			"schema.cue": schemaBody,
			"facts.cue":  "package io_surface\n\nenforcement: \"yolo\"\n\n" + site,
		})
		m, err := Load(dir)
		if err != nil {
			t.Fatalf("load must succeed (Go-side validate owns the closed set here): %v", err)
		}
		errs := m.Validate()
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "unknown policy") {
			t.Errorf("Validate() = %v, want one unknown-policy error", errs)
		}
	})
}

// TestLoad_NonEmptyViolationsFails pins the disciplines contract for the
// common shape: a computed violations list with no conflicting default
// anywhere. The load fails and the message names the actual violation.
func TestLoad_NonEmptyViolationsFails(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "a.B": {location: ["a.go:1"], family: "file-read", facts: {call: []}}
`,
		"claims.cue": `package io_surface

sites: "a.B": claims: {resource: "", trigger: ""}
`,
		"disciplines.cue": `package io_surface

violations: [{id: "a.B", msg: "planted discipline failure"}]
`,
	})
	_, err := Load(dir)
	if !errors.Is(err, ErrConstraints) {
		t.Fatalf("want ErrConstraints, got %v", err)
	}
	if !strings.Contains(err.Error(), "a.B") || !strings.Contains(err.Error(), "planted discipline failure") {
		t.Fatalf("error must name the violation entry, got: %v", err)
	}
}

// TestLoad_ConflictingViolationsSurfaceEntries reproduces the opaque failure:
// a declared empty violations list conflicts with a comprehension that yields
// an entry, and CUE reports only "incompatible list lengths". The loader must
// recover the entries from the conjuncts and show them.
func TestLoad_ConflictingViolationsSurfaceEntries(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "a.B": {location: ["a.go:1"], family: "file-read", facts: {call: []}}
`,
		"claims.cue": `package io_surface

sites: "a.B": claims: {resource: "", trigger: ""}
`,
		"disciplines.cue": `package io_surface

import "list"

violations: []
violations: list.Concat([
	[
		for id, s in sites
		if s.claims.resource == "" {
			id:  id
			msg: "no claims.resource"
		},
	],
])
`,
	})
	_, err := Load(dir)
	if !errors.Is(err, ErrConstraints) {
		t.Fatalf("want ErrConstraints, got %v", err)
	}
	// The conflict keeps the comprehension's id binding non-concrete, so the
	// loader surfaces the message body (and the id whenever CUE resolves it);
	// the opaque "incompatible list lengths" must never be the whole story.
	if !strings.Contains(err.Error(), "no claims.resource") {
		t.Fatalf("conflict error must surface the violation entries, got: %v", err)
	}
}

// validateFixture builds an in-memory manifest whose dir carries a schema.cue,
// so ID and taxonomy errors are isolated from the schema-presence check.
func validateFixture(t *testing.T, sites map[string]Site) *Manifest {
	t.Helper()
	dir := writeManifest(t, map[string]string{"schema.cue": schemaBody})

	return &Manifest{Dir: dir, Sites: sites}
}

func TestValidate_UnknownFamilyAndDiscipline(t *testing.T) {
	m := validateFixture(t, map[string]Site{
		"a.B": {
			Family: "teleport",
			Claims: Claims{Discipline: []string{"chrono-sync"}},
		},
	})
	errs := m.Validate()
	if len(errs) != 2 {
		t.Fatalf("want 2 errors, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "unknown family") || !strings.Contains(errs[1].Error(), "unknown discipline") {
		t.Fatalf("wrong errors: %v", errs)
	}
}

func TestValidate_MalformedID(t *testing.T) {
	m := validateFixture(t, map[string]Site{
		"#meta": {Family: "env"},
		"bad.":  {Family: "env"},
	})
	errs := m.Validate()
	if len(errs) != 2 {
		t.Fatalf("want 2 malformed-ID errors, got %v", errs)
	}
}

// TestValidate_DottedSubpackageID pins that dotted UNEXPORTED-TYPE forms are
// statically legal: "extractor" is a real type in package extract, so a
// lexical dotted-subpackage rejection would false-positive on the tool's own
// manifest. The never-pair detection lives in check (rule io-surface/id-form)
// where the extracted set can confirm the slash variant actually exists.
func TestValidate_DottedUnexportedTypeFormIsLegal(t *testing.T) {
	m := validateFixture(t, map[string]Site{
		"internal/extract.extractor.parseFile": {Family: "file-read"},
	})
	if errs := m.Validate(); len(errs) != 0 {
		t.Fatalf("unexported-type dotted form must validate clean, got %v", errs)
	}
}

func TestValidate_AcceptsLegalIDForms(t *testing.T) {
	m := validateFixture(t, map[string]Site{
		"internal/device.LeaseCache.Leases": {Family: "file-read"},
		"internal/server/sso.Manager.Ping":  {Family: "outbound"},
		"internal/server.openAPISpec":       {Family: "embed"},
		"cmd/app.fetch":                     {Family: "outbound"},
		"logging.sink-topology":             {Family: "ipc-subprocess"},
		"internal/config.Load#file":         {Family: "file-read"},
	})
	if errs := m.Validate(); len(errs) != 0 {
		t.Fatalf("legal ID forms must validate clean, got %v", errs)
	}
}

func TestValidate_MissingSchemaCUE(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{Dir: dir, Sites: map[string]Site{
		"a.B": {Family: "env"},
	}}
	errs := m.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "no schema.cue") {
		t.Fatalf("want missing-schema error, got %v", errs)
	}
}

func TestLoad_CapabilitiesAndAuditDecode(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "internal/store.LoadConfig": {
	location: ["internal/store/store.go:9"]
	family:   "file-read"
}
version: "io-surface/v1.1"
capabilities: [
	{
		pkg:    "internal/store"
		direct: ["FILES"]
		deep:   ["EXEC", "FILES"]
	},
]
capabilityAudit: {
	engine:  "capslock"
	version: "v0.3.4"
	commit:  "abc1234"
	date:    "2026-11-06"
}
`,
	})
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Version != "io-surface/v1.1" {
		t.Errorf("version = %q", m.Version)
	}
	if len(m.Capabilities) != 1 || m.Capabilities[0].Pkg != "internal/store" {
		t.Fatalf("capabilities = %+v", m.Capabilities)
	}
	if want := []string{"FILES"}; !reflect.DeepEqual(m.Capabilities[0].Direct, want) {
		t.Errorf("direct = %v, want %v", m.Capabilities[0].Direct, want)
	}
	if want := []string{"EXEC", "FILES"}; !reflect.DeepEqual(m.Capabilities[0].Deep, want) {
		t.Errorf("deep = %v, want %v", m.Capabilities[0].Deep, want)
	}
	if m.CapabilityAudit == nil || m.CapabilityAudit.Engine != "capslock" || m.CapabilityAudit.Version != "v0.3.4" {
		t.Errorf("capabilityAudit = %+v", m.CapabilityAudit)
	}
	if m.CapabilityFor("internal/store") == nil || m.CapabilityFor("internal/absent") != nil {
		t.Error("CapabilityFor lookup broken")
	}
}

func TestLoad_V1ManifestHasNoCapabilities(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": schemaBody,
		"facts.cue": `package io_surface

sites: "internal/store.LoadConfig": {
	location: ["internal/store/store.go:9"]
	family:   "file-read"
}
`,
	})
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load v1 manifest: %v", err)
	}
	if m.Version != "" || m.Capabilities != nil || m.CapabilityAudit != nil {
		t.Errorf("v1 manifest must decode with no v1.1 sections, got version=%q caps=%v audit=%v",
			m.Version, m.Capabilities, m.CapabilityAudit)
	}
}

func TestLoad_UnknownCapabilityNameFailsAtCUELevel(t *testing.T) {
	dir := writeManifest(t, map[string]string{
		"schema.cue": `package io_surface

sites: {[string]: _}

#CapabilityName: "FILES" | "NETWORK" | "EXEC"

capabilities: [...{pkg!: string, direct!: [...#CapabilityName], deep!: [...#CapabilityName]}] | *[]
`,
		"facts.cue": `package io_surface

sites: "internal/store.LoadConfig": {
	location: ["internal/store/store.go:9"]
	family:   "file-read"
}
capabilities: [{pkg: "internal/store", direct: ["CGO"], deep: ["CGO"]}]
`,
	})
	_, err := Load(dir)
	if err == nil {
		t.Fatal("an unmapped capability name must fail the closed enum")
	}
	if !errors.Is(err, ErrConstraints) {
		t.Fatalf("want ErrConstraints, got %v", err)
	}
}

func TestValidate_CapabilityInvariants(t *testing.T) {
	tests := []struct {
		name  string
		caps  []Capability
		check func(*testing.T, []error)
	}{
		{
			name: "unknown name",
			caps: []Capability{{Pkg: "internal/store", Direct: []string{"FILES"}, Deep: []string{"FILES", "CGO"}}},
			check: func(t *testing.T, errs []error) {
				if len(errs) != 1 || !strings.Contains(errs[0].Error(), `unknown capability "CGO"`) {
					t.Fatalf("want unknown-capability error, got %v", errs)
				}
			},
		},
		{
			name: "direct missing from deep",
			caps: []Capability{{Pkg: "internal/store", Direct: []string{"NETWORK"}, Deep: []string{"FILES"}}},
			check: func(t *testing.T, errs []error) {
				if len(errs) != 1 || !strings.Contains(errs[0].Error(), `direct "NETWORK" missing from deep`) {
					t.Fatalf("want deep-superset error, got %v", errs)
				}
			},
		},
		{
			name: "duplicate pkg",
			caps: []Capability{
				{Pkg: "internal/store", Direct: []string{"FILES"}, Deep: []string{"FILES"}},
				{Pkg: "internal/store", Direct: []string{"EXEC"}, Deep: []string{"EXEC"}},
			},
			check: func(t *testing.T, errs []error) {
				if len(errs) != 1 || !strings.Contains(errs[0].Error(), `duplicate pkg "internal/store"`) {
					t.Fatalf("want duplicate-pkg error, got %v", errs)
				}
			},
		},
		{
			name: "empty pkg",
			caps: []Capability{{Pkg: "", Deep: []string{"FILES"}}},
			check: func(t *testing.T, errs []error) {
				if len(errs) != 1 || !strings.Contains(errs[0].Error(), "empty pkg") {
					t.Fatalf("want empty-pkg error, got %v", errs)
				}
			},
		},
		{
			name: "clean set",
			caps: []Capability{{Pkg: "internal/store", Direct: []string{"FILES"}, Deep: []string{"EXEC", "FILES"}}},
			check: func(t *testing.T, errs []error) {
				if len(errs) != 0 {
					t.Fatalf("valid capabilities must pass, got %v", errs)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validateFixture(t, map[string]Site{"a.B": {Family: "env"}})
			m.Capabilities = tt.caps
			tt.check(t, m.Validate())
		})
	}
}

// v12Schema reads the REAL standard schema from the scaffold package's
// source tree (the single source of truth), keeping this test cycle-free.
func v12Schema(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../internal/scaffold/schema.cue")
	if err != nil {
		t.Fatalf("read standard schema: %v", err)
	}

	return string(data)
}

// TestLoad_V12ClosedSchema pins the v1.2 severity vocabulary (plan T19):
// the REAL embedded schema accepts the new claims fields and rejects values
// outside the closed enums — a typo like "critcal" must fail unification,
// not silently decode.
func TestLoad_V12ClosedSchema(t *testing.T) {
	base := map[string]string{
		"schema.cue": v12Schema(t),
		"facts.cue": `package io_surface

sites: "internal/store.LoadConfig": {
	location: ["internal/store/store.go:9"]
	family:   "file-read"
}
`,
		"meta.cue": `package io_surface

audit: {commit: "abc1234", date: "2026-11-07", claims: 1, method: "test audit"}
scope: {in: [], out: []}
`,
	}
	withClaims := func(claims string) map[string]string {
		m := map[string]string{}
		maps.Copy(m, base)
		m["claims.cue"] = "package io_surface\n\nsites: \"internal/store.LoadConfig\": claims: " + claims + "\n"

		return m
	}
	valid := writeManifest(t, withClaims(`{criticality: "critical", durability: "ephemeral"}`))
	m, err := Load(valid)
	if err != nil {
		t.Fatalf("v1.2 claims rejected by the real schema: %v", err)
	}
	if s := m.Sites["internal/store.LoadConfig"]; s.Claims.Criticality != "critical" || s.Claims.Durability != "ephemeral" {
		t.Errorf("v1.2 claims decoded wrong: %+v", s.Claims)
	}

	invalid := writeManifest(t, withClaims(`{criticality: "critcal", durability: "whenever"}`))
	if _, err := Load(invalid); err == nil {
		t.Fatal("out-of-vocabulary claims unified — the closed enums are broken")
	}

	defaults := writeManifest(t, withClaims(`{rationale: "defaults only"}`))
	m, err = Load(defaults)
	if err != nil {
		t.Fatalf("v1.1-style claims must keep unifying: %v", err)
	}
	// The schema defaults materialize on unification: unclaimed fields read
	// as normal/recoverable — the exact values the severity modulation
	// would have assumed anyway.
	if s := m.Sites["internal/store.LoadConfig"]; s.Claims.Criticality != "normal" || s.Claims.Durability != "recoverable" {
		t.Errorf("unclaimed fields must decode to the schema defaults, got %+v", s.Claims)
	}
}

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
