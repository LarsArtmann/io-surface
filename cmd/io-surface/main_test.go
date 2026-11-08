package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-atomic-write"
	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
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

// TestApplyDoc_ConcurrentModification pins the doc seam's TOCTOU guard
// (TODO #115): writeDocVerified commits only against the fingerprint captured
// at read time, so a concurrent writer fails the command with re-run
// guidance instead of being clobbered, and their bytes survive on disk. This
// is the same-seam proof the baseline append path carries; the race mechanics
// themselves are go-atomic-write's own test suite.
func TestApplyDoc_ConcurrentModification(t *testing.T) {
	docPath := filepath.Join(t.TempDir(), "NOTES.md")
	if err := os.WriteFile(docPath, []byte(renderDocProse), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	t.Run("stale fingerprint refuses and preserves the concurrent edit", func(t *testing.T) {
		data, err := os.ReadFile(docPath)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		fp := atomicwrite.FingerprintFromBytes(data)

		if err := os.WriteFile(docPath, []byte("# concurrent edit by another session\n"), 0o644); err != nil {
			t.Fatalf("concurrent write: %v", err)
		}

		err = writeDocVerified(docPath, []byte("# regenerated block"), fp)
		if !errors.Is(err, atomicwrite.ErrConcurrentModification) {
			t.Fatalf("want ErrConcurrentModification, got %v", err)
		}
		if !strings.Contains(err.Error(), "re-run the command to merge") {
			t.Errorf("error lacks re-run guidance: %v", err)
		}
		got, rErr := os.ReadFile(docPath)
		if rErr != nil {
			t.Fatalf("read back: %v", rErr)
		}
		if string(got) != "# concurrent edit by another session\n" {
			t.Errorf("concurrent edit was clobbered, file holds %q", got)
		}
	})

	t.Run("fresh fingerprint commits", func(t *testing.T) {
		data, err := os.ReadFile(docPath)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		fp := atomicwrite.FingerprintFromBytes(data)
		if err := writeDocVerified(docPath, []byte("# regenerated block"), fp); err != nil {
			t.Fatalf("verified write: %v", err)
		}
		got, _ := os.ReadFile(docPath)
		if string(got) != "# regenerated block" {
			t.Errorf("file holds %q, want the regenerated block", got)
		}
	})
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

// TestRunCheck_EnforcementAuditMode pins the D4 policy contract end to end:
// a manifest that records `enforcement: "audit"` reports error findings but
// exits 0 (the opt-out lives in the manifest — a committed, reviewable diff),
// `--enforce` overrides it, and the JSON report carries the policy so CI logs
// show it without re-reading the manifest.
func TestRunCheck_EnforcementAuditMode(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	// Plant one undeclared resolved site so the gate has real errors to
	// suppress, then flip the recorded policy to audit.
	late := filepath.Join(root, "internal", "store", "late.go")
	if err := os.WriteFile(late, []byte("package store\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n"), 0o644); err != nil {
		t.Fatalf("plant undeclared site: %v", err)
	}
	facts := filepath.Join(root, scaffold.Dir, "facts.cue")
	raw, err := os.ReadFile(facts)
	if err != nil {
		t.Fatalf("read facts: %v", err)
	}
	audited := strings.Replace(string(raw), "enforcement: \"gate\"", "enforcement: \"audit\"", 1)
	if audited == string(raw) {
		t.Fatal("fixture facts.cue lost the enforcement line — scaffold template changed")
	}
	if err := os.WriteFile(facts, []byte(audited), 0o644); err != nil {
		t.Fatalf("write facts: %v", err)
	}

	t.Run("audit reports but exits 0", func(t *testing.T) {
		out, err := runApp(t, "check", "--root", root, "--json")
		if err != nil {
			t.Fatalf("audit mode must not fail: %v", err)
		}
		var rep check.Report
		if uErr := json.Unmarshal([]byte(out), &rep); uErr != nil {
			t.Fatalf("stdout not JSON: %v\n%s", uErr, out)
		}
		if rep.Enforcement != manifest.EnforcementAudit {
			t.Errorf("report enforcement = %q, want %q", rep.Enforcement, manifest.EnforcementAudit)
		}
		if rep.Stats.Errors == 0 {
			t.Error("audit mode must still COUNT the errors, not hide them")
		}
	})

	t.Run("enforce overrides the audit policy", func(t *testing.T) {
		_, err := runApp(t, "check", "--root", root, "--enforce")
		if !errors.Is(err, errFindings) {
			t.Fatalf("--enforce err = %v, want errFindings", err)
		}
	})
}

const absorbFixtureReport = `{
  "capabilityInfo": [
    {"capabilityName": "FILES", "capabilityType": "CAPABILITY_TYPE_DIRECT", "packageDir": "example.com/fixture/internal/store"},
    {"capabilityName": "NETWORK", "capabilityType": "CAPABILITY_TYPE_DIRECT", "packageDir": "example.com/fixture/internal/store"},
    {"capabilityName": "EXEC", "capabilityType": "CAPABILITY_TYPE_TRANSITIVE", "packageDir": "example.com/fixture/internal/store"},
    {"capabilityName": "REFLECT", "capabilityType": "CAPABILITY_TYPE_TRANSITIVE", "packageDir": "example.com/fixture/internal/store"}
  ]
}`

func absorbFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	scaffoldFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/fixture\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	return root
}

func TestRunAbsorb_WritesCapabilitiesOnly(t *testing.T) {
	root := absorbFixtureRoot(t)
	report := filepath.Join(t.TempDir(), "capslock.json")
	if err := os.WriteFile(report, []byte(absorbFixtureReport), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runApp(t, "absorb", "--root", root, "--capslock", report, "--engine-version", "v0.3.4")
	if err != nil {
		t.Fatalf("absorb: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, "capabilities.cue") || !strings.Contains(out, "v0.3.4") {
		t.Errorf("absorb summary missing facts: %s", out)
	}
	if !strings.Contains(out, "REFLECT") {
		t.Errorf("unmapped capability must be reported loudly, out: %s", out)
	}

	body, err := os.ReadFile(filepath.Join(root, scaffold.Dir, "capabilities.cue"))
	if err != nil {
		t.Fatalf("capabilities.cue missing: %v", err)
	}
	for _, want := range []string{`pkg: "internal/store"`, `direct: ["FILES", "NETWORK"]`, `deep: ["EXEC", "FILES", "NETWORK"]`, `engine:  "capslock"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("capabilities.cue missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"claims:", "baseline:", "sites:"} {
		if strings.Contains(string(body), forbidden) {
			t.Errorf("capabilities.cue must never carry %q", forbidden)
		}
	}

	// The manifest with capabilities must still load and gate green
	// (capabilities are advisory; check ignores them).
	if _, err := runApp(t, "check", "--root", root); err != nil {
		t.Errorf("check after absorb must stay green: %v", err)
	}
}

func TestRunAbsorb_CheckReportsDriftAdvisory(t *testing.T) {
	root := absorbFixtureRoot(t)
	report := filepath.Join(t.TempDir(), "capslock.json")
	if err := os.WriteFile(report, []byte(absorbFixtureReport), 0o644); err != nil {
		t.Fatal(err)
	}

	// Fresh manifest (no capabilities committed): every package is missing.
	out, err := runApp(t, "absorb", "--root", root, "--capslock", report, "--check", "--module", "example.com/fixture")
	if err != nil {
		t.Fatalf("absorb --check must never fail: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, "io-surface/capability-drift") || !strings.Contains(out, "internal/store") {
		t.Errorf("drift report missing capability-drift line, out: %s", out)
	}
	if !strings.Contains(out, "advisory") {
		t.Errorf("drift output must say advisory, out: %s", out)
	}

	// After --write, the same report must produce zero drift.
	if _, err := runApp(t, "absorb", "--root", root, "--capslock", report, "--engine-version", "v0.3.4"); err != nil {
		t.Fatalf("absorb write: %v", err)
	}
	out, err = runApp(t, "absorb", "--root", root, "--capslock", report, "--check", "--module", "example.com/fixture")
	if err != nil {
		t.Fatalf("absorb --check after write: %v", err)
	}
	if strings.Contains(out, "internal/store: committed") {
		t.Errorf("in-sync run must not report store drift, out: %s", out)
	}
}

func TestRunAbsorb_MissingCapslockFlag(t *testing.T) {
	root := absorbFixtureRoot(t)
	_, err := runApp(t, "absorb", "--root", root)
	if err == nil {
		t.Fatal("--capslock is required")
	}
	if exitCode(err) != 2 {
		t.Errorf("usage error exit class = %d, want 2", exitCode(err))
	}
}

// TestRunBaselineAdd pins the #22 contract end to end: the command appends a
// note+date entry to baseline.cue (the machine's only sanctioned judgment-file
// write), the suppressed site drops out of the gate, duplicates are refused,
// and a noteless call is a usage error — the baseline only shrinks because
// every entry must carry its reason.
func TestRunBaselineAdd(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	// Plant an undeclared resolved site to grandfather.
	late := filepath.Join(root, "internal", "store", "late.go")
	if err := os.WriteFile(late, []byte("package store\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n"), 0o644); err != nil {
		t.Fatalf("plant undeclared site: %v", err)
	}

	t.Run("noteless call is a usage error", func(t *testing.T) {
		if _, err := runApp(t, "baseline", "add", "--root", root, "internal/store.Late"); err == nil {
			t.Fatal("--note is required")
		}
	})

	t.Run("appends and suppresses", func(t *testing.T) {
		out, err := runApp(t, "baseline", "add", "--root", root, "internal/store.Late", "--note", "tracked in T-next", "--date", "2026-12-01")
		if err != nil {
			t.Fatalf("baseline add: %v", err)
		}
		if !strings.Contains(out, "internal/store.Late") {
			t.Errorf("confirmation lost the ID: %s", out)
		}
		raw, rErr := os.ReadFile(filepath.Join(root, scaffold.Dir, "baseline.cue"))
		if rErr != nil {
			t.Fatalf("read baseline: %v", rErr)
		}
		text := string(raw)
		for _, want := range []string{"internal/store.Late", "tracked in T-next", "2026-12-01"} {
			if !strings.Contains(text, want) {
				t.Errorf("baseline.cue lost %q", want)
			}
		}
		if !strings.Contains(text, "// baseline.cue — grandfathered") {
			t.Error("the surgical write must preserve the file's comments")
		}

		// The gate is green again: the entry suppresses the site.
		if _, err := runApp(t, "check", "--root", root); err != nil {
			t.Errorf("check after baselining = %v, want clean", err)
		}
	})

	t.Run("duplicate is refused", func(t *testing.T) {
		_, err := runApp(t, "baseline", "add", "--root", root, "internal/store.Late", "--note", "again")
		if err == nil {
			t.Fatal("duplicate entry must fail")
		}
		if !strings.Contains(err.Error(), "already baselined") {
			t.Errorf("error should name the duplicate: %v", err)
		}
	})

	t.Run("unknown ID warns but lands", func(t *testing.T) {
		out, err := runApp(t, "baseline", "add", "--root", root, "ghostpkg.Missing", "--note", "future site")
		if err != nil {
			t.Fatalf("unknown ID must still land: %v", err)
		}
		if !strings.Contains(out, "suppresses nothing") {
			t.Errorf("missing advisory warning: %s", out)
		}
	})
}

// TestExtractStats pins the histogram shape: a header, per-package rows, and
// a TOTAL line whose site count matches the JSON extraction.
func TestExtractStats(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	out, err := runApp(t, "extract", "--root", root, "--stats")
	if err != nil {
		t.Fatalf("extract --stats: %v", err)
	}
	if !strings.Contains(out, "package") || !strings.Contains(out, "TOTAL") {
		t.Errorf("stats lost header/total:\n%s", out)
	}

	jsonOut, err := runApp(t, "extract", "--root", root)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	var doc struct {
		Sites []map[string]any `json:"sites"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &doc); err != nil {
		t.Fatalf("extract JSON: %v", err)
	}
	var total int
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, "TOTAL") {
			if n, _ := fmt.Sscanf(line, "%d", &total); n != 1 {
				t.Fatalf("cannot parse TOTAL line %q as an integer", line)
			}
		}
	}
	if total != len(doc.Sites) {
		t.Errorf("stats TOTAL = %d, JSON sites = %d", total, len(doc.Sites))
	}
}
