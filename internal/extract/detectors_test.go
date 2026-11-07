package extract

import (
	"strings"
	"testing"
)

// observationsBySite collects the observations for one rule, keyed by the
// enclosing symbol ID.
func observationsBySite(t *testing.T, root, rule string) map[string][]Observation {
	t.Helper()
	_, obs, err := ExtractWithObservations(root, Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	out := map[string][]Observation{}
	for _, o := range obs {
		if o.Rule == rule {
			out[o.SiteID] = append(out[o.SiteID], o)
		}
	}

	return out
}

// TestDetectErrDropped_ShapeTable is the positive/negative contract for the
// err-dropped rule (plan T3.6): exactly the bug shapes fire, exactly the
// deliberate/handled shapes stay silent.
func TestDetectErrDropped_ShapeTable(t *testing.T) {
	got := observationsBySite(t, "../../testdata/detectors", RuleErrDropped)

	tests := []struct {
		site      string
		wantCount int
		wantInEv  string
	}{
		{"cmd/app.positiveBare", 1, "os.Remove(path)"},
		{"cmd/app.positiveBlank", 1, "data, _ := os.ReadFile(path)"},
		{"cmd/app.positiveBlankStat", 1, "info, _ := os.Stat(path)"},
	}
	for _, tc := range tests {
		obs := got[tc.site]
		if len(obs) != tc.wantCount {
			t.Errorf("%s: %d observations, want %d (all: %+v)", tc.site, len(obs), tc.wantCount, obs)

			continue
		}
		if !strings.Contains(obs[0].Evidence, tc.wantInEv) {
			t.Errorf("%s: evidence %q, want it to contain %q", tc.site, obs[0].Evidence, tc.wantInEv)
		}
		if obs[0].Call != "os.Remove" && obs[0].Call != "os.ReadFile" && obs[0].Call != "os.Stat" {
			t.Errorf("%s: call = %q", tc.site, obs[0].Call)
		}
		if !strings.HasPrefix(obs[0].Location, "cmd/app/main.go:") {
			t.Errorf("%s: location %q not repo-relative main.go", tc.site, obs[0].Location)
		}
	}

	negatives := []string{
		"cmd/app.negativeDeliberate", // D8: _ = f()
		"cmd/app.negativeAllBlank",   // D8: _, _ = f()
		"cmd/app.negativeHandled",    // err kept
		"cmd/app.negativePropagate",  // return f()
		"cmd/app.negativeNamed",      // data, err := f()
		"cmd/app.negativeDeferred",   // defer cleanup
		"cmd/app.negativeExpression", // consumed by comparison
	}
	for _, site := range negatives {
		if obs := got[site]; len(obs) != 0 {
			t.Errorf("%s: must stay silent, got %+v", site, obs)
		}
	}
}

// TestDetectorRegistry_ParityPinsMetadata pins the two halves of the rule
// contract together: every registered detector has metadata and every
// metadata entry has a detector. A rule that ships without a message builder
// would crash the lint face; metadata without a detector would lie about the
// rule vocabulary.
func TestDetectorRegistry_ParityPinsMetadata(t *testing.T) {
	for _, d := range detectors {
		if _, ok := DetectorRule(d.rule); !ok {
			t.Errorf("detector %q has no detectorRules metadata", d.rule)
		}
	}
	for _, d := range fnDetectors {
		if _, ok := DetectorRule(d.rule); !ok {
			t.Errorf("fnDetector %q has no detectorRules metadata", d.rule)
		}
	}
	for _, r := range DetectorRules() {
		found := false
		for _, d := range detectors {
			if d.rule == r.Name {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("detectorRules entry %q has no registered detector", r.Name)
		}
		if r.Severity != "info" {
			t.Logf("rule %s ships at %s (promotion recorded via T17 evidence)", r.Name, r.Severity)
		}
		if r.Message == nil {
			t.Errorf("rule %q has no Message builder", r.Name)
		}
	}
}

// TestErrResults_CoversWatchedFamilies is a smoke table: the calls the fleet
// scan named as dominant (os.ReadFile 821, os.Stat 825, os.WriteFile 405,
// os.MkdirAll 350, http.Get, sql.Open) must all be in the error table.
func TestErrResults_CoversWatchedFamilies(t *testing.T) {
	for _, call := range []string{
		"os.ReadFile", "os.Stat", "os.WriteFile", "os.MkdirAll", "os.Create",
		"io.ReadAll", "http.Get", "sql.Open", "net.Listen", "net.Dial", "filepath.WalkDir",
	} {
		if _, ok := errResults[call]; !ok {
			t.Errorf("errResults missing %q", call)
		}
	}
	if _, ok := errResults["os.TempDir"]; ok {
		t.Errorf("os.TempDir returns no error and must not be in errResults")
	}
}

// TestExtractWithObservations_TokenSitesNeverObserved pins D3 on the
// detectors tree too: no observation may attach to a token-tier site.
func TestExtractWithObservations_TokenSitesNeverObserved(t *testing.T) {
	sites, obs, err := ExtractWithObservations("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	token := map[string]bool{}
	for _, s := range sites {
		if s.Confidence == ConfidenceToken {
			token[s.ID] = true
		}
	}
	for _, o := range obs {
		if token[o.SiteID] {
			t.Errorf("D3 violated: %s/%s on token site %s", o.Rule, o.Location, o.SiteID)
		}
	}
}

// TestDetectCloserMissing_OwnershipTable pins the closer-missing contract:
// never-closed-and-never-handed-off fires (even when the value is used),
// while Close, return, call-argument hand-off, storage, and blank slots stay
// silent.
func TestDetectCloserMissing_OwnershipTable(t *testing.T) {
	got := observationsBySite(t, "../../testdata/detectors", RuleCloserMissing)

	positives := []string{"internal/store.leakNeverUsed", "internal/store.leakUsedNotClosed"}
	for _, site := range positives {
		if len(got[site]) != 1 {
			t.Errorf("%s: %d closer-missing observations, want 1 (all: %+v)", site, len(got[site]), got[site])
		}
	}

	negatives := []string{
		"internal/store.positiveDeferred",
		"internal/store.positivePlainClose",
		"internal/store.negativeReturned",
		"internal/store.negativeHandedOff",
		"internal/store.negativeStored",
		"internal/store.negativeBlank",
		"internal/store.negativeDBOpen",
	}
	for _, site := range negatives {
		if obs := got[site]; len(obs) != 0 {
			t.Errorf("%s: must stay silent, got %+v", site, obs)
		}
	}
}

// TestDetectCloserMissing_HttpBody pins the resp.Body variant: body reads
// alone are the classic leak (io.ReadAll does not close), defer and plain
// Close silence, whole-response return or hand-off transfers ownership.
func TestDetectCloserMissing_HttpBody(t *testing.T) {
	got := observationsBySite(t, "../../testdata/detectors", RuleCloserMissing)

	positives := []string{"cmd/app.leakBody", "cmd/app.statusOnlyLeak"}
	for _, site := range positives {
		if len(got[site]) != 1 {
			t.Errorf("%s: %d observations, want 1 (all: %+v)", site, len(got[site]), got[site])
		}
	}

	negatives := []string{
		"cmd/app.closedBody",
		"cmd/app.plainCloseBody",
		"cmd/app.transferredResp",
		"cmd/app.handedOffResp",
	}
	for _, site := range negatives {
		if obs := got[site]; len(obs) != 0 {
			t.Errorf("%s: must stay silent, got %+v", site, obs)
		}
	}
}

// TestDetectCtxDropped_ScopeGate pins the ctx rule: variant pairs fire only
// when the enclosing function takes a context.Context; no-ctx functions,
// receiver-typed params, and already-context-aware calls stay silent.
func TestDetectCtxDropped_ScopeGate(t *testing.T) {
	got := observationsBySite(t, "../../testdata/detectors", RuleCtxDropped)

	positives := []string{"cmd/app.positiveExec", "cmd/app.positiveDial", "cmd/app.positiveDialTimeout", "cmd/app.positiveGet"}
	for _, site := range positives {
		if len(got[site]) != 1 {
			t.Errorf("%s: %d observations, want 1 (all: %+v)", site, len(got[site]), got[site])
		}
	}
	for _, site := range positives {
		for _, o := range got[site] {
			if !strings.Contains(o.Evidence, "context.Context") {
				t.Errorf("%s: evidence lacks the variant hint: %q", site, o.Evidence)
			}
		}
	}

	negatives := []string{
		"cmd/app.negativeNoCtx",       // no ctx in scope
		"cmd/app.negativeReceiverCtx", // param is not context.Context
		"cmd/app.negativeCtxVariant",  // already uses DialContext
	}
	for _, site := range negatives {
		if obs := got[site]; len(obs) != 0 {
			t.Errorf("%s: must stay silent, got %+v", site, obs)
		}
	}
}
