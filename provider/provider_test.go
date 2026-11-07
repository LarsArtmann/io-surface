package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/lint"
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

// plantFile writes a Go file into the fixture-relative path.
func plantFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("plant: %v", err)
	}
}

// TestDetect_EmptyModuleIsSilent pins the floor of the fleet-sweep
// contract: a manifestless module with no code yields no findings and no
// step error.
func TestDetect_EmptyModuleIsSilent(t *testing.T) {
	findings, err := detectAt(t, t.TempDir())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("findings = %d, want 0 (nothing to scan)", len(findings))
	}
}

// TestDetect_ManifestlessRunsRadar pins the fleet-sweep contract for real
// code: BuildFlow gates every repo, and a repo that never adopted io-surface
// gets the manifestless radar face — real findings at info severity, never
// a step error, so an unadopted repo cannot fail a fleet run.
func TestDetect_ManifestlessRunsRadar(t *testing.T) {
	root := t.TempDir()
	plantFile(t, filepath.Join(root, "internal", "store", "w.go"),
		"package store\n\nimport \"os\"\n\nfunc W() error {\n\treturn os.WriteFile(\"x\", []byte(\"y\"), 0o644)\n}\n")

	findings, err := detectAt(t, root)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	radarRule := finding.RuleName(lint.RuleID(extract.RuleWriteNonAtomic))
	var hit *finding.Finding
	for i := range findings {
		if findings[i].Rule == radarRule {
			hit = &findings[i]

			break
		}
	}
	if hit == nil {
		t.Fatalf("no %s finding in %d findings", radarRule, len(findings))
	}
	if hit.Severity != finding.SeverityInfo {
		t.Errorf("severity = %q, want info (radar ships advisory until D4 promotion)", hit.Severity)
	}
}

// TestDetect_ReportsUndeclaredSite pins the happy path: a planted undeclared
// site surfaces with the CLI's rule ID (suppression vocabulary transfers
// verbatim), error severity, and the parsed file:line position.
func TestDetect_ReportsUndeclaredSite(t *testing.T) {
	root := fixtureRoot(t)
	plantFile(t, filepath.Join(root, "internal", "store", "late.go"),
		"package store\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n")

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

// TestDetect_AdoptedModuleMergesGateAndRadar pins the merged face: an
// adopted module (manifest present) gets gate findings AND radar findings in
// one report — one vocabulary, one enforcement policy.
func TestDetect_AdoptedModuleMergesGateAndRadar(t *testing.T) {
	root := fixtureRoot(t)
	plantFile(t, filepath.Join(root, "internal", "store", "late.go"),
		"package store\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n")
	plantFile(t, filepath.Join(root, "internal", "store", "w.go"),
		"package store\n\nimport \"os\"\n\nfunc W() error {\n\treturn os.WriteFile(\"x\", []byte(\"y\"), 0o644)\n}\n")

	findings, err := detectAt(t, root)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	has := func(rule finding.RuleName) bool {
		for _, f := range findings {
			if f.Rule == rule {
				return true
			}
		}

		return false
	}
	if !has(finding.RuleName(check.RuleUndeclared)) {
		t.Errorf("no %s finding in %d findings (gate face missing)", check.RuleUndeclared, len(findings))
	}
	if !has(finding.RuleName(lint.RuleID(extract.RuleWriteNonAtomic))) {
		t.Errorf("no %s finding in %d findings (radar face missing)", lint.RuleID(extract.RuleWriteNonAtomic), len(findings))
	}
}

// TestDetect_AuditPolicyDemotesErrors pins the D4 mapping: a manifest that
// records enforcement audit demotes error findings to warning, so BuildFlow's
// fail-on gate follows the repo's recorded posture instead of the CLI's.
func TestDetect_AuditPolicyDemotesErrors(t *testing.T) {
	root := fixtureRoot(t)
	plantFile(t, filepath.Join(root, "internal", "store", "late.go"),
		"package store\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n")
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

// TestConvert_AuditDemotesRadarEscalations pins that the one policy governs
// both faces: a radar finding a critical claim escalated to error demotes
// under the audit policy, exactly like a gate error.
func TestConvert_AuditDemotesRadarEscalations(t *testing.T) {
	rep := &check.Report{Findings: []check.Finding{
		{Rule: lint.RuleID(extract.RuleCloserMissing), Severity: check.SeverityError, ID: "a.B", Location: "a.go:1"},
	}}
	out := convert(rep, manifest.EnforcementAudit)
	if len(out) != 1 || out[0].Severity != finding.SeverityWarning {
		t.Fatalf("severity = %v, want warning (audit demotion covers radar escalations)", out)
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
