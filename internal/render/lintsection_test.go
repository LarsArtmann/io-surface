package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/check"
)

// lintSectionReport is a hand-built radar report in the order lint.Run
// delivers (check.SortFindings: severity rank, then rule, then ID) — the
// renderer trusts that order, so the golden pins the sorted contract. It
// exercises every cell: three severities, a location-less finding (gate
// rules can lack locations), and a message containing a pipe (must not
// break the GFM table).
func lintSectionReport() *check.Report {
	return &check.Report{
		Enforcement: "lint",
		Stats:       check.Stats{Extracted: 12, Resolved: 7, Token: 5, Errors: 1, Warnings: 1, Infos: 2},
		Findings: []check.Finding{
			{Rule: "io-surface/stale-declaration", Severity: check.SeverityError, ID: "docs/x.Declared", Message: "declared site no longer extracted"},
			{Rule: "io-surface/write-non-atomic", Severity: check.SeverityWarning, ID: "internal/c.Three", Location: "internal/c/c.go:8", Message: "os.WriteFile|truncate risk"},
			{Rule: "io-surface/err-dropped", Severity: check.SeverityInfo, ID: "internal/a.One", Location: "internal/a/a.go:12", Message: "error from Close dropped"},
			{Rule: "io-surface/io-in-loop", Severity: check.SeverityInfo, ID: "internal/b.Two", Location: "internal/b/b.go:30", Message: "per-item Read inside range loop"},
		},
	}
}

func TestLintSection_Golden(t *testing.T) {
	got := LintSection(lintSectionReport())
	golden := filepath.Join("testdata", "lint-section.golden")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Errorf("lint section drifted from golden; re-run `go test ./internal/render -update` only after word-by-word review")
	}
}

func TestLintSection_EmptyReport(t *testing.T) {
	got := LintSection(&check.Report{Enforcement: "lint"})
	if !strings.Contains(got, "## Lint radar (findings)") {
		t.Error("empty report lost the section heading")
	}
	if !strings.Contains(got, "No detector findings.") {
		t.Error("empty report must say so explicitly (absence is indistinguishable from staleness)")
	}
	if strings.Contains(got, "| Severity") {
		t.Errorf("empty report rendered an empty table:\n%s", got)
	}
}

func TestLintSection_Deterministic(t *testing.T) {
	if LintSection(lintSectionReport()) != LintSection(lintSectionReport()) {
		t.Error("LintSection is not deterministic for the same report")
	}
}
