package lint

import (
	"testing"

	"github.com/larsartmann/io-surface/internal/check"
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
	if rep.Stats.Infos != 11 || rep.Stats.Errors != 0 || rep.Stats.Warnings != 0 {
		t.Errorf("stats = %+v, want 11 infos", rep.Stats)
	}
	if rep.Stats.Extracted != 32 || rep.Stats.Resolved != 31 {
		t.Errorf("stats = %+v, want 32 extracted / 31 resolved", rep.Stats)
	}
	if len(rep.Findings) != 11 {
		t.Fatalf("findings = %d, want 3", len(rep.Findings))
	}
	allowed := map[string]bool{
		RuleID("err-dropped"):    true,
		RuleID("closer-missing"): true,
		RuleID("ctx-dropped"):    true,
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

// TestSeverityOf pins the promotion vocabulary mapping.
func TestSeverityOf(t *testing.T) {
	cases := map[string]check.Severity{
		"error":   check.SeverityError,
		"warning": check.SeverityWarning,
		"info":    check.SeverityInfo,
		"":        check.SeverityInfo,
		"bogus":   check.SeverityInfo,
	}
	for in, want := range cases {
		if got := severityOf(in); got != want {
			t.Errorf("severityOf(%q) = %q, want %q", in, got, want)
		}
	}
}
