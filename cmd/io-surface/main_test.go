package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/scaffold"
)

// scaffoldFixture copies the shared miniature tree into root and writes the
// four-file manifest for it (green by construction).
func scaffoldFixture(t *testing.T, root string) {
	t.Helper()
	if err := os.CopyFS(root, os.DirFS("../../testdata/extract")); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	sites, err := extract.Extract(root, extract.Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err = scaffold.Write(root, sites, "2026-11-06", false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
}

// TestRunCheck_ValidationFailureEmitsJSON is the machine-contract proof: a
// manifest that fails validation must produce a parseable JSON report on
// stdout (rule io-surface/manifest-validation, error severity, exit 1) —
// never a stderr-only text wall with no JSON.
func TestRunCheck_ValidationFailureEmitsJSON(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)
	if err := os.Remove(filepath.Join(root, scaffold.Dir, "schema.cue")); err != nil {
		t.Fatalf("remove schema.cue: %v", err)
	}

	var out bytes.Buffer
	err := runCheck([]string{"--root", root, "--json"}, &out)
	if !errors.Is(err, errFindings) {
		t.Fatalf("runCheck err = %v, want errFindings", err)
	}

	var rep check.Report
	if uErr := json.Unmarshal(out.Bytes(), &rep); uErr != nil {
		t.Fatalf("stdout is not a JSON report: %v\nstdout: %s", uErr, out.String())
	}
	if rep.Stats.Errors == 0 {
		t.Errorf("stats.errors = 0, want >= 1 (findings: %+v)", rep.Findings)
	}
	found := false
	for _, f := range rep.Findings {
		if f.Rule == check.RuleManifestValidation && f.Severity == check.SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("no %s error finding in %+v", check.RuleManifestValidation, rep.Findings)
	}
}

// TestRunCheck_CleanEmitsJSON locks the other half of the contract: a clean
// gate under --json parses as a report with zero errors.
func TestRunCheck_CleanEmitsJSON(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	var out bytes.Buffer
	if err := runCheck([]string{"--root", root, "--json"}, &out); err != nil {
		t.Fatalf("runCheck on green-by-construction fixture: %v", err)
	}
	var rep check.Report
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("stdout is not a JSON report: %v\nstdout: %s", err, out.String())
	}
	if rep.Stats.Errors != 0 || rep.Stats.Declared == 0 {
		t.Errorf("unexpected clean stats: %+v", rep.Stats)
	}
}

// TestRunCheck_TextModeStillPrints pins the human-readable path next to the
// JSON path so the summary line keeps its shape.
func TestRunCheck_TextModeStillPrints(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	var out bytes.Buffer
	if err := runCheck([]string{"--root", root}, &out); err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("io-surface: ")) {
		t.Errorf("text report missing summary line:\n%s", out.String())
	}
}

// renderDocProse is a hand-authored doc with an embedded (stale) inventory
// block, exercising the --doc/--check contract end to end.
const renderDocProse = `# Notes

Hand-written intro.

<!-- io-surface:begin -->
stale generated content
<!-- io-surface:end -->

Hand-written outro.
`

func TestRunRender_DocAppliesInPlace(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)
	docPath := filepath.Join(root, "NOTES.md")
	if err := os.WriteFile(docPath, []byte(renderDocProse), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	var out bytes.Buffer
	if err := runRender([]string{"--root", root, "--doc", docPath}, &out); err != nil {
		t.Fatalf("runRender --doc: %v", err)
	}
	got, _ := os.ReadFile(docPath)
	if bytes.Contains(got, []byte("stale generated content")) {
		t.Error("stale block survived --doc apply")
	}
	if !bytes.Contains(got, []byte("## Taxonomy at a glance")) {
		t.Error("fresh block missing after --doc apply")
	}
	for _, keep := range []string{"# Notes", "Hand-written intro.", "Hand-written outro.", "io-surface:begin", "io-surface:end"} {
		if !bytes.Contains(got, []byte(keep)) {
			t.Errorf("prose/marker lost: %q", keep)
		}
	}
}

func TestRunRender_CheckFreshAndStale(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)
	docPath := filepath.Join(root, "NOTES.md")
	if err := os.WriteFile(docPath, []byte(renderDocProse), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	var out bytes.Buffer
	if err := runRender([]string{"--root", root, "--doc", docPath}, &out); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if err := runRender([]string{"--root", root, "--doc", docPath, "--check"}, &out); err != nil {
		t.Fatalf("--check on fresh doc: %v", err)
	}

	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	stale := bytes.Replace(data, []byte("Taxonomy at a glance"), []byte("CHANGED HEADING"), 1)
	if err := os.WriteFile(docPath, stale, 0o644); err != nil {
		t.Fatalf("write stale doc: %v", err)
	}
	err = runRender([]string{"--root", root, "--doc", docPath, "--check"}, &out)
	if !errors.Is(err, errFindings) {
		t.Fatalf("--check on stale doc err = %v, want errFindings", err)
	}
}

// TestRunCheck_FormatMatrix locks the flag surface: --json, --format json and
// --format sarif all emit parseable machine output on every outcome.
func TestRunCheck_FormatMatrix(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	cases := []struct {
		args    []string
		isJSON  bool
		isSARIF bool
	}{
		{args: []string{"--root", root, "--json"}, isJSON: true},
		{args: []string{"--root", root, "--format", "json"}, isJSON: true},
		{args: []string{"--root", root, "--format", "sarif"}, isSARIF: true},
		{args: []string{"--root", root}},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		if err := runCheck(tc.args, &out); err != nil {
			t.Fatalf("runCheck %v: %v", tc.args, err)
		}
		switch {
		case tc.isJSON:
			var rep check.Report
			if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
				t.Fatalf("--json path not JSON: %v\n%s", err, out.String())
			}
		case tc.isSARIF:
			var doc struct {
				Version string `json:"version"`
				Runs    []struct {
					Results []any `json:"results"`
				} `json:"runs"`
			}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatalf("sarif path not JSON: %v\n%s", err, out.String())
			}
			if len(doc.Runs) != 1 {
				t.Fatalf("sarif runs = %d", len(doc.Runs))
			}
		default:
			if !bytes.Contains(out.Bytes(), []byte("io-surface: ")) {
				t.Errorf("text path lost summary:\n%s", out.String())
			}
		}
	}
}
