package manifest

import (
	"errors"
	"os"
	"path/filepath"
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
	resource:   "service config file"
	trigger:    "process start"
	discipline: ["bounded"]
	exception:  "wrapped"
	rationale:  "read once at boot"
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
	if !strings.Contains(err.Error(), "a.B") || !strings.Contains(err.Error(), "no claims.resource") {
		t.Fatalf("conflict error must surface the real violation entries, got: %v", err)
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

func TestValidate_DottedSubpackageID(t *testing.T) {
	m := validateFixture(t, map[string]Site{
		"internal/server.sso.Manager.ensureProvider": {Family: "outbound"},
		"internal/server.views.CSS":                  {Family: "file-read"},
	})
	errs := m.Validate()
	if len(errs) != 2 {
		t.Fatalf("want 2 dotted-subpackage errors, got %v", errs)
	}
	for _, want := range []string{"dotted subpackage", "internal/server/sso.Manager.ensureProvider", "internal/server/views.CSS"} {
		if !strings.Contains(errs[0].Error(), want) && !strings.Contains(errs[1].Error(), want) {
			t.Errorf("error must mention %q, got: %v / %v", want, errs[0], errs[1])
		}
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
