package check

import (
	"os"
	"path/filepath"
	"strings"
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
				Location: []string{"cmd/app/main.go:19"},
				Family:   "stream-copy",
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

// TestRun_UnusedBaselineWarns pins the only-shrinks rule: an entry that
// suppresses nothing today is stale (warning) — keeping it would silently
// swallow a future re-offender under the same canonical key.
func TestRun_UnusedBaselineWarns(t *testing.T) {
	m := fixtureManifest()
	// LoadConfig is declared AND extracts (paired), so its baseline entry
	// suppresses nothing; cmd/app.fetch does suppress an undeclared fixture
	// site and must stay quiet.
	m.Baseline = []manifest.BaselineEntry{
		{ID: "internal/store.LoadConfig", Note: "stale", Date: "2026-11-01", Owner: "infra"},
		{ID: "cmd/app.fetch", Note: "tracked in T1", Date: "2026-01-01"},
	}
	rep := runFixture(t, m, Options{Advisory: false})
	used := findingsByRule(rep, RuleBaselineUnused)
	if len(used) != 1 {
		t.Fatalf("want exactly 1 unused-baseline warning, got %d: %v", len(used), used)
	}
	if used[0].ID != "internal/store.LoadConfig" {
		t.Errorf("unused entry ID = %q, want the LoadConfig entry", used[0].ID)
	}
	if !strings.Contains(used[0].Message, "infra") {
		t.Errorf("owner should surface in the message: %s", used[0].Message)
	}
}

// TestRun_BaselineExpiry pins the owner/expiry contract: an expired entry is
// an ERROR even while it suppresses a real finding; a future expiry stays
// quiet; an omitted expiry never expires.
func TestRun_BaselineExpiry(t *testing.T) {
	m := fixtureManifest()
	m.Baseline = []manifest.BaselineEntry{
		{ID: "cmd/app.fetch", Note: "expired long ago", Date: "2026-01-01", Expires: "1999-01-01", Owner: "infra"},
	}
	rep := runFixture(t, m, Options{Advisory: false})
	expired := findingsByRule(rep, RuleBaselineExpired)
	if len(expired) != 1 {
		t.Fatalf("want exactly 1 baseline-expired error, got %d", len(expired))
	}
	if expired[0].Severity != SeverityError {
		t.Errorf("expired baseline must be an error, got %s", expired[0].Severity)
	}

	m.Baseline[0].Expires = "2999-12-31"
	rep = runFixture(t, m, Options{Advisory: false})
	if n := len(findingsByRule(rep, RuleBaselineExpired)); n != 0 {
		t.Errorf("future expiry must stay quiet, got %d findings", n)
	}
}

// TestRun_ScopeHintNamesTheDirsCause pins the --dirs trap: narrowing the scan
// below the declared surface produces findings that look like manifest rot;
// the report must name the flag as the cause (info, never a failure).
func TestRun_ScopeHintNamesTheDirsCause(t *testing.T) {
	m := fixtureManifest()
	rep := runFixture(t, m, Options{Advisory: false, Dirs: []string{"cmd"}})
	hints := findingsByRule(rep, RuleScopeHint)
	if len(hints) != 1 {
		t.Fatalf("want exactly 1 scope hint, got %d", len(hints))
	}
	if hints[0].Severity != SeverityInfo {
		t.Errorf("scope hint must be info, got %s", hints[0].Severity)
	}
	if !strings.Contains(hints[0].Message, "--dirs") || !strings.Contains(hints[0].Message, "cmd") {
		t.Errorf("hint must name the flag and the scanned dirs: %s", hints[0].Message)
	}

	// The full default scope covers every fixture site: no hint.
	rep = runFixture(t, fixtureManifest(), Options{Advisory: false})
	if n := len(findingsByRule(rep, RuleScopeHint)); n != 0 {
		t.Errorf("default scope must not hint, got %d findings", n)
	}
}

func TestRun_ShadowCapabilityNamesHiddenIO(t *testing.T) {
	m := fixtureManifest()
	m.Capabilities = []manifest.Capability{
		// store declares file-read/file-write/embed: FILES is covered, NETWORK is not.
		{Pkg: "internal/store", Direct: []string{"NETWORK"}, Deep: []string{"FILES", "NETWORK"}},
		// cmd/app has no declared sites in the fixture: direct EXEC is a pure shadow.
		{Pkg: "cmd/app", Direct: []string{"EXEC"}, Deep: []string{"EXEC"}},
	}
	rep := runFixture(t, m, Options{})

	shadow := findingsByRule(rep, RuleShadowCapability)
	if len(shadow) != 2 {
		t.Fatalf("want 2 shadow findings, got %d (%v)", len(shadow), shadow)
	}
	for _, f := range shadow {
		if f.Severity != SeverityInfo {
			t.Errorf("shadow findings are advisory by contract, got %s for %s", f.Severity, f.ID)
		}
	}
	joined := shadow[0].Message + shadow[1].Message
	if !strings.Contains(joined, "NETWORK") || !strings.Contains(joined, "EXEC") {
		t.Errorf("shadow messages must name the uncovered capability, got: %s", joined)
	}
	if !strings.Contains(joined, "listener") || !strings.Contains(joined, "outbound") {
		t.Errorf("shadow message must name the mapped families, got: %s", joined)
	}
	if len(findingsByRule(rep, RuleUndeclared)) != 2 {
		t.Errorf("fixture baseline drifted: want exactly the 2 known undeclared sites, stats: %+v", rep.Stats)
	}
}

func TestRun_ShadowCapabilityCoveredAndBaselined(t *testing.T) {
	m := fixtureManifest()
	m.Capabilities = []manifest.Capability{
		{Pkg: "internal/store", Direct: []string{"FILES"}, Deep: []string{"FILES"}},
		// Deep-only reachability (stdlib linkage) never shadows: every Go
		// package transits the dependency graph, so it carries no signal.
		{Pkg: "internal/store", Deep: []string{"EXEC", "NETWORK"}},
	}
	rep := runFixture(t, m, Options{})
	if got := findingsByRule(rep, RuleShadowCapability); len(got) != 0 {
		t.Fatalf("covered capability must not shadow, got %v", got)
	}

	// A baseline entry keyed on the package suppresses the shadow and stays
	// "used" for the only-shrinks machinery.
	m.Capabilities = []manifest.Capability{{Pkg: "cmd/app", Direct: []string{"EXEC"}, Deep: []string{"EXEC"}}}
	m.Baseline = []manifest.BaselineEntry{{ID: "cmd/app", Note: "engine-only transitives, judged fine", Date: "2026-11-06"}}
	rep = runFixture(t, m, Options{})
	if got := findingsByRule(rep, RuleShadowCapability); len(got) != 0 {
		t.Fatalf("baselined shadow must be suppressed, got %v", got)
	}
	if got := findingsByRule(rep, RuleBaselineUnused); len(got) != 0 {
		t.Fatalf("a shadow-suppressing baseline entry must count as used, got %v", got)
	}
}

func TestRun_ShadowCapabilityUnknownNameIsSkipped(t *testing.T) {
	m := fixtureManifest()
	m.Capabilities = []manifest.Capability{{Pkg: "internal/store", Deep: []string{"CGO"}}}
	rep := runFixture(t, m, Options{})
	if got := findingsByRule(rep, RuleShadowCapability); len(got) != 0 {
		t.Fatalf("unknown names are manifest.Validate's job, got %v", got)
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

// TestRun_StrictDirsFailsOnBlindSpot pins the --strict-dirs contract: the
// scope-hint info upgrades to an error when the operator asks for teeth, so
// a --dirs narrowing that blinds the gate to declared sites fails the run.
func TestRun_StrictDirsFailsOnBlindSpot(t *testing.T) {
	m := fixtureManifest()
	m.Baseline = []manifest.BaselineEntry{
		{ID: "cmd/app.fetch", Note: "known", Date: "2026-01-01"},
		{ID: "cmd/app.openDB", Note: "known", Date: "2026-01-01"},
	}
	hinted := runFixture(t, m, Options{Advisory: false, Dirs: []string{"cmd"}})
	hints := findingsByRule(hinted, RuleScopeHint)
	if len(hints) != 1 || hints[0].Severity != SeverityInfo {
		t.Fatalf("default scope-hint = %+v, want one info", hints)
	}
	if hinted.Failing() {
		t.Error("the hint alone must not fail the gate")
	}

	strict := runFixture(t, m, Options{Advisory: false, Dirs: []string{"cmd"}, StrictDirs: true})
	blinds := findingsByRule(strict, RuleScopeHint)
	if len(blinds) != 1 || blinds[0].Severity != SeverityError {
		t.Fatalf("strict-dirs finding = %+v, want one error", blinds)
	}
	if !strict.Failing() {
		t.Error("--strict-dirs blinding must fail the gate")
	}
}

// TestRun_AllowUnusedBaselineGrace pins the grace flag's exact shape: the
// unused-baseline finding still reports, but strictWarnings (the strict-mode
// counter the CLI consults) stops counting it.
func TestRun_AllowUnusedBaselineGrace(t *testing.T) {
	m := fixtureManifest()
	m.Baseline = append(m.Baseline, manifest.BaselineEntry{ID: "ghost.Never", Note: "stale", Date: "2026-01-01"})
	rep := runFixture(t, m, Options{Advisory: false})
	unused := findingsByRule(rep, RuleBaselineUnused)
	if len(unused) != 1 {
		t.Fatalf("unused-baseline findings = %d, want 1", len(unused))
	}
	if rep.StrictWarnings(false) != rep.Stats.Warnings || rep.Stats.Warnings < 1 {
		t.Errorf("without grace, strictWarnings = %d (stats warnings %d), want equal and >=1", rep.StrictWarnings(false), rep.Stats.Warnings)
	}
	if got := rep.StrictWarnings(true); got != rep.Stats.Warnings-1 {
		t.Errorf("with grace, strictWarnings = %d, want %d (one exempted)", got, rep.Stats.Warnings-1)
	}
}
