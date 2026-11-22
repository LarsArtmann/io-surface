package render

import (
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-error-family"
	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/manifest"
)

const proseDoc = `# Architecture Notes

Hand-written prose that must never be touched.

` + BeginMarker + `
stale generated content
` + EndMarker + `

More hand-written prose after the block.
`

func TestApplyToDoc_ReplacesOnlyInsideBlock(t *testing.T) {
	got, err := ApplyToDoc(proseDoc, Block(fixtureManifest()))
	if err != nil {
		t.Fatalf("ApplyToDoc: %v", err)
	}
	if strings.Contains(got, "stale generated content") {
		t.Error("stale block content survived the apply")
	}
	if !strings.Contains(got, "## Taxonomy at a glance") {
		t.Error("fresh block content missing")
	}
	for _, keep := range []string{
		"# Architecture Notes",
		"Hand-written prose that must never be touched.",
		"More hand-written prose after the block.",
		BeginMarker,
		EndMarker,
	} {
		if !strings.Contains(got, keep) {
			t.Errorf("prose or marker lost: %q", keep)
		}
	}
	if strings.Contains(got, "# I/O Surface Inventory") {
		t.Error("standalone title leaked into an embedded block")
	}
}

func TestApplyToDoc_Idempotent(t *testing.T) {
	once, err := ApplyToDoc(proseDoc, Block(fixtureManifest()))
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	twice, err := ApplyToDoc(once, Block(fixtureManifest()))
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if once != twice {
		t.Error("ApplyToDoc is not idempotent")
	}
}

func TestApplyToDoc_Errors(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{"no markers", "# Doc\n\nplain prose\n"},
		{"unbalanced begin", "# Doc\n\n" + BeginMarker + "\ncontent without end\n"},
		{"unbalanced end", "# Doc\n\n" + EndMarker + "\n"},
		{"two blocks", BeginMarker + "\n" + EndMarker + "\n" + BeginMarker + "\n" + EndMarker + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ApplyToDoc(tc.doc, Block(fixtureManifest()))
			if err == nil {
				t.Errorf("expected error for %s", tc.name)

				return
			}
			if !errors.Is(err, ErrDocBlock) {
				t.Errorf("error = %v, want ErrDocBlock", err)
			}
			if got := errorfamily.Code(err); got != "render.doc-block" {
				t.Errorf("doc-block code = %q, want render.doc-block", got)
			}
		})
	}
}

func TestCheckDoc_FreshAndStale(t *testing.T) {
	fresh, err := ApplyToDoc(proseDoc, Block(fixtureManifest()))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	stored, want, err := CheckDoc(fresh, Block(fixtureManifest()))
	if err != nil {
		t.Fatalf("check fresh: %v", err)
	}
	if stored != want {
		t.Error("freshly applied block does not match its own render")
	}

	stale := strings.Replace(fresh, "process start", "CHANGED", 1)
	stored, want, err = CheckDoc(stale, Block(fixtureManifest()))
	if err != nil {
		t.Fatalf("check stale: %v", err)
	}
	if stored == want {
		t.Error("stale block falsely reported fresh")
	}
}

func TestBlock_EmptyManifest(t *testing.T) {
	got := Block(&manifest.Manifest{Dir: "x"})
	if strings.Contains(got, "## `") || strings.Contains(got, "Deliberately out of scope") {
		t.Errorf("empty manifest block rendered sections:\n%s", got)
	}
	if !strings.Contains(got, "Taxonomy at a glance") {
		t.Error("block lost the taxonomy header")
	}
}

// TestApplyToDoc_ComposedBlock is the one-doc-both-faces contract (TODO
// #153): the caller composes the gate block plus the lint-radar appendix and
// the splice carries it inside the markers; checking the same composition is
// fresh, and checking a flag-different composition (appendix dropped) is
// drift — the --check invocation must repeat the writing invocation's flags.
func TestApplyToDoc_ComposedBlock(t *testing.T) {
	m := fixtureManifest()
	composed := Block(m) + LintSection(&check.Report{Enforcement: "lint"})

	applied, err := ApplyToDoc(proseDoc, composed)
	if err != nil {
		t.Fatalf("apply composed block: %v", err)
	}
	if !strings.Contains(applied, "## Lint radar (findings)") {
		t.Error("appendix did not land inside the block")
	}
	if !strings.Contains(applied, EndMarker) || strings.Index(applied, "## Lint radar") > strings.Index(applied, EndMarker) {
		t.Error("appendix leaked outside the end marker")
	}

	storedBlock, freshBlock, err := CheckDoc(applied, composed)
	if err != nil {
		t.Fatalf("check composed: %v", err)
	}
	if storedBlock != freshBlock {
		t.Error("freshly applied composed block does not match its own composition")
	}

	gateOnly := Block(m)
	storedBlock, freshBlock, err = CheckDoc(applied, gateOnly)
	if err != nil {
		t.Fatalf("check flag-different: %v", err)
	}
	if storedBlock == freshBlock {
		t.Error("checking without the writing invocation's flags must report drift")
	}
}
