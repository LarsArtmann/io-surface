package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/lint"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/render"
)

// copyFixture copies the shared extraction fixture into dst so a test can
// scaffold and check against a writable tree.
func copyFixture(t *testing.T, dst string) {
	t.Helper()
	src := "../../testdata/extract"
	err := filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
}

func TestWrite_ThenLoadCheckRenderRoundTrip(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, root)

	sites, err := extract.Extract(root, extract.Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	dir, err := Write(root, sites, "2026-11-06", false)
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	m, err := manifest.Load(dir)
	if err != nil {
		t.Fatalf("manifest load: %v", err)
	}
	if errs := m.Validate(); len(errs) > 0 {
		t.Fatalf("scaffolded manifest does not validate: %v", errs)
	}
	wantSites := 0
	for _, s := range sites {
		if s.Family() != "" {
			wantSites++
		}
	}
	if len(m.Sites) != wantSites {
		t.Errorf("declared %d sites, want %d (one per family-bearing extraction)", len(m.Sites), wantSites)
	}

	// Green by construction: the snapshot IS the tree it was taken from.
	rep, err := check.Run(root, m, check.Options{Advisory: false})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if rep.Stats.Errors != 0 || rep.Stats.Warnings != 0 {
		t.Errorf("fresh scaffold should be clean, got %d errors / %d warnings: %+v",
			rep.Stats.Errors, rep.Stats.Warnings, rep.Findings)
	}

	// Green by construction extends to the detector face (plan T4.1): a
	// scaffold can never hand a repo a RED lint day one. Findings ship at
	// info by design (D4) — the fixture tree itself carries legitimate info
	// hits (its bare http.Get exists to exercise that extraction shape) —
	// so the contract is severity-based: zero errors, zero warnings.
	lintRep, err := lint.Run(root, lint.Options{})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	if lintRep.Stats.Errors != 0 || lintRep.Stats.Warnings != 0 {
		t.Errorf("fresh scaffold produced failing-severity detector findings: %+v", lintRep.Findings)
	}

	doc := render.Markdown(m)
	for _, want := range []string{"# I/O Surface Inventory", "## `file-read`", "## `embed`", "Deliberately out of scope", "internal/store.LoadConfig"} {
		if !strings.Contains(doc, want) {
			t.Errorf("rendered doc missing %q", want)
		}
	}
}

func TestWrite_RefusesToClobber(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, root)
	sites, err := extract.Extract(root, extract.Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	_, err = Write(root, sites, "2026-11-06", false)
	if err != nil {
		t.Fatalf("first scaffold: %v", err)
	}
	_, err = Write(root, sites, "2026-11-06", false)
	if err == nil {
		t.Fatal("second scaffold without --force should fail")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v, want an exists error", err)
	}
	if _, err = Write(root, sites, "2026-11-06", true); err != nil {
		t.Fatalf("forced scaffold: %v", err)
	}
}

func TestFiles_AllParse(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, root)
	sites, err := extract.Extract(root, extract.Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	dir, err := Write(root, sites, "2026-11-06", false)
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	// Any syntax error in the emitted CUE surfaces as a load failure.
	if _, err = manifest.Load(dir); err != nil {
		t.Fatalf("emitted CUE does not load: %v", err)
	}
}

// TestWrite_ForceRefreshesFactsOnly is the plant-a-claim proof: an adopting
// repo's hand-written judgment (claims, baseline, schema marker) must survive
// a facts regeneration byte-for-byte, while facts.cue actually refreshes.
func TestWrite_ForceRefreshesFactsOnly(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, root)
	sites, err := extract.Extract(root, extract.Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	dir, err := Write(root, sites, "2026-11-06", false)
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	planted := map[string]string{
		"claims.cue":   "package io_surface\n\nsites: \"internal/store.LoadConfig\": claims: {rationale: \"keeper\"}\n",
		"baseline.cue": "package io_surface\n\nbaseline: [{id: \"cmd/app.fetch\", note: \"tracked in T1\", date: \"2026-11-06\"}]\n",
		"schema.cue":   "package io_surface\n\n// planted schema marker\n",
	}
	for name, content := range planted {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("plant %s: %v", name, err)
		}
	}

	if _, err = Write(root, sites, "2026-11-07", true); err != nil {
		t.Fatalf("forced refresh: %v", err)
	}
	for name, want := range planted {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(data) != want {
			t.Errorf("%s was rewritten by --force:\n got: %q\nwant: %q", name, data, want)
		}
	}
	facts, err := os.ReadFile(filepath.Join(dir, "facts.cue"))
	if err != nil {
		t.Fatalf("read facts.cue: %v", err)
	}
	if !strings.Contains(string(facts), "\"2026-11-07\"") {
		t.Error("facts.cue was not refreshed by --force")
	}
}

// TestAuditCommit_WorkingTreeFallback pins the no-git path: a plain temp dir
// has no repository, so the audit provenance falls back to the literal that
// keeps scaffolded manifests valid anywhere.
func TestAuditCommit_WorkingTreeFallback(t *testing.T) {
	if got := auditCommit(t.TempDir()); got != "working-tree" {
		t.Fatalf("want working-tree fallback, got %q", got)
	}
}

// TestAuditCommit_HeadAndDirty pins the real-repo path: a committed tree
// yields the short HEAD hash, and uncommitted changes append -dirty.
func TestAuditCommit_HeadAndDirty(t *testing.T) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}
	run := func(dir string, args ...string) string {
		cmd := exec.Command(gitBin, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}

		return string(out)
	}
	dir := t.TempDir()
	run(dir, "init", "-q", "-b", "main")
	if got := auditCommit(dir); got != "working-tree" {
		t.Fatalf("repo without commits must fall back, got %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write seed.txt: %v", err)
	}
	run(dir, "add", ".")
	run(dir, "commit", "-q", "-m", "seed")
	head := strings.TrimSpace(run(dir, "rev-parse", "--short", "HEAD"))
	if got := auditCommit(dir); got != head {
		t.Fatalf("want clean-repo hash %q, got %q", head, got)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("y"), 0o644); err != nil {
		t.Fatalf("write extra.txt: %v", err)
	}
	if got := auditCommit(dir); got != head+"-dirty" {
		t.Fatalf("want dirty marker %q, got %q", head+"-dirty", got)
	}
}
