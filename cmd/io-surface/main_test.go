package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// runApp drives a real cmdguard execution against the wired CLI and returns
// what the command wrote to the output sink. Every CLI test goes through
// this seam: flag structs, cmdguard parsing, and the handler all run — only
// the bytes land in a buffer instead of the terminal.
func runApp(t *testing.T, args ...string) (string, error) {
	t.Helper()

	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}

	sink := &bytes.Buffer{}
	stdout = sink
	app.RootCommand().SetOut(sink)
	t.Cleanup(func() { stdout = os.Stdout })

	err = app.ExecuteWithArgs(context.Background(), args)

	return sink.String(), err
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want int
	}{
		{"clean", nil, 0},
		{"findings", errFindings, 1},
		{"wrapped findings", fmt.Errorf("checking: %w", errFindings), 1},
		{"usage", errors.New("unknown command"), 2},
		{"internal", os.ErrNotExist, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := exitCode(tc.err); got != tc.want {
				t.Errorf("exitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
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

	out, err := runApp(t, "check", "--root", root, "--json")
	if !errors.Is(err, errFindings) {
		t.Fatalf("check err = %v, want errFindings", err)
	}

	var rep check.Report
	if uErr := json.Unmarshal([]byte(out), &rep); uErr != nil {
		t.Fatalf("stdout is not a JSON report: %v\nstdout: %s", uErr, out)
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

	out, err := runApp(t, "check", "--root", root, "--json")
	if err != nil {
		t.Fatalf("check on green-by-construction fixture: %v", err)
	}
	var rep check.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("stdout is not a JSON report: %v\nstdout: %s", err, out)
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

	out, err := runApp(t, "check", "--root", root)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("io-surface: ")) {
		t.Errorf("text report missing summary line:\n%s", out)
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

	if _, err := runApp(t, "render", "--root", root, "--doc", docPath); err != nil {
		t.Fatalf("render --doc: %v", err)
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
	if _, err := runApp(t, "render", "--root", root, "--doc", docPath); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if _, err := runApp(t, "render", "--root", root, "--doc", docPath, "--check"); err != nil {
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
	_, err = runApp(t, "render", "--root", root, "--doc", docPath, "--check")
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
		name    string
		args    []string
		isJSON  bool
		isSARIF bool
	}{
		{name: "json flag", args: []string{"--json"}, isJSON: true},
		{name: "format json", args: []string{"--format", "json"}, isJSON: true},
		{name: "format sarif", args: []string{"--format", "sarif"}, isSARIF: true},
		{name: "text default", args: nil},
	}
	for _, tc := range cases {
		// No t.Parallel(): the subtests share the package output sink.
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"check", "--root", root}, tc.args...)
			out, err := runApp(t, args...)
			if err != nil {
				t.Fatalf("check %v: %v", tc.args, err)
			}
			switch {
			case tc.isJSON:
				var rep check.Report
				if err := json.Unmarshal([]byte(out), &rep); err != nil {
					t.Fatalf("--json path not JSON: %v\n%s", err, out)
				}
			case tc.isSARIF:
				var doc struct {
					Version string `json:"version"`
					Runs    []struct {
						Results []any `json:"results"`
					} `json:"runs"`
				}
				if err := json.Unmarshal([]byte(out), &doc); err != nil {
					t.Fatalf("sarif path not JSON: %v\n%s", err, out)
				}
				if len(doc.Runs) != 1 {
					t.Fatalf("sarif runs = %d", len(doc.Runs))
				}
			default:
				if !bytes.Contains([]byte(out), []byte("io-surface: ")) {
					t.Errorf("text path lost summary:\n%s", out)
				}
			}
		})
	}
}

// TestExecute_VersionAndUnknownCommand pin the two trivial surface facts:
// the version command prints the same line it always has, and an unknown
// command is a usage error (exit-code class 2), not a gate finding.
func TestExecute_VersionAndUnknownCommand(t *testing.T) {
	t.Parallel()

	out, err := runApp(t, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if out != "io-surface "+version+"\n" {
		t.Errorf("version output = %q, want %q", out, "io-surface "+version+"\n")
	}

	_, err = runApp(t, "definitely-not-a-command")
	if err == nil {
		t.Fatal("unknown command must fail")
	}
	if exitCode(err) != 2 {
		t.Errorf("unknown command exit class = %d, want 2", exitCode(err))
	}
}
