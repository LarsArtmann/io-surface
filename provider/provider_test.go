package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/scaffold"
)

// fixtureRoot copies the shared miniature tree and scaffolds its manifest.
func fixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../testdata/extract")); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	sites, err := extract.Extract(root, extract.Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err = scaffold.Write(root, sites, "2026-11-06", false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	return root
}

func detectAt(t *testing.T, root string) ([]finding.Finding, error) {
	t.Helper()

	return Provider.Detect.Detect(finding.WithWorkingDir(context.Background(), root))
}

// TestDetect_NoManifestIsCleanNoOp pins the fleet-sweep contract: BuildFlow
// gates every repo, and a repo without a manifest must not fail the run.
func TestDetect_NoManifestIsCleanNoOp(t *testing.T) {
	findings, err := detectAt(t, t.TempDir())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("findings = %d, want 0 (unadopted repo is a no-op)", len(findings))
	}
}

// TestDetect_ReportsUndeclaredSite pins the happy path: a planted undeclared
// site surfaces with the CLI's rule ID (suppression vocabulary transfers
// verbatim), error severity, and the parsed file:line position.
func TestDetect_ReportsUndeclaredSite(t *testing.T) {
	root := fixtureRoot(t)
	late := filepath.Join(root, "internal", "store", "late.go")
	if err := os.WriteFile(late, []byte("package store\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n"), 0o644); err != nil {
		t.Fatalf("plant: %v", err)
	}

	findings, err := detectAt(t, root)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	var hit *finding.Finding
	for i := range findings {
		if findings[i].Rule == finding.RuleName(check.RuleUndeclared) {
			hit = &findings[i]

			break
		}
	}
	if hit == nil {
		t.Fatalf("no %s finding in %d findings", check.RuleUndeclared, len(findings))
	}
	if hit.Severity != finding.SeverityError {
		t.Errorf("severity = %q, want error (gate policy passes through)", hit.Severity)
	}
	if string(hit.Position.File) != filepath.Join("internal", "store", "late.go") || hit.Position.Line != 6 {
		t.Errorf("position = %s:%d, want internal/store/late.go:6", hit.Position.File, hit.Position.Line)
	}
}

// TestDetect_AuditPolicyDemotesErrors pins the D4 mapping: a manifest that
// records enforcement audit demotes error findings to warning, so BuildFlow's
// fail-on gate follows the repo's recorded posture instead of the CLI's.
func TestDetect_AuditPolicyDemotesErrors(t *testing.T) {
	root := fixtureRoot(t)
	late := filepath.Join(root, "internal", "store", "late.go")
	if err := os.WriteFile(late, []byte("package store\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n"), 0o644); err != nil {
		t.Fatalf("plant: %v", err)
	}
	facts := filepath.Join(root, scaffold.Dir, "facts.cue")
	raw, err := os.ReadFile(facts)
	if err != nil {
		t.Fatalf("read facts: %v", err)
	}
	audited := replaceOnce(string(raw), "enforcement: \"gate\"", "enforcement: \"audit\"")
	if audited == string(raw) {
		t.Fatal("fixture lost the enforcement line")
	}
	if err := os.WriteFile(facts, []byte(audited), 0o644); err != nil {
		t.Fatalf("write facts: %v", err)
	}

	findings, err := detectAt(t, root)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	for _, f := range findings {
		if f.Rule == finding.RuleName(check.RuleUndeclared) && f.Severity == finding.SeverityError {
			t.Errorf("audit policy leaked an error-severity finding: %+v", f)
		}
	}
}

// TestConvert_LocationShapes pins the position parsing: plain path:line
// parses, the "--dirs" pseudo-location stays path-only.
func TestConvert_LocationShapes(t *testing.T) {
	rep := &check.Report{Findings: []check.Finding{
		{Rule: check.RuleUndeclared, Severity: check.SeverityError, ID: "a.B", Location: "cmd/app/main.go:19"},
		{Rule: check.RuleScopeHint, Severity: check.SeverityInfo, ID: "--dirs", Location: "--dirs"},
	}}
	out := convert(rep, manifest.EnforcementGate)
	if len(out) != 2 {
		t.Fatalf("findings = %d, want 2", len(out))
	}
	if out[0].Position.Line != 19 || out[0].Position.File != "cmd/app/main.go" {
		t.Errorf("parsed position = %s:%d", out[0].Position.File, out[0].Position.Line)
	}
	if out[1].Position.File != "--dirs" || out[1].Position.Line != 0 {
		t.Errorf("pseudo-location = %s:%d, want --dirs:0", out[1].Position.File, out[1].Position.Line)
	}
}

func replaceOnce(s, old, new string) string {
	idx := indexOf(s, old)
	if idx < 0 {
		return s
	}

	return s[:idx] + new + s[idx+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}

	return -1
}
