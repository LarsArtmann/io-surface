package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
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
