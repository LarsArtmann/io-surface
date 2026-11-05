package extract

import (
	"testing"
)

func siteByID(t *testing.T, sites []*Site, id string) *Site {
	t.Helper()
	for _, s := range sites {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("site %q not extracted; got %d sites", id, len(sites))

	return nil
}

func TestExtract_FamiliesAndConfidence(t *testing.T) {
	sites, err := Extract("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	tests := []struct {
		name     string
		id       string
		family   string
		wantCall string
		conf     string
	}{
		{"resolved file read", "internal/store.LoadConfig", "file-read", "os.ReadFile", ConfidenceResolved},
		{"resolved file write", "internal/store.WriteTemp", "file-write", "os.CreateTemp", ConfidenceResolved},
		{"resolved outbound", "cmd/app.fetch", "outbound", "http.Get", ConfidenceResolved},
		{"resolved database", "cmd/app.openDB", "database", "sql.Open", ConfidenceResolved},
		{"seam ident is weak tier", "internal/store.persistAll", "file-write", "atomicWriteFile", ConfidenceToken},
		{"method token is weak tier", "internal/store.Server.Serve", "listener", ".ListenAndServe", ConfidenceToken},
		{"embed var", "internal/store.quotaJSON", "embed", "go:embed quota.json", ConfidenceResolved},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := siteByID(t, sites, tc.id)
			if got := s.Family(); got != tc.family {
				t.Errorf("family = %q, want %q (families=%v)", got, tc.family, s.Families)
			}
			if !contains(s.Calls, tc.wantCall) {
				t.Errorf("calls = %v, want %q", s.Calls, tc.wantCall)
			}
			if s.Confidence != tc.conf {
				t.Errorf("confidence = %q, want %q", s.Confidence, tc.conf)
			}
			if len(s.Locations) == 0 {
				t.Error("no locations recorded")
			}
		})
	}
}

func TestExtract_IDsAreRepoRelative(t *testing.T) {
	sites, err := Extract("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	// The manifest contract keys on the repo-relative package path; a walk-root
	// relative ID would never match a manifest entry.
	s := siteByID(t, sites, "internal/store.LoadConfig")
	if len(s.Locations) != 1 || s.Locations[0] != "internal/store/store.go:9" {
		t.Errorf("locations = %v, want [internal/store/store.go:9]", s.Locations)
	}
}

func TestExtract_Deterministic(t *testing.T) {
	first, err := Extract("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	second, err := Extract("../../testdata/extract", Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("site count differs between runs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Fatalf("order differs at %d: %q vs %q", i, first[i].ID, second[i].ID)
		}
	}
}

func TestExtract_MissingDirsIsNotAnError(t *testing.T) {
	sites, err := Extract("../../testdata/does-not-exist", Options{})
	if err != nil {
		t.Fatalf("extract on empty tree: %v", err)
	}
	if len(sites) != 0 {
		t.Errorf("expected no sites, got %d", len(sites))
	}
}
