package extract

// Suppression tests (superb plan T5.6): the directive filters exactly its
// own rule on exactly its own line neighborhood, and every malformed form
// turns into an ignore-invalid observation instead of a silent swallow.

import (
	"testing"
)

func TestSuppress_FiltersExactlyOneRule(t *testing.T) {
	_, obs, err := ExtractWithObservations("../../testdata/detectors", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	byFn := map[string][]Observation{}
	for _, o := range obs {
		byFn[o.SiteID] = append(byFn[o.SiteID], o)
	}

	cases := []struct {
		fn       string
		wantRule string // empty: nothing must fire
	}{
		{fn: "cmd/app.suppressedAbove", wantRule: ""},
		{fn: "cmd/app.suppressedTrailing", wantRule: ""},
		{fn: "cmd/app.wrongRuleDoesNotSuppress", wantRule: RuleErrDropped},
		{fn: "cmd/app.reasonlessDirectiveDoesNotSuppress", wantRule: RuleErrDropped},
		{fn: "cmd/app.unknownRuleDirectiveDoesNotSuppress", wantRule: RuleErrDropped},
	}
	for _, tc := range cases {
		got := byFn[tc.fn]
		if tc.wantRule == "" {
			if len(got) != 0 {
				t.Errorf("%s: %d observations fired despite a valid directive (%+v)", tc.fn, len(got), got)
			}

			continue
		}
		if len(got) != 1 || got[0].Rule != tc.wantRule {
			t.Errorf("%s: want exactly one %s observation, got %+v", tc.fn, tc.wantRule, got)
		}
	}
}

func TestSuppress_InvalidDirectivesReported(t *testing.T) {
	_, obs, err := ExtractWithObservations("../../testdata/detectors", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	invalid := 0
	for _, o := range obs {
		if o.Rule != RuleIgnoreInvalid {
			continue
		}
		invalid++
		if o.Call != "missing reason" && o.Call != "unknown rule name" {
			t.Errorf("ignore-invalid at %s: unexpected classification %q", o.Location, o.Call)
		}
		if o.Location == "" || o.Evidence == "" {
			t.Errorf("ignore-invalid at %s: location/evidence must be populated (%+v)", o.Location, o)
		}
	}
	if invalid != 2 {
		t.Errorf("ignore-invalid findings = %d, want 2 (reasonless + unknown rule)", invalid)
	}
}

func TestSuppress_PlainExtractUnaffected(t *testing.T) {
	// Plain Extract must stay byte-identical to pre-suppression behavior
	// (D1): no directives collected, no observations emitted, sites equal.
	sites, obs, err := ExtractWithObservations("../../testdata/detectors", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	plain, err := Extract("../../testdata/detectors", Options{})
	if err != nil {
		t.Fatalf("plain extract: %v", err)
	}
	if len(plain) != len(sites) {
		t.Fatalf("site count differs between faces: %d vs %d", len(plain), len(sites))
	}
	if len(obs) == 0 {
		t.Fatal("observation channel empty — suppression scan must not eat detector output wholesale")
	}
}
