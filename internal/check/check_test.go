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
			// resolved fs-walk site from the second store file, paired exactly
			"internal/store.WalkFiles": {
				Location: []string{"internal/store/walk.go:10"},
				Family:   "file-read",
				Facts:    manifest.Facts{Extractable: true},
			},
			// new-family sites (mmap / bufio / ServeContent), paired exactly
			"internal/store.MmapFile": {
				Location: []string{"internal/store/mapped.go:13", "internal/store/mapped.go:19"},
				Family:   "file-read",
				Facts:    manifest.Facts{Extractable: true},
			},
			"internal/store.Buffered": {
				Location: []string{"internal/store/mapped.go:25"},
				Family:   "stream-copy",
				Facts:    manifest.Facts{Extractable: true},
			},
			"cmd/app.serve": {
				Location: []string{"cmd/app/main.go:23"},
				Family:   "stream-copy",
				Facts:    manifest.Facts{Extractable: true},
			},
			// listenconfig var-ref sites (TODO #19): token tier — declared like
			// any other site; the family is the claim, the extraction is advisory
			"cmd/app.listen": {
				Location: []string{"cmd/app/main.go:28"},
				Family:   "listener",
				Facts:    manifest.Facts{Extractable: true},
			},
			"cmd/app.listenPacket": {
				Location: []string{"cmd/app/main.go:33"},
				Family:   "listener",
				Facts:    manifest.Facts{Extractable: true},
			},
			// generated file: declared like any other site (repos without --ignore
			// declare machine-written I/O too); the ignore-glob path is covered
			// by TestExtract_IgnoreGlobs
			"internal/store.GeneratedSave": {
				Location: []string{"internal/store/zz_generated.go:8"},
				Family:   "file-write",
				Facts:    manifest.Facts{Extractable: true},
			},
			// repo-specific seam: the raw wrapper extracts (resolved); its
			// caller is exempt here because the built-in vocabulary does not
			// know genWrite — TestRun_ManifestDerivedSeamPairs proves the
			// manifest teaches it
			"internal/store.genWrite": {
				Location: []string{"internal/store/gen_seam.go:8"},
				Family:   "file-write",
				Facts:    manifest.Facts{Extractable: true},
			},
			// third-party atomic-write seam: resolved via pkgOf+callFamily,
			// the same pairing as its os.WriteFile predecessors
			"internal/store.SaveCheckpoint": {
				Location: []string{"internal/store/atomicwrite.go:6"},
				Family:   "file-write",
				Facts:    manifest.Facts{Extractable: true},
			},
			// unexported-type method: dotted ID but LEGAL (ttl is a type,
			// not a subpackage) — the permanent id-form false-positive guard
			"internal/store.ttl.Sweep": {
				Location: []string{"internal/store/ttl.go:10"},
				Family:   "file-read",
				Facts:    manifest.Facts{Extractable: true},
			},
			"internal/store.callGenWrite": {
				Location: []string{"internal/store/gen_seam.go:12"},
				Family:   "file-write",
				Facts:    manifest.Facts{Extractable: false},
			},
			// cross-package seam caller: exempt here (resolves only when the
			// manifest vocabulary reaches the extractor via facts.via; pinned by
			// the seam pairing tests)
			"cmd/app.mirror": {
				Location: []string{"cmd/app/main.go:24"},
				Family:   "file-write",
				Facts:    manifest.Facts{Extractable: false},
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
