// Package check implements the bidirectional io-surface gate.
//
// The four directions, in the standard's order:
//
//  1. declared-sites-exist   — every declared location exists in the tree, and
//     for extractable sites the extraction agrees on the symbol
//     (rule: location-drift / missing-location)
//  2. undeclared-sites-absent — every high-confidence extraction finding is
//     declared or baselined (rule: undeclared-site)
//  3. constraints hold       — enforced by CUE at load time (disciplines.cue)
//     plus the schema checks in manifest.Validate
//  4. published-artifact fresh — the renderer's freshness mode
//
// Direction 2 is the one that makes the manifest complete: before it, a silent
// manifest passed every gate. Only ConfidenceResolved sites may fail; the weak
// token tier is advisory by construction.
package check

import (
	"fmt"
	"sort"

	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/match"
)

// Rule identifiers. These are stable: they become SARIF ruleIds and
// suppression keys, so renaming one is a breaking change.
const (
	RuleUndeclared      = "io-surface/undeclared-site"
	RuleStaleClaim      = "io-surface/stale-claim"
	RuleIDDrift         = "io-surface/id-drift"
	RuleLocationDrift   = "io-surface/location-drift"
	RuleMissingLocation = "io-surface/missing-location"
	RuleExtractionBlind = "io-surface/extraction-blind"
	RuleWeakSite        = "io-surface/weak-site"
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

// Run extracts the tree under root and compares it against the manifest.
func Run(root string, m *manifest.Manifest, opts Options) (*Report, error) {
	sites, err := extract.Extract(root, extract.Options{Dirs: opts.Dirs})
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", root, err)
	}

	rep := &Report{}
	declared := map[string]string{} // canonical -> declared raw ID
	for _, id := range m.IDs() {
		declared[match.Canonical(id)] = id
	}
	baselined := map[string]bool{}
	for _, id := range m.BaselineIDs() {
		baselined[match.Canonical(id)] = true
	}

	rep.Stats.Declared = len(m.Sites)
	rep.Stats.Baselined = len(m.Baseline)
	rep.Stats.Extracted = len(sites)

	extracted := map[string]*extract.Site{}
	for _, s := range sites {
		extracted[match.Canonical(s.ID)] = s
		if s.Confidence == extract.ConfidenceResolved {
			rep.Stats.Resolved++
		} else {
			rep.Stats.Token++
		}
	}

	// Direction 2: undeclared-sites-absent.
	for _, s := range sites {
		canon := match.Canonical(s.ID)
		if _, ok := declared[canon]; ok {
			continue
		}
		if baselined[canon] {
			continue
		}
		if s.Confidence != extract.ConfidenceResolved {
			if opts.Advisory {
				loc := ""
				if len(s.Locations) > 0 {
					loc = s.Locations[0]
				}
				rep.add(Finding{
					Rule: RuleWeakSite, Severity: SeverityInfo, ID: s.ID,
					Family: s.Family(), Location: loc,
					Message: fmt.Sprintf("weak-tier I/O site (%s) is not declared; declare it or convert the call to a resolved form", joinCalls(s.Calls)),
				})
			}

			continue
		}
		loc := ""
		if len(s.Locations) > 0 {
			loc = s.Locations[0]
		}
		rep.add(Finding{
			Rule: RuleUndeclared, Severity: SeverityError, ID: s.ID,
			Family: s.Family(), Location: loc,
			Message: fmt.Sprintf("undeclared I/O site (%s); declare it in facts/claims or add it to baseline with a note", joinCalls(s.Calls)),
		})
	}

	// Directions 1 + 2's converse: declared sites must still exist and agree.
	for _, id := range m.IDs() {
		site := m.Sites[id]
		if !site.Facts.Extractable {
			continue
		}
		if len(site.Location) == 0 {
			rep.add(Finding{
				Rule: RuleMissingLocation, Severity: SeverityWarning, ID: id, Family: site.Family,
				Message: "declared as extractable but carries no location; regenerate facts.cue with -mode audit",
			})
		}
		found, ok := extracted[match.Canonical(id)]
		if !ok {
			// Two very different situations look identical from the extracted
			// set: the symbol was deleted/renamed (a lying claim) or the
			// extractor simply cannot see its I/O (a blind spot). Telling
			// them apart is what keeps the warning list believable.
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

			continue
		}
		if match.Canonical(found.ID) == match.Canonical(id) && found.ID != id {
			rep.add(Finding{
				Rule: RuleIDDrift, Severity: SeverityWarning, ID: id, Family: site.Family,
				Location: firstLocation(found.Locations),
				Message:  fmt.Sprintf("declared as %q but extracts as %q; update the ID form (receiver-insensitive match, not a new site)", id, found.ID),
			})
		}
		for _, loc := range site.Location {
			if !hasString(found.Locations, loc) {
				rep.add(Finding{
					Rule: RuleLocationDrift, Severity: SeverityInfo, ID: id, Family: site.Family,
					Location: loc,
					Message:  "declared location differs from the extraction (line numbers are derived data); regenerate facts.cue",
				})
			}
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

func firstLocation(locs []string) string {
	if len(locs) == 0 {
		return ""
	}

	return locs[0]
}

func hasString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}

	return false
}

func joinCalls(calls []string) string {
	out := ""
	for i, c := range calls {
		if i > 0 {
			out += ", "
		}
		out += c
	}

	return out
}
