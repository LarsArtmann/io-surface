package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
)

// TestRun_FixtureReport pins the package-level contract: the bug fixture
// produces exactly the err-dropped findings, the report is Enforcement "lint",
// and severity conversion maps the registry strings to check.Severity.
func TestRun_FixtureReport(t *testing.T) {
	rep, err := Run("../../testdata/detectors", Options{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if rep.Enforcement != "lint" {
		t.Errorf("enforcement = %q, want lint", rep.Enforcement)
	}
	if rep.Stats.Infos != 41 || rep.Stats.Errors != 0 || rep.Stats.Warnings != 0 {
		t.Errorf("stats = %+v, want 41 infos", rep.Stats)
	}
	if rep.Stats.Extracted != 56 || rep.Stats.Resolved != 54 {
		t.Errorf("stats = %+v, want 56 extracted / 54 resolved", rep.Stats)
	}
	if len(rep.Findings) != 41 {
		t.Fatalf("findings = %d, want 41", len(rep.Findings))
	}
	allowed := map[string]bool{
		RuleID("err-dropped"):       true,
		RuleID("closer-missing"):    true,
		RuleID("ctx-dropped"):       true,
		RuleID("defer-in-loop"):     true,
		RuleID("io-in-loop"):        true,
		RuleID("client-no-timeout"): true,
		RuleID("stat-then-open"):    true,
		RuleID("write-non-atomic"):  true,
		RuleID("tiny-buffer"):       true,
		RuleID("ignore-invalid"):    true,
	}
	for _, f := range rep.Findings {
		if !allowed[f.Rule] {
			t.Errorf("unexpected rule %q (vocabulary drift)", f.Rule)
		}
		if f.Severity != check.SeverityInfo {
			t.Errorf("severity = %q, want info (D4: rules ship at info)", f.Severity)
		}
		if f.ID == "" || f.Location == "" || f.Message == "" {
			t.Errorf("finding missing ID/location/message: %+v", f)
		}
	}
}

// TestSeverityOf pins the promotion vocabulary mapping. Off-vocabulary
// values (including the empty string) degrade to info — the same default
// the registry ships at.
func TestSeverityOf(t *testing.T) {
	cases := map[extract.Severity]check.Severity{
		extract.SeverityError:     check.SeverityError,
		extract.SeverityWarning:   check.SeverityWarning,
		extract.SeverityInfo:      check.SeverityInfo,
		"":                        check.SeverityInfo,
		extract.Severity("bogus"): check.SeverityInfo,
	}
	for in, want := range cases {
		if got := severityOf(in); got != want {
			t.Errorf("severityOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRun_ClaimSeverityModulation pins the v1.2 escalation (plan T20): a
// finding on a site whose claims declare criticality=critical escalates
// info→warning, and a missing manifest degrades to no modulation.
func TestRun_ClaimSeverityModulation(t *testing.T) {
	dir := t.TempDir()
	manifestFiles := map[string]string{
		"schema.cue": schemaOf(t),
		"facts.cue": `package io_surface

sites: "cmd/app.positiveBlank": {
	location: ["cmd/app/main.go:16"]
	family:   "file-read"
}
`,
		"claims.cue": `package io_surface

sites: "cmd/app.positiveBlank": claims: {
	resource:    "hot config"
	criticality: "critical"
	durability:  "durable"
}
`,
		"baseline.cue": "package io_surface\n\nbaseline: []\n",
		"meta.cue": `package io_surface

audit: {commit: "abc", date: "2026-11-07", claims: 1, method: "test audit"}
scope: {in: [], out: []}
`,
	}
	manifestPath := filepath.Join(dir, "docs", "architecture", "io-surface")
	if err := os.MkdirAll(manifestPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for name, body := range manifestFiles {
		if err := os.WriteFile(filepath.Join(manifestPath, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	rep, err := Run("../../testdata/detectors", Options{Manifest: manifestPath})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	escalated := 0
	for _, f := range rep.Findings {
		if f.ID == "cmd/app.positiveBlank" && f.Rule == RuleID("err-dropped") {
			if f.Severity != check.SeverityWarning {
				t.Fatalf("critical site finding severity = %q, want warning", f.Severity)
			}
			if !strings.Contains(f.Message, "escalated") {
				t.Errorf("escalated finding must say so: %q", f.Message)
			}
			escalated++
		}
	}
	if escalated != 1 {
		t.Fatalf("escalated findings = %d, want 1", escalated)
	}
	if rep.Stats.Warnings != 1 {
		t.Errorf("warnings = %d, want 1", rep.Stats.Warnings)
	}

	// No manifest: the same finding stays info.
	plain, err := Run("../../testdata/detectors", Options{})
	if err != nil {
		t.Fatalf("run without manifest: %v", err)
	}
	for _, f := range plain.Findings {
		if f.ID == "cmd/app.positiveBlank" && f.Rule == RuleID("err-dropped") && f.Severity != check.SeverityInfo {
			t.Fatalf("without a manifest the finding must stay info, got %q", f.Severity)
		}
	}
}

// TestRun_OwnershipModulation pins the v1.3 Q6 semantics on the real
// detector tree: a `caller` claim drops closer-missing unconditionally (the
// declared transfer-out covers shapes the AST cannot see), a `helper` claim
// drops only when the named helper is really called in the enclosing
// function, and a helper claim that names a call the function never makes
// KEEPS the finding (a lying claim must not silence the detector).
func TestRun_OwnershipModulation(t *testing.T) {
	dir := t.TempDir()
	manifestFiles := map[string]string{
		"schema.cue": schemaOf(t),
		"facts.cue": `package io_surface

sites: {
	"internal/store.leakUsedNotClosed": {
		location: ["internal/store/closers.go:21"]
		family:   "file-read"
	}
	"internal/store.leakNeverUsed": {
		location: ["internal/store/closers.go:11"]
		family:   "file-read"
	}
	"cmd/app.leakBody": {
		location: ["cmd/app/http.go:11"]
		family:   "outbound"
	}
}
`,
		"claims.cue": `package io_surface

sites: {
	"internal/store.leakUsedNotClosed": claims: ownership: "caller"
	"cmd/app.leakBody": claims: {
		ownership: "helper"
		helper:    "io.ReadAll"
	}
	"internal/store.leakNeverUsed": claims: {
		ownership: "helper"
		helper:    "store.drain"
	}
}
`,
		"baseline.cue": "package io_surface\n\nbaseline: []\n",
		"meta.cue": `package io_surface

audit: {commit: "abc", date: "2026-11-22", claims: 3, method: "test audit"}
scope: {in: [], out: []}
`,
	}
	manifestPath := filepath.Join(dir, "docs", "architecture", "io-surface")
	if err := os.MkdirAll(manifestPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for name, body := range manifestFiles {
		if err := os.WriteFile(filepath.Join(manifestPath, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	rep, err := Run("../../testdata/detectors", Options{Manifest: manifestPath})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	present := map[string]bool{}
	for _, f := range rep.Findings {
		if f.Rule == RuleID("closer-missing") {
			present[f.ID] = true
		}
	}
	if present["internal/store.leakUsedNotClosed"] {
		t.Error("caller claim must drop closer-missing (declared transfer-out policy)")
	}
	if present["cmd/app.leakBody"] {
		t.Error("helper claim whose helper is really called (io.ReadAll) must drop closer-missing")
	}
	if !present["internal/store.leakNeverUsed"] {
		t.Error("helper claim naming an uncalled helper (store.drain) must KEEP closer-missing")
	}
	if !present["cmd/app.statusOnlyLeak"] {
		t.Error("unclaimed site keeps the finding (self default)")
	}
}

// schemaOf reads the real standard schema (cycle-free, like the manifest
// package's own v1.2 test).
func schemaOf(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../internal/scaffold/schema.cue")
	if err != nil {
		t.Fatalf("read standard schema: %v", err)
	}

	return string(data)
}
