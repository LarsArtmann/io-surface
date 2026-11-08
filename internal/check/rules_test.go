package check

import (
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/manifest"
)

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

// TestClaimConflict pins the discipline-vs-family coherence rule (plan
// T20): "atomic-write" on a file-read site conflicts, the same discipline on
// a file-write site is fine, and family-agnostic disciplines never conflict.
func TestClaimConflict(t *testing.T) {
	m := fixtureManifest()
	s := m.Sites["internal/store.LoadConfig"] // file-read, paired exactly
	s.Claims.Discipline = []string{"atomic-write"}
	m.Sites["internal/store.LoadConfig"] = s
	w := m.Sites["internal/store.Store.WriteTemp"] // file-write, id-drift pair
	w.Claims.Discipline = []string{"atomic-write", "bounded"}
	m.Sites["internal/store.Store.WriteTemp"] = w

	rep := runFixture(t, m, Options{Advisory: false})
	conflicts := findingsByRule(rep, RuleClaimConflict)
	if len(conflicts) != 1 || conflicts[0].ID != "internal/store.LoadConfig" {
		t.Fatalf("claim conflicts = %+v, want exactly internal/store.LoadConfig", conflicts)
	}
	if conflicts[0].Severity != SeverityWarning {
		t.Errorf("claim-conflict severity = %q, want warning", conflicts[0].Severity)
	}
}
