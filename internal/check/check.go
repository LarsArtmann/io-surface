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
	RuleManifestValidation = "io-surface/manifest-validation"
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
}

// Failing reports whether the report blocks a merge (any error-severity
// finding).
func (r *Report) Failing() bool { return r.Stats.Errors > 0 }

// Options controls the check.
type Options struct {
	// Dirs limits extraction (default cmd, internal).
	Dirs []string
	// Advisory includes weak-tier findings (default true).
	Advisory bool
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
	rep := &Report{}
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
	sites, err := extract.Extract(root, extract.Options{Dirs: opts.Dirs})
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", root, err)
	}

	rep := &Report{}
	rep.Stats.Declared = len(m.Sites)
	rep.Stats.Baselined = len(m.Baseline)
	rep.Stats.Extracted = len(sites)

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

	for _, canon := range sortedKeys(buckets) {
		b := buckets[canon]
		declaredSet := map[string]bool{}
		for _, id := range b.declared {
			declaredSet[id] = true
		}
		paired := map[string]*extract.Site{} // declared ID -> extracted site
		usedExtracted := map[*extract.Site]bool{}

		// 1. Exact ID matches pair first; they are never ambiguous.
		for _, s := range b.extracted {
			if declaredSet[s.ID] && !usedExtracted[s] {
				paired[s.ID] = s
				usedExtracted[s] = true
			}
		}
		// 2. Then a 1:1 leftover pair is ID drift (method -> function etc.).
		var leftoversDeclared []string
		for _, id := range b.declared {
			if paired[id] == nil {
				leftoversDeclared = append(leftoversDeclared, id)
			}
		}
		var leftoversExtracted []*extract.Site
		for _, s := range b.extracted {
			if !usedExtracted[s] {
				leftoversExtracted = append(leftoversExtracted, s)
			}
		}
		if len(leftoversDeclared) == 1 && len(leftoversExtracted) == 1 {
			id, s := leftoversDeclared[0], leftoversExtracted[0]
			paired[id] = s
			usedExtracted[s] = true
			rep.add(Finding{
				Rule: RuleIDDrift, Severity: SeverityWarning, ID: id, Family: familyOf(m, id, s),
				Location: firstLocation(s.Locations),
				Message:  fmt.Sprintf("declared as %q but extracts as %q; update the ID form (same symbol, different receiver)", id, s.ID),
			})
			leftoversDeclared, leftoversExtracted = nil, nil
		}

		// 3. Anything still unpaired on the extraction side is undeclared.
		for _, s := range leftoversExtracted {
			if baselined[canon] {
				usedBaseline[canon] = true

				continue
			}
			if s.Confidence != extract.ConfidenceResolved {
				if opts.Advisory {
					rep.add(Finding{
						Rule: RuleWeakSite, Severity: SeverityInfo, ID: s.ID, Family: s.Family(),
						Location: firstLocation(s.Locations),
						Message:  fmt.Sprintf("weak-tier I/O site (%s) is not declared; declare it or convert the call to a resolved form", join(s.Calls)),
					})
				}

				continue
			}
			// The site whose dotted variant is declared is already covered by
			// the id-form error on the declaration; reporting undeclared too
			// would double-count one authoring mistake.
			if dotted, ok := match.DottedVariant(s.ID); ok && declaredByID[dotted] {
				continue
			}
			rep.add(Finding{
				Rule: RuleUndeclared, Severity: SeverityError, ID: s.ID, Family: s.Family(),
				Location: firstLocation(s.Locations),
				Message:  fmt.Sprintf("undeclared I/O site (%s); declare it in facts/claims or add it to baseline with a note", join(s.Calls)),
			})
		}

		// 4. Declared sites with no extraction partner: classify honestly.
		for _, id := range leftoversDeclared {
			site := m.Sites[id]
			if !site.Facts.Extractable {
				continue
			}
			// A dotted-subpackage ID can never pair: the extractor always
			// emits slash-package paths. Confirmed against the extracted set
			// so the unexported-type dotted form (internal/extract.extractor.
			// parseFile — "extractor" IS a type) stays legal. This is an
			// error, not drift: the declaration guards nothing while the
			// real call site goes unguarded (dnsblockd adoption lesson).
			if variant, ok := match.SlashVariant(id); ok && extractedByID[variant] {
				rep.add(Finding{
					Rule: RuleIDForm, Severity: SeverityError, ID: id, Family: site.Family,
					Location: firstLocation(site.Location),
					Message:  fmt.Sprintf("declared with a dotted subpackage; the extractor emits %q — the dotted form can never pair with an extraction", variant),
				})

				continue
			}
			if len(site.Location) == 0 {
				rep.add(Finding{
					Rule: RuleMissingLocation, Severity: SeverityWarning, ID: id, Family: site.Family,
					Message: "declared as extractable but carries no location; regenerate facts.cue",
				})
			}
			if len(b.extracted) > 0 {
				// The group exists but the pairing stayed ambiguous: two
				// different symbols share a base name, so a guess would lie.
				rep.add(Finding{
					Rule: RuleAmbiguousMatch, Severity: SeverityInfo, ID: id, Family: site.Family,
					Location: firstLocation(site.Location),
					Message:  fmt.Sprintf("cannot be paired unambiguously (%d declared, %d extracted here share a base symbol); declare the exact ID form", len(b.declared), len(b.extracted)),
				})

				continue
			}
			if symbolVisible(root, site.Location, match.Symbol(id)) {
				rep.add(Finding{
					Rule: RuleExtractionBlind, Severity: SeverityInfo, ID: id, Family: site.Family,
					Location: firstLocation(site.Location),
					Message:  "declared symbol exists but no watched I/O call extracts from it; mark facts.extractable=false or teach the extractor the seam",
				})

				continue
			}
			rep.add(Finding{
				Rule: RuleStaleClaim, Severity: SeverityWarning, ID: id, Family: site.Family,
				Location: firstLocation(site.Location),
				Message:  "declared site no longer extracts and its symbol is not in the declared file; the symbol moved, was renamed, or the call was removed",
			})
		}

		// 5. Paired sites: locations are derived data, so a mismatch is a
		// maintenance nudge (info), never a failure.
		for _, id := range b.declared {
			s := paired[id]
			if s == nil {
				continue
			}
			for _, loc := range m.Sites[id].Location {
				if !hasString(s.Locations, loc) {
					rep.add(Finding{
						Rule: RuleLocationDrift, Severity: SeverityInfo, ID: id, Family: m.Sites[id].Family,
						Location: loc,
						Message:  "declared location differs from the extraction (line numbers are derived data); regenerate facts.cue",
					})
				}
			}
		}
	}

	// Scope hint: a --dirs narrowing that excludes declared sites produces
	// findings that look like manifest rot but are really the flag's doing.
	// Name the cause so the fix is one edit instead of an archaeology dig.
	if outside := declaredOutside(m, effectiveDirs(opts.Dirs)); outside > 0 {
		rep.add(Finding{
			Rule: RuleScopeHint, Severity: SeverityInfo, ID: "--dirs",
			Message: fmt.Sprintf("%d declared site(s) lie outside the scanned dirs %s — this run cannot see them; widen --dirs or refresh facts.cue", outside, strings.Join(effectiveDirs(opts.Dirs), ", ")),
		})
	}

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
