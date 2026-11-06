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

// TestPkgOf_CoversEveryWatchedPackage pins the bufio lesson: a callFamily
// entry whose package key no import can resolve to is dead code — the call
// sites silently never extract, and only an end-to-end fixture test catches
// it. Every watched package key must be reachable through pkgOf (exact) or
// importSubstring (fragment).
func TestPkgOf_CoversEveryWatchedPackage(t *testing.T) {
	resolvable := map[string]bool{}
	for _, key := range pkgOf {
		if key != "" {
			resolvable[key] = true
		}
	}
	for _, key := range importSubstring {
		resolvable[key] = true
	}
	for pkgKey := range callFamily {
		if !resolvable[pkgKey] {
			t.Errorf("callFamily[%q] is unreachable: add %q to pkgOf (or an importSubstring fragment), otherwise its call sites never extract", pkgKey, pkgKey)
		}
	}
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
		{"resolved fs walk", "internal/store.WalkFiles", "file-read", "filepath.WalkDir", ConfidenceResolved},
		{"seam ident is weak tier", "internal/store.persistAll", "file-write", "atomicWriteFile", ConfidenceToken},
		{"method token is weak tier", "internal/store.Server.Serve", "listener", ".ListenAndServe", ConfidenceToken},
		{"embed var", "internal/store.quotaJSON", "embed", "go:embed quota.json", ConfidenceResolved},
		{"resolved mmap", "internal/store.MmapFile", "file-read", "syscall.Mmap", ConfidenceResolved},
		{"resolved bufio wrapper", "internal/store.Buffered", "stream-copy", "bufio.NewReader", ConfidenceResolved},
		{"resolved serve content", "cmd/app.serve", "stream-copy", "http.ServeContent", ConfidenceResolved},
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

func TestSiteFamily_Precedence(t *testing.T) {
	tests := []struct {
		name     string
		families []string
		want     string
	}{
		{"writer beats reader", []string{"file-read", "file-write"}, "file-write"},
		{"listener beats read", []string{"file-read", "listener"}, "listener"},
		{"db beats read", []string{"file-read", "database"}, "database"},
		{"stream beats read", []string{"file-read", "stream-copy"}, "stream-copy"},
		{"single family", []string{"entropy"}, "entropy"},
		{"unclassified", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Site{Families: tc.families}
			if got := s.Family(); got != tc.want {
				t.Errorf("Family() = %q, want %q", got, tc.want)
			}
		})
	}
}
