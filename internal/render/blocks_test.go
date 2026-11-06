package render

import (
	"strings"
	"testing"

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
	got, err := ApplyToDoc(proseDoc, fixtureManifest())
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
	once, err := ApplyToDoc(proseDoc, fixtureManifest())
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	twice, err := ApplyToDoc(once, fixtureManifest())
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
			if _, err := ApplyToDoc(tc.doc, fixtureManifest()); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestCheckDoc_FreshAndStale(t *testing.T) {
	fresh, err := ApplyToDoc(proseDoc, fixtureManifest())
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	stored, want, err := CheckDoc(fresh, fixtureManifest())
	if err != nil {
		t.Fatalf("check fresh: %v", err)
	}
	if stored != want {
		t.Error("freshly applied block does not match its own render")
	}

	stale := strings.Replace(fresh, "process start", "CHANGED", 1)
	stored, want, err = CheckDoc(stale, fixtureManifest())
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
