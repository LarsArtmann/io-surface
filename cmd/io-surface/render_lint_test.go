package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderLint_StdoutAppendsRadar pins the one-doc-both-faces stdout shape
// (TODO #153): the gate inventory renders first, the lint radar appends as an
// appendix, and the command stays exit-0 (the radar is advisory by policy).
func TestRenderLint_StdoutAppendsRadar(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	out, err := runApp(t, "render", "--root", root, "--lint")
	if err != nil {
		t.Fatalf("render --lint: %v", err)
	}
	if !strings.Contains(out, "# I/O Surface Inventory") {
		t.Error("gate face missing from --lint render")
	}
	if !strings.Contains(out, "## Lint radar (findings)") {
		t.Error("lint appendix missing from --lint render")
	}
	if strings.Index(out, "## Lint radar") < strings.Index(out, "## Taxonomy at a glance") {
		t.Error("appendix must come after the gate sections")
	}
}

func TestRenderLint_StdoutWithoutFlag(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)

	out, err := runApp(t, "render", "--root", root)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "Lint radar") {
		t.Error("appendix leaked into a plain render (flag must opt in)")
	}
}

// TestRenderLint_DocConsistency is the --check contract: the appendix lives
// inside the markers, so freshness covers both faces — but a --check that
// does not repeat the writing invocation's flags must report drift, never
// silently pass on a doc whose block carries a face the check did not render.
func TestRenderLint_DocConsistency(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)
	doc := filepath.Join(root, "INVENTORY.md")
	seed := "# Inventory\n\n<!-- io-surface:begin -->\nstale generated content\n<!-- io-surface:end -->\n"
	if err := os.WriteFile(doc, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed doc: %v", err)
	}

	if _, err := runApp(t, "render", "--root", root, "--doc", doc, "--lint"); err != nil {
		t.Fatalf("render --doc --lint: %v", err)
	}
	data, err := os.ReadFile(doc)
	if err != nil {
		t.Fatalf("read applied doc: %v", err)
	}
	applied := string(data)
	if !strings.Contains(applied, "## Lint radar (findings)") {
		t.Error("appendix missing from applied doc")
	}
	if end := strings.Index(applied, "<!-- io-surface:end -->"); end >= 0 {
		if idx := strings.Index(applied, "## Lint radar"); idx < 0 || idx > end {
			t.Error("appendix landed outside the generated block")
		}
	}

	if _, err := runApp(t, "render", "--root", root, "--doc", doc, "--lint", "--check"); err != nil {
		t.Errorf("matching-flag check on a fresh doc reported drift: %v", err)
	}

	_, err = runApp(t, "render", "--root", root, "--doc", doc, "--check")
	if !errors.Is(err, errFindings) {
		t.Errorf("flag-different check exit = %v, want errFindings (exit 1)", err)
	}
}
