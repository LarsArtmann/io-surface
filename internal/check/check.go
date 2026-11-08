// Package check implements the bidirectional io-surface gate.
//
// The four directions, in the standard's order:
//
//  1. declared-sites-exist    — every declared site still resolves in the tree
//     (rule: stale-claim) and its locations agree with the extraction
//     (rule: location-drift)
//  2. undeclared-sites-absent — every high-confidence extraction finding is
//     declared or baselined (rule: undeclared-site)
//  3. constraints hold        — enforced by CUE at load time (disciplines.cue)
//     plus the schema checks in manifest.Validate
//  4. published-artifact fresh — the renderer's job
//
// Direction 2 is the one that makes the manifest complete: before it, a silent
// manifest passed every gate. Only ConfidenceResolved sites may fail; the weak
// token tier is advisory by construction.
//
// Pairing is done per canonical group (package + base symbol), so a method that
// became a free function is drift, not a new site — while two DIFFERENT symbols
// that share a base name in one package (AllowlistManager.Load and
// PermanentAllowlist.Load) are never silently merged.
package check

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/larsartmann/io-surface/internal/capability"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/match"
)

// Rule identifiers. These are stable: they become SARIF ruleIds and
// suppression keys, so renaming one is a breaking change.
const (
	RuleUndeclared         = "io-surface/undeclared-site"
	RuleStaleClaim         = "io-surface/stale-claim"
	RuleIDDrift            = "io-surface/id-drift"
	RuleIDForm             = "io-surface/id-form"
	RuleLocationDrift      = "io-surface/location-drift"
	RuleMissingLocation    = "io-surface/missing-location"
	RuleExtractionBlind    = "io-surface/extraction-blind"
	RuleWeakSite           = "io-surface/weak-site"
	RuleAmbiguousMatch     = "io-surface/ambiguous-match"
	RuleBaselineUnused     = "io-surface/unused-baseline"
	RuleBaselineExpired    = "io-surface/baseline-expired"
	RuleScopeHint          = "io-surface/scope-hint"
	RuleShadowCapability   = "io-surface/shadow-capability"
	RuleManifestValidation = "io-surface/manifest-validation"
	RuleClaimConflict      = "io-surface/claim-conflict"
)

// Severity of a finding.
type Severity string

// Severities, strongest first.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Finding is one gate result.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	ID       string   `json:"id"`
	Family   string   `json:"family,omitempty"`
	Location string   `json:"location,omitempty"`
	Message  string   `json:"message"`
}

// Stats are the counters a caller prints in a summary line.
type Stats struct {
	Declared  int `json:"declared"`
	Extracted int `json:"extracted"`
	Resolved  int `json:"resolved"`
	Token     int `json:"token"`
	Baselined int `json:"baselined"`
	Errors    int `json:"errors"`
	Warnings  int `json:"warnings"`
	Infos     int `json:"infos"`
}

// Report is the whole gate result.
type Report struct {
	Stats    Stats     `json:"stats"`
	Findings []Finding `json:"findings"`
	// Enforcement is the manifest's recorded gate policy (D4): "gate" or
	// "audit". Machine consumers read the policy from the report itself, so
	// an audit-mode run is visible in CI logs and SARIF/JSON output without
	// re-reading the manifest.
	Enforcement string `json:"enforcement"`
}

// Failing reports whether the report blocks a merge (any error-severity
// finding).
func (r *Report) Failing() bool { return r.Stats.Errors > 0 }

// StrictWarnings counts the warnings --strict promotes to failure, honoring
// the unused-baseline grace flag: the stale-suppression finding still
// reports, but it stops failing while a team works the baseline down.
func (r *Report) StrictWarnings(allowUnusedBaseline bool) int {
	if !allowUnusedBaseline {
		return r.Stats.Warnings
	}
	n := r.Stats.Warnings
	for _, f := range r.Findings {
		if f.Rule == RuleBaselineUnused && f.Severity == SeverityWarning {
			n--
		}
	}

	return n
}

// Options controls the check.
type Options struct {
	// Dirs limits extraction (default cmd, internal).
	Dirs []string
	// Advisory includes weak-tier findings (default true).
	Advisory bool
	// Ignore skips files matching these extraction globs (generated code);
	// kept in lockstep with extract.Options.Ignore.
	Ignore []string
	// StrictDirs upgrades the scope-hint (declared sites outside --dirs)
	// from info to error: a narrowing that blinds the gate to declared sites
	// fails instead of hinting. The default stays advisory because a narrower
	// scan is often deliberate (module-scoped runs).
	StrictDirs bool
	// AllowUnusedBaseline exempts unused-baseline findings from strict-mode
	// failure: a stale suppression warns, but teams mid-cleanup can opt out
	// of the teeth while the baseline shrinks. The finding still reports.
	AllowUnusedBaseline bool
}

// bucket is one canonical group: every declared ID and every extracted site
// that share a package and base symbol.
type bucket struct {
	declared  []string
	extracted []*extract.Site
}

// ValidationReport builds the report for a manifest that failed Validate.
// Extraction is skipped on purpose: an invalid manifest must not be gated,
// it must be fixed — but the caller still gets a full report so machine
// consumers (CI, SARIF) see one uniform shape for every gate outcome.
func ValidationReport(m *manifest.Manifest, errs []error) *Report {
	rep := &Report{Enforcement: m.EnforcementPolicy()}
	rep.Stats.Declared = len(m.Sites)
	for _, e := range errs {
		rep.add(Finding{
			Rule:     RuleManifestValidation,
			Severity: SeverityError,
			ID:       "manifest",
			Location: m.Dir,
			Message:  e.Error(),
		})
	}

	return rep
}

// Run extracts the tree under root and compares it against the manifest.
func Run(root string, m *manifest.Manifest, opts Options) (*Report, error) {
	sites, err := extract.Extract(root, extract.Options{Dirs: opts.Dirs, Ignore: opts.Ignore, Seams: manifestSeams(m)})
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", root, err)
	}

	rep := &Report{Enforcement: m.EnforcementPolicy()}
	rep.Stats.Declared = len(m.Sites)
	rep.Stats.Baselined = len(m.Baseline)
	rep.Stats.Extracted = len(sites)
	checkClaimConflicts(m, rep)

	buckets := map[string]*bucket{}
	get := func(canon string) *bucket {
		b, ok := buckets[canon]
		if !ok {
			b = &bucket{}
			buckets[canon] = b
		}

		return b
	}
	for _, id := range m.IDs() {
		b := get(match.Canonical(id))
		b.declared = append(b.declared, id)
	}
	for _, s := range sites {
		b := get(match.Canonical(s.ID))
		b.extracted = append(b.extracted, s)
		if s.Confidence == extract.ConfidenceResolved {
			rep.Stats.Resolved++
		} else {
			rep.Stats.Token++
		}
	}
	baselined := map[string]bool{}
	for _, id := range m.BaselineIDs() {
		baselined[match.Canonical(id)] = true
	}
	usedBaseline := map[string]bool{}
	// Extracted IDs by exact form: the id-form rule needs to know whether the
	// slash-package variant of a declared ID actually extracts.
	extractedByID := map[string]bool{}
	for _, s := range sites {
		extractedByID[s.ID] = true
	}
	// Declared IDs by exact form: an extraction whose dotted variant is
	// declared is the id-form finding's root cause, not a second one.
	declaredByID := map[string]bool{}
	for _, id := range m.IDs() {
		declaredByID[id] = true
	}

	// The bidirectional pairing walk (directions 1+2) lives in pairing.go:
	// one self-contained phase over the buckets assembled above. All rule
	// firings and their order are unchanged — the split is mechanical (TODO #64).
	p := &pairer{
		root:          root,
		m:             m,
		opts:          opts,
		rep:           rep,
		buckets:       buckets,
		baselined:     baselined,
		usedBaseline:  usedBaseline,
		extractedByID: extractedByID,
		declaredByID:  declaredByID,
	}
	p.pairGroups()

	// Scope hint: a --dirs narrowing that excludes declared sites produces
	// findings that look like manifest rot but are really the flag's doing.
	// Name the cause so the fix is one edit instead of an archaeology dig.
	// StrictDirs gives the hint teeth: blinding the gate to declared sites
	// is then a failure, not a note.
	if outside := declaredOutside(m, effectiveDirs(opts.Dirs)); outside > 0 {
		scopeSev := SeverityInfo
		if opts.StrictDirs {
			scopeSev = SeverityError
		}
		rep.add(Finding{
			Rule: RuleScopeHint, Severity: scopeSev, ID: "--dirs",
			Message: fmt.Sprintf("%d declared site(s) lie outside the scanned dirs %s — this run cannot see them; widen --dirs or refresh facts.cue", outside, strings.Join(effectiveDirs(opts.Dirs), ", ")),
		})
	}

	// Shadow capabilities: the reachability engine says a package's OWN code
	// can do I/O in a class the manifest declares no site for. This is the
	// hole neither engine fills alone: the hermetic extractor cannot see
	// transitive reachability, and the reachability engine cannot name a call
	// site. The gate names the package; claims justify it or baseline
	// grandfathers it.
	//
	// The rule fires on DIRECT capabilities only. Deep (transitive through
	// dependencies) reaches into every Go package via the stdlib — universal
	// linkage, not hidden I/O — so shadowing on it would be a permanent noise
	// floor (verified by the first real run: every package shadowed, zero
	// signal). Direct edges carry the signal; they also carry the engine's
	// conflation phantoms, which is exactly why this tier is advisory (info
	// forever unless an enforcement policy says otherwise) — the blind-spot
	// run measured three false-positive classes in one small repo.
	shadowCapabilities(m, baselined, usedBaseline, rep)

	// Baseline lifecycle: the baseline only shrinks, so an entry that
	// suppresses nothing today is a stale suppression (warning — it would
	// silently swallow a future re-offender under the same canonical key),
	// and an expired entry is a broken contract (error).
	today := time.Now().UTC().Format("2006-01-02")
	for _, entry := range m.Baseline {
		if !usedBaseline[match.Canonical(entry.ID)] {
			rep.add(Finding{
				Rule: RuleBaselineUnused, Severity: SeverityWarning, ID: entry.ID,
				Message: fmt.Sprintf("baseline entry (added %s) suppresses nothing today — remove it: only-shrinks means stale suppressions leave (owner: %s)", entry.Date, ownerLabel(entry.Owner)),
			})
		}
		if entry.Expires != "" && entry.Expires < today {
			rep.add(Finding{
				Rule: RuleBaselineExpired, Severity: SeverityError, ID: entry.ID,
				Message: fmt.Sprintf("baseline entry expired %s — fix the site or re-baseline deliberately (owner: %s)", entry.Expires, ownerLabel(entry.Owner)),
			})
		}
	}

	sort.SliceStable(rep.Findings, func(i, j int) bool {
		if rep.Findings[i].Severity != rep.Findings[j].Severity {
			return rep.Findings[i].Severity < rep.Findings[j].Severity
		}
		if rep.Findings[i].Rule != rep.Findings[j].Rule {
			return rep.Findings[i].Rule < rep.Findings[j].Rule
		}

		return rep.Findings[i].ID < rep.Findings[j].ID
	})

	return rep, nil
}

// shadowCapabilities emits one info per package whose DIRECT reachability
// facts name a capability class the package declares no mapped-family site
// for. Suppression is the ordinary baseline: an entry keyed on the package
// marks the shadow deliberate and keeps the entry alive for only-shrinks.
func shadowCapabilities(m *manifest.Manifest, baselined map[string]bool, usedBaseline map[string]bool, rep *Report) {
	declaredFamilies := map[string]map[string]bool{}
	for _, site := range m.Sites {
		if len(site.Location) == 0 {
			continue
		}
		path, _, _ := strings.Cut(filepath.ToSlash(site.Location[0]), ":")
		pkg := path[:strings.LastIndex(path, "/")+1]
		pkg = strings.TrimSuffix(pkg, "/")
		if declaredFamilies[pkg] == nil {
			declaredFamilies[pkg] = map[string]bool{}
		}
		declaredFamilies[pkg][site.Family] = true
	}

	for _, cap := range m.Capabilities {
		if baselined[match.Canonical(cap.Pkg)] {
			usedBaseline[match.Canonical(cap.Pkg)] = true

			continue
		}
		var shadowed []string
		for _, name := range cap.Direct {
			cn := capability.Name(name)
			if !capability.Valid(name) {
				continue // manifest.Validate already reports unknown names
			}
			covered := false
			for _, fam := range capability.FamiliesFor(cn) {
				if declaredFamilies[cap.Pkg][fam] {
					covered = true
					break
				}
			}
			if !covered {
				shadowed = append(shadowed, fmt.Sprintf("%s (needs one of: %s)", name, strings.Join(capability.FamiliesFor(cn), ", ")))
			}
		}
		if len(shadowed) == 0 {
			continue
		}
		sort.Strings(shadowed)
		rep.add(Finding{
			Rule: RuleShadowCapability, Severity: SeverityInfo, ID: cap.Pkg,
			Message: fmt.Sprintf("reachability engine observes %s in this package, but no declared site covers those families — declare the hidden I/O, or baseline this package deliberately",
				strings.Join(shadowed, ", ")),
		})
	}
}

// ownerLabel renders an optional baseline owner for finding messages.
func ownerLabel(owner string) string {
	if owner == "" {
		return "unowned"
	}

	return owner
}

// effectiveDirs resolves the scan scope exactly as the extractor defaults it:
// an empty --dirs means cmd and internal.
func effectiveDirs(dirs []string) []string {
	if len(dirs) == 0 {
		return []string{"cmd", "internal"}
	}

	return dirs
}

// declaredOutside counts declared sites whose first location path does not
// fall under any of the scanned dirs (path-prefix match on slash form).
func declaredOutside(m *manifest.Manifest, dirs []string) int {
	under := func(path string) bool {
		for _, d := range dirs {
			d = strings.TrimPrefix(strings.TrimSuffix(filepath.ToSlash(d), "/"), "./")
			if path == d || strings.HasPrefix(path, d+"/") {
				return true
			}
		}

		return false
	}
	count := 0
	for _, s := range m.Sites {
		if len(s.Location) == 0 {
			continue
		}
		path, _, _ := strings.Cut(filepath.ToSlash(s.Location[0]), ":")
		if !under(path) {
			count++
		}
	}

	return count
}

func (r *Report) add(f Finding) {
	r.Findings = append(r.Findings, f)
	switch f.Severity {
	case SeverityError:
		r.Stats.Errors++
	case SeverityWarning:
		r.Stats.Warnings++
	case SeverityInfo:
		r.Stats.Infos++
	}
}

// familyOf prefers the declared family, falling back to the extraction hint.
func familyOf(m *manifest.Manifest, id string, s *extract.Site) string {
	if site, ok := m.Sites[id]; ok && site.Family != "" {
		return site.Family
	}

	return s.Family()
}

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)

	return out
}

func firstLocation(locs []string) string {
	if len(locs) == 0 {
		return ""
	}

	return locs[0]
}

func hasString(list []string, v string) bool {
	return slices.Contains(list, v)
}

func join(items []string) string {
	var out strings.Builder
	for i, c := range items {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(c)
	}

	return out.String()
}

// manifestSeams learns the repo's wrapper vocabulary from the manifest: every
// declared site whose facts.via names a wrapper contributes ident → family.
// This is how a repo teaches the extractor its own seams without an extractor
// edit — the hardcoded table is only the built-in default vocabulary.
func manifestSeams(m *manifest.Manifest) map[string]string {
	seams := map[string]string{}
	for id := range m.Sites {
		s := m.Sites[id]
		if via := s.Facts.Via; via != "" {
			seams[via] = s.Family
		}
	}

	return seams
}

// disciplineFamilies maps disciplines to the families they can only make
// sense on (claim-conflict, plan T20). Disciplines absent from the table
// (bounded, pooled, no-domain-labels) are family-agnostic.
var disciplineFamilies = map[string][]string{
	"atomic-write":   {"file-write"},
	"buffered-write": {"file-write"},
	"single-writer":  {"database"},
	"wal-pragmas":    {"database"},
}

// familyAllowsDiscipline reports whether a site claiming `discipline` on
// `family` is coherent.
func familyAllowsDiscipline(family, discipline string) bool {
	required, constrained := disciplineFamilies[discipline]
	if !constrained {
		return true
	}

	return slices.Contains(required, family)
}

// checkClaimConflicts flags declared sites whose discipline claims cannot
// hold on the site's family: "atomic-write" on a file-read site is a lying
// claim, not a judgment. Warning severity — the manifest still gates, but a
// conflicted claim weakens everything it documents.
func checkClaimConflicts(m *manifest.Manifest, rep *Report) {
	for _, id := range m.IDs() {
		s := m.Sites[id]
		for _, d := range s.Claims.Discipline {
			if familyAllowsDiscipline(s.Family, d) {
				continue
			}
			rep.add(Finding{
				Rule:     RuleClaimConflict,
				Severity: SeverityWarning,
				ID:       id,
				Family:   s.Family,
				Location: m.Dir,
				Message:  fmt.Sprintf("discipline %q cannot hold on family %q — the claim conflicts with the extraction", d, s.Family),
			})
		}
	}
}
