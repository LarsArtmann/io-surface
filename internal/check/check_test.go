package check

import (
	"testing"

	"github.com/larsartmann/io-surface/internal/manifest"
)

// fixtureManifest declares a subset of testdata/extract so the checker has both
// paired and unpaired sites to reason about.
func fixtureManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Sites: map[string]manifest.Site{
			// exact pair
			"internal/store.LoadConfig": {
				Location: []string{"internal/store/store.go:9"},
				Family:   "file-read",
				Facts:    manifest.Facts{Extractable: true},
			},
			// canonical pair, different ID form -> id-drift
			"internal/store.Store.WriteTemp": {
				Location: []string{"internal/store/store.go:13"},
				Family:   "file-write",
				Facts:    manifest.Facts{Extractable: true},
			},
			// symbol exists in the file but does no watched I/O -> extraction-blind
			"internal/store.atomicWriteFile": {
				Location: []string{"internal/store/store.go:26"},
				Family:   "file-write",
				Facts:    manifest.Facts{Extractable: true},
			},
			// symbol is gone from the declared file -> stale-claim
			"internal/store.DeletedThing": {
				Location: []string{"internal/store/store.go:1"},
				Family:   "file-read",
				Facts:    manifest.Facts{Extractable: true},
			},
			// resolved embed site, declared and paired exactly
			"internal/store.quotaJSON": {
				Location: []string{"internal/store/store.go:5"},
				Family:   "embed",
				Facts:    manifest.Facts{Extractable: true},
			},
			// visible claim, exempt from extraction
			"internal/store.DocOnly": {
				Location: []string{"internal/store/store.go:1"},
				Family:   "file-read",
				Facts:    manifest.Facts{Extractable: false},
			},
		},
	}
}

func findingsByRule(rep *Report, rule string) []Finding {
	var out []Finding
	for _, f := range rep.Findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}

	return out
}

func runFixture(t *testing.T, m *manifest.Manifest, opts Options) *Report {
	t.Helper()
	rep, err := Run("../../testdata/extract", m, opts)
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	return rep
}

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
