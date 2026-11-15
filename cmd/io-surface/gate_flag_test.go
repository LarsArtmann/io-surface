package main

// gate_flag_test.go — the cmd-level contract of the check gate's tier
// escalation flags. check.Options.StrictWeak has a check-level test
// (TestRun_StrictWeakEscalates in internal/check); this file drives the same
// contract through the real cmdguard execution, so flag parsing, the exit
// code mapping, and the enforcement-policy interplay are locked too. Kept
// out of main_test.go to honor the 650-line file gate.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/scaffold"
)

// scaffoldWithoutWeak copies the shared miniature tree and writes a manifest
// that declares only the resolved-tier sites, leaving the token-tier sites
// undeclared — the shape the strict-weak contract reasons about, driven
// through the CLI. Returns the number of undeclared weak sites so the test
// cannot pass vacuously if the fixture loses its token tier.
func scaffoldWithoutWeak(t *testing.T, root string) int {
	t.Helper()
	if err := os.CopyFS(root, os.DirFS("../../testdata/extract")); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	sites, err := extract.Extract(root, extract.Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	resolved := make([]*extract.Site, 0, len(sites))
	for _, s := range sites {
		if s.Confidence != extract.ConfidenceToken {
			resolved = append(resolved, s)
		}
	}
	weak := len(sites) - len(resolved)
	if weak == 0 {
		t.Fatal("fixture lost its token-tier sites; the strict-weak e2e would be vacuous")
	}
	if _, err = scaffold.Write(root, resolved, "2026-11-06", false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	return weak
}

func countWeakFindings(out string) int {
	return strings.Count(out, "io-surface/weak-site")
}

// TestRunCheck_StrictWeakEscalates pins the CLI contract (ROADMAP Q3): the
// undeclared weak tier stays advisory by default (exit 0, info severity),
// and the opt-in flag escalates the SAME findings to errors that fail the
// gate (exit 1) — identical count, moved severity.
func TestRunCheck_StrictWeakEscalates(t *testing.T) {
	root := t.TempDir()
	weak := scaffoldWithoutWeak(t, root)

	defOut, defErr := runApp(t, "check", "--root", root)
	if defErr != nil {
		t.Fatalf("default run must stay advisory (exit 0), got err %v", defErr)
	}
	defWeak := countWeakFindings(defOut)
	if defWeak != weak {
		t.Fatalf("default run shows %d weak-site findings, want %d", defWeak, weak)
	}
	if strings.Count(defOut, "info    io-surface/weak-site") != weak {
		t.Errorf("default weak-site findings must print at info severity:\n%s", defOut)
	}

	escOut, escErr := runApp(t, "check", "--root", root, "--strict-weak")
	if !errors.Is(escErr, errFindings) {
		t.Fatalf("--strict-weak err = %v, want errFindings (exit 1)", escErr)
	}
	if got := countWeakFindings(escOut); got != weak {
		t.Errorf("--strict-weak changed the finding count: %d, want %d", got, weak)
	}
	if strings.Count(escOut, "error   io-surface/weak-site") != weak {
		t.Errorf("--strict-weak weak-site findings must print at error severity:\n%s", escOut)
	}
}

// TestRunCheck_StrictWeakEnforcementInterplay pins how the flag composes
// with the manifest's recorded policy: audit demotes the escalated errors
// (exit 0 — the recorded posture wins), and --enforce overrides the policy
// so the escalation bites (exit 1).
func TestRunCheck_StrictWeakEnforcementInterplay(t *testing.T) {
	root := t.TempDir()
	scaffoldWithoutWeak(t, root)

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

	if _, err = runApp(t, "check", "--root", root, "--strict-weak"); err != nil {
		t.Errorf("audit policy must demote strict-weak errors to warnings (exit 0), got %v", err)
	}

	if _, err = runApp(t, "check", "--root", root, "--strict-weak", "--enforce"); !errors.Is(err, errFindings) {
		t.Errorf("--enforce must override the audit policy (exit 1), got %v", err)
	}
}

// TestRunCheck_StrictWeakBadFlags pins the usage contract: an unknown flag
// and a malformed bool value are usage errors (exit 2), not gate findings.
func TestRunCheck_StrictWeakBadFlags(t *testing.T) {
	root := t.TempDir()
	scaffoldWithoutWeak(t, root)

	if _, err := runApp(t, "check", "--root", root, "--strict-weakk"); exitCode(err) != 2 {
		t.Errorf("unknown flag exit = %d (err %v), want 2", exitCode(err), err)
	}

	if _, err := runApp(t, "check", "--root", root, "--strict-weak=banana"); exitCode(err) != 2 {
		t.Errorf("malformed bool value exit = %d (err %v), want 2", exitCode(err), err)
	}
}
