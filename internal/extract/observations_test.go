package extract

import (
	"slices"
	"testing"
)

// TestExtractWithObservations_SitesIdentical pins D1: the observing entrypoint
// must produce byte-identical sites to plain Extract — Site's JSON is the
// facts contract, and the capture machinery must never write site state.
func TestExtractWithObservations_SitesIdentical(t *testing.T) {
	plain, err := Extract("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	sites, obs, err := ExtractWithObservations("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract with observations: %v", err)
	}
	if len(plain) != len(sites) {
		t.Fatalf("site count differs: %d vs %d", len(plain), len(sites))
	}
	for i := range plain {
		if plain[i].ID != sites[i].ID || plain[i].Confidence != sites[i].Confidence {
			t.Fatalf("site %d differs: %+v vs %+v", i, plain[i], sites[i])
		}
		if !slices.Equal(plain[i].Locations, sites[i].Locations) ||
			!slices.Equal(plain[i].Calls, sites[i].Calls) ||
			!slices.Equal(plain[i].Families, sites[i].Families) {
			t.Fatalf("site %s slices differ", plain[i].ID)
		}
	}
	if obs == nil {
		t.Fatal("observations slice must be non-nil (empty allowed)")
	}
}

// TestExtractWithObservations_TokenTierNeverCaptured pins D3: method-token
// and seam sites produce zero observations — only the resolved branch hands a
// capture to the detectors. With the registry empty this asserts the channel
// stays empty; once detectors land they must never observe token sites.
func TestExtractWithObservations_TokenTierNeverCaptured(t *testing.T) {
	_, obs, err := ExtractWithObservations("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	sites, _, err := ExtractWithObservations("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	tokenIDs := map[string]bool{}
	for _, s := range sites {
		if s.Confidence == ConfidenceToken {
			tokenIDs[s.ID] = true
		}
	}
	for _, o := range obs {
		if tokenIDs[o.SiteID] {
			t.Errorf("observation %s/%s on token-tier site %s — D3 violated", o.Rule, o.Location, o.SiteID)
		}
	}
}

// TestObservations_Deterministic pins the ordering contract: two runs over
// the same tree yield identical observation sequences (golden stability).
func TestObservations_Deterministic(t *testing.T) {
	_, first, err := ExtractWithObservations("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	_, second, err := ExtractWithObservations("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("observation count differs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("observation %d differs: %+v vs %+v", i, first[i], second[i])
		}
	}
}
