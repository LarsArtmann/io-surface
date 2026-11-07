package main

// The manifestless lint face's end-to-end contract (plan T5/T4): policy
// behavior through the real cmdguard execution, and byte-pinned text + SARIF
// goldens. Split from main_test.go to honor the 650-line file gate.

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/check"
)

// TestExecute_LintFace is the e2e contract for the manifestless lint face
// (plan T5): findings on the bug fixture, the exit-0-by-default policy, the
// --enforce policy, and the JSON report shape machines consume.
func TestExecute_LintFace(t *testing.T) {
	out, err := runApp(t, "lint", "--root", "../../testdata/detectors")
	if err != nil {
		t.Fatalf("lint: %v (findings must not fail by default)\n%s", err, out)
	}
	if !strings.Contains(out, "io-surface/err-dropped") || !strings.Contains(out, "data, _ := os.ReadFile(path)") {
		t.Errorf("lint text output lost the err-dropped finding:\n%s", out)
	}
	if !strings.Contains(out, "report only; pass --enforce to fail on errors") {
		t.Errorf("lint summary line lost the policy hint:\n%s", out)
	}
	if strings.Contains(out, "cmd/app.negativeDeliberate") || strings.Contains(out, "negativePropagate") {
		t.Errorf("lint reported a deliberate-ignore or propagated call:\n%s", out)
	}

	out, err = runApp(t, "lint", "--root", "../../testdata/detectors", "--enforce")
	if err != nil {
		t.Fatalf("lint --enforce: %v (info findings must not fail)\n%s", err, out)
	}

	out, err = runApp(t, "lint", "--root", "../../testdata/detectors", "--format", "json")
	if err != nil {
		t.Fatalf("lint --format json: %v\n%s", err, out)
	}
	var rep check.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("lint JSON not parseable: %v\n%s", err, out)
	}
	if rep.Enforcement != "lint" {
		t.Errorf("enforcement = %q, want lint", rep.Enforcement)
	}
	if rep.Stats.Infos != 41 || rep.Stats.Errors != 0 {
		t.Errorf("stats = %+v, want 41 infos / 0 errors", rep.Stats)
	}
	if len(rep.Findings) != 41 {
		t.Errorf("findings = %d, want 41", len(rep.Findings))
	}

	out, err = runApp(t, "lint", "--root", "../../testdata/detectors", "--format", "sarif")
	if err != nil {
		t.Fatalf("lint --format sarif: %v\n%s", err, out)
	}
	if !strings.Contains(out, "io-surface/err-dropped") || !strings.Contains(out, `"version"`) {
		t.Errorf("lint SARIF lost rule IDs or version:\n%.200s", out)
	}
}

var updateGoldens = flag.Bool("update", false, "rewrite the lint golden files")

// TestGolden_LintOutput byte-pins the lint face's two machine-relevant
// outputs (plan T4): the text report users read and the SARIF document
// pipelines consume (rule IDs + severities + version are the suppression
// contract — D7). Intentional output changes re-run with -update.
func TestGolden_LintOutput(t *testing.T) {
	out, err := runApp(t, "lint", "--root", "../../testdata/detectors")
	if err != nil {
		t.Fatalf("lint: %v\n%s", err, out)
	}
	goldenPath := "testdata/lint-text.golden"
	if *updateGoldens {
		if err := os.WriteFile(goldenPath, []byte(out), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if out != string(want) {
		t.Errorf("lint text output drifted from the golden (re-run `go test ./cmd/io-surface -run TestGolden_LintOutput -update` after an intentional change):\n--- want ---\n%s\n--- got ---\n%s", want, out)
	}

	sarifOut, err := runApp(t, "lint", "--root", "../../testdata/detectors", "--format", "sarif")
	if err != nil {
		t.Fatalf("lint sarif: %v\n%s", err, sarifOut)
	}
	sarifOut += "\n"
	sarifPath := "testdata/lint-sarif.golden"
	if *updateGoldens {
		if err := os.WriteFile(sarifPath, []byte(sarifOut), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
	}
	wantSarif, err := os.ReadFile(sarifPath)
	if err != nil {
		t.Fatalf("read sarif golden: %v", err)
	}
	if sarifOut != string(wantSarif) {
		t.Errorf("lint SARIF output drifted from the golden (re-run with -update):\n--- want (first 400) ---\n%.400s\n--- got (first 400) ---\n%.400s", wantSarif, sarifOut)
	}
}

// TestExecute_LintGitHubFormat pins the workflow-annotation face (superb
// plan T14): every finding renders as one ::notice/error/warning command
// anchored at file:line, and the exit-code contract stays report-only.
func TestExecute_LintGitHubFormat(t *testing.T) {
	out, err := runApp(t, "lint", "--root", "../../testdata/detectors", "--format", "github")
	if err != nil {
		t.Fatalf("lint --format github: %v\n%s", err, out)
	}
	if !strings.Contains(out, "::notice title=io-surface/err-dropped,file=cmd/app/main.go,line=") {
		t.Errorf("annotations lost the file/line anchor:\n%.400s", out)
	}
	if !strings.Contains(out, "title=io-surface/err-dropped") {
		t.Errorf("annotations lost the rule title:\n%.400s", out)
	}
	if strings.Contains(out, "::error") || strings.Contains(out, "::warning") {
		t.Errorf("info findings must render as ::notice only:\n%.400s", out)
	}
	if !strings.Contains(out, "::notice title=io-surface lint::") {
		t.Errorf("annotations lost the summary line:\n%.400s", out)
	}
}
