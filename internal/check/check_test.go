package check

import (
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/match"
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

// TestSortFindings_StrongestFirst pins the presentation contract: errors
// before warnings before infos, regardless of insertion order. Lexical
// severity string order would bury warnings under infos ("error" < "info"
// < "warning") — the exact regression this guards.
func TestSortFindings_StrongestFirst(t *testing.T) {
	fs := []Finding{
		{Rule: RuleScopeHint, Severity: SeverityInfo, ID: "z"},
		{Rule: RuleUndeclared, Severity: SeverityError, ID: "a"},
		{Rule: RuleClaimConflict, Severity: SeverityWarning, ID: "m"},
		{Rule: RuleScopeHint, Severity: SeverityInfo, ID: "a"},
		{Rule: RuleClaimConflict, Severity: SeverityWarning, ID: "a"},
	}
	sortFindings(fs)
	want := []Severity{SeverityError, SeverityWarning, SeverityWarning, SeverityInfo, SeverityInfo}
	for i, f := range fs {
		if f.Severity != want[i] {
			t.Fatalf("position %d: got %s, want %s (full order: %v)", i, f.Severity, want[i], fs)
		}
	}
}

// TestRuntimePowerDirectRule pins the v1.3 split (Q5): DIRECT runtime-power
// reachability reports through runtime-power-direct (info, baseline-
// suppressible), and the SAME caps never fire shadow-capability (they have
// no site families; shadowing them would be a permanent noise floor).
func TestRuntimePowerDirectRule(t *testing.T) {
	m := &manifest.Manifest{
		Capabilities: []manifest.Capability{
			{Pkg: "internal/render", Direct: []string{"REFLECT", "FILES"}, Deep: []string{"REFLECT", "FILES", "NETWORK"}},
			{Pkg: "internal/store", Direct: []string{"FILES"}, Deep: []string{"FILES"}},
		},
	}

	rep := &Report{}
	used := map[string]bool{}
	runtimePowerDirect(m, map[string]bool{}, used, rep)
	shadowCapabilities(m, map[string]bool{}, used, rep)

	var powerFound, storeShadow bool
	for _, f := range rep.Findings {
		switch {
		case f.Rule == RuleRuntimePowerDirect:
			powerFound = true
			if f.ID != "internal/render" {
				t.Errorf("runtime-power fired on %q, want internal/render only", f.ID)
			}
			if f.Severity != SeverityInfo {
				t.Errorf("severity = %q, want info (advisory forever)", f.Severity)
			}
			if !strings.Contains(f.Message, "REFLECT") || strings.Contains(f.Message, "FILES") {
				t.Errorf("message must name the runtime-power caps only: %q", f.Message)
			}
		case f.Rule == RuleShadowCapability && f.ID == "internal/render":
			// FILES still shadows (mapped cap, no covering family); the
			// assertion is that REFLECT never appears in the shadow set.
			if strings.Contains(f.Message, "REFLECT") {
				t.Errorf("shadow message must not name runtime-power caps: %q", f.Message)
			}
		case f.Rule == RuleShadowCapability && f.ID == "internal/store":
			storeShadow = true
		}
	}
	if !powerFound {
		t.Fatal("runtime-power-direct did not fire on a DIRECT REFLECT package")
	}
	if !storeShadow {
		t.Error("mapped cap FILES without a covering family still shadows (semantics unchanged for mapped caps)")
	}

	// Baseline suppression: the canonical-key entry keeps the rule silent
	// AND marks the baseline used (only-shrinks contract).
	baselined := map[string]bool{match.Canonical("internal/render"): true}
	used2 := map[string]bool{}
	rep2 := &Report{}
	runtimePowerDirect(m, baselined, used2, rep2)
	for _, f := range rep2.Findings {
		if f.Rule == RuleRuntimePowerDirect && f.ID == "internal/render" {
			t.Fatal("baselined package must not report runtime-power-direct")
		}
	}
	if !used2[match.Canonical("internal/render")] {
		t.Error("baselined entry must be marked used")
	}
}
