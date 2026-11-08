package check

import (
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/manifest"
	"os"
	"path/filepath"
)

func TestRun_UndeclaredResolvedSitesFail(t *testing.T) {
	rep := runFixture(t, fixtureManifest(), Options{Advisory: false})

	undeclared := findingsByRule(rep, RuleUndeclared)
	got := map[string]bool{}
	for _, f := range undeclared {
		got[f.ID] = true
		if f.Severity != SeverityError {
			t.Errorf("%s severity = %q, want error", f.ID, f.Severity)
		}
	}
	// cmd/app.fetch and cmd/app.openDB are resolved and undeclared.
	for _, want := range []string{"cmd/app.fetch", "cmd/app.openDB"} {
		if !got[want] {
			t.Errorf("missing undeclared finding for %s (got %v)", want, got)
		}
	}
	if !rep.Failing() {
		t.Error("report should fail the gate")
	}
}

func TestRun_BaselineSuppressesUndeclared(t *testing.T) {
	m := fixtureManifest()
	m.Baseline = []manifest.BaselineEntry{
		{ID: "cmd/app.fetch", Note: "tracked in T1", Date: "2026-01-01"},
		{ID: "cmd/app.openDB", Note: "tracked in T1", Date: "2026-01-01"},
	}
	rep := runFixture(t, m, Options{Advisory: false})
	if n := len(findingsByRule(rep, RuleUndeclared)); n != 0 {
		t.Fatalf("expected 0 undeclared findings with a baseline, got %d", n)
	}
	if rep.Stats.Baselined != 2 {
		t.Errorf("baselined stat = %d, want 2", rep.Stats.Baselined)
	}
}

func TestRun_IDDriftIsNotANewSite(t *testing.T) {
	rep := runFixture(t, fixtureManifest(), Options{Advisory: false})
	drift := findingsByRule(rep, RuleIDDrift)
	if len(drift) != 1 {
		t.Fatalf("expected exactly one id-drift, got %d: %+v", len(drift), drift)
	}
	if drift[0].ID != "internal/store.Store.WriteTemp" {
		t.Errorf("drift ID = %q", drift[0].ID)
	}
	if drift[0].Severity != SeverityWarning {
		t.Errorf("drift severity = %q, want warning", drift[0].Severity)
	}
	// The point of drift detection: the extraction finding must NOT also be
	// reported as undeclared.
	for _, f := range findingsByRule(rep, RuleUndeclared) {
		if f.ID == "internal/store.WriteTemp" {
			t.Error("drifted site also reported undeclared")
		}
	}
}

func TestRun_ClassifiesMissingDeclaredSites(t *testing.T) {
	rep := runFixture(t, fixtureManifest(), Options{Advisory: false})

	blind := findingsByRule(rep, RuleExtractionBlind)
	if len(blind) != 1 || blind[0].ID != "internal/store.atomicWriteFile" {
		t.Errorf("extraction-blind = %+v, want internal/store.atomicWriteFile", blind)
	}
	stale := findingsByRule(rep, RuleStaleClaim)
	if len(stale) != 1 || stale[0].ID != "internal/store.DeletedThing" {
		t.Errorf("stale-claim = %+v, want internal/store.DeletedThing", stale)
	}
	for _, f := range findingsByRule(rep, RuleStaleClaim) {
		if f.ID == "internal/store.DocOnly" {
			t.Error("non-extractable site must be exempt from extraction checks")
		}
	}
}

func TestRun_WeakTierIsAdvisoryOnly(t *testing.T) {
	strict := runFixture(t, fixtureManifest(), Options{Advisory: false})
	if n := len(findingsByRule(strict, RuleWeakSite)); n != 0 {
		t.Fatalf("advisory=false still produced %d weak-site findings", n)
	}

	withAdvisory := runFixture(t, fixtureManifest(), Options{Advisory: true})
	weak := findingsByRule(withAdvisory, RuleWeakSite)
	if len(weak) == 0 {
		t.Fatal("advisory=true produced no weak-site findings")
	}
	for _, f := range weak {
		if f.Severity != SeverityInfo {
			t.Errorf("%s severity = %q, want info", f.ID, f.Severity)
		}
	}
	if withAdvisory.Stats.Errors != strict.Stats.Errors {
		t.Error("advisory findings must not change the error count")
	}
}

func TestRun_LocationDriftIsInfoOnly(t *testing.T) {
	m := fixtureManifest()
	site := m.Sites["internal/store.LoadConfig"]
	site.Location = []string{"internal/store/store.go:9", "internal/store/store.go:99"}
	m.Sites["internal/store.LoadConfig"] = site

	rep := runFixture(t, m, Options{Advisory: false})
	drift := findingsByRule(rep, RuleLocationDrift)
	if len(drift) != 1 {
		t.Fatalf("expected one location-drift, got %d", len(drift))
	}
	if drift[0].Severity != SeverityInfo {
		t.Errorf("location-drift severity = %q, want info (line numbers are derived)", drift[0].Severity)
	}
}

// TestRun_DottedSubpackageIDIsAnError pins the dnsblockd adoption lesson: a
// hand-authored ID that encodes a subpackage with a dot (internal/store.cache
// for package internal/store/cache) can never pair with the extractor's
// slash-package form. The gate reports one id-form error naming the correct
// form — and does NOT double-report the extraction as a second undeclared
// site, because that is one authoring mistake, not two.
func TestRun_DottedSubpackageIDIsAnError(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "internal", "store", "cache", "cache.go")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	code := "package cache\n\nimport \"os\"\n\nfunc Load() ([]byte, error) {\n\treturn os.ReadFile(\"cache.bin\")\n}\n"
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &manifest.Manifest{
		Sites: map[string]manifest.Site{
			"internal/store.cache.Load": {
				Location: []string{"internal/store/cache/cache.go:5"},
				Family:   "file-read",
				Facts:    manifest.Facts{Extractable: true},
			},
		},
	}
	rep, err := Run(root, m, Options{Advisory: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	idForm := findingsByRule(rep, RuleIDForm)
	if len(idForm) != 1 {
		t.Fatalf("want exactly 1 id-form finding, got %d: %v", len(idForm), rep.Findings)
	}
	if idForm[0].Severity != SeverityError {
		t.Errorf("id-form must be an error, got %s", idForm[0].Severity)
	}
	if !strings.Contains(idForm[0].Message, "internal/store/cache.Load") {
		t.Errorf("id-form message must name the correct form: %s", idForm[0].Message)
	}
	if undeclared := findingsByRule(rep, RuleUndeclared); len(undeclared) != 0 {
		t.Errorf("the dotted declaration's site must not double-report as undeclared: %v", undeclared)
	}
}

// TestRun_ManifestDerivedSeamPairs pins the end-to-end #20 outcome: a caller
// of a repo-specific wrapper pairs green once the manifest declares it with
// facts.via — no extractor edit, no hardcoded table row.
func TestRun_ManifestDerivedSeamPairs(t *testing.T) {
	m := fixtureManifest()
	m.Sites["internal/store.genWrite"] = manifest.Site{
		Location: []string{"internal/store/gen_seam.go:8"},
		Family:   "file-write",
		Facts:    manifest.Facts{Extractable: true},
	}
	m.Sites["internal/store.callGenWrite"] = manifest.Site{
		Location: []string{"internal/store/gen_seam.go:12"},
		Family:   "file-write",
		Facts:    manifest.Facts{Via: "genWrite"},
	}

	rep := runFixture(t, m, Options{Advisory: false})
	for _, f := range rep.Findings {
		if f.ID == "internal/store.callGenWrite" && f.Rule != RuleLocationDrift {
			t.Errorf("seam caller reported as %s (%s), want only benign location-drift", f.Rule, f.Severity)
		}
	}
}

// TestRun_UnexportedTypeMethodIsLegalID is the permanent fixture guard for
// the id-form rule's narrow scope: a dotted ID whose first segment is an
// UNEXPORTED TYPE (internal/store.ttl.Sweep) is legal and must never produce
// an id-form finding — only a real package dot (internal/store.cache.Load in
// a subpackage dir) is the authoring error.
func TestRun_UnexportedTypeMethodIsLegalID(t *testing.T) {
	rep := runFixture(t, fixtureManifest(), Options{Advisory: false})
	for _, f := range findingsByRule(rep, RuleIDForm) {
		if f.ID == "internal/store.ttl.Sweep" {
			t.Errorf("id-form false-positive on an unexported-type method: %+v", f)
		}
	}
}
