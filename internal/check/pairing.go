package check

import (
	"fmt"

	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/match"
)

// pairer carries one Run's pairing context: the manifest, the options, the
// report under construction, and the exact-form index sets the rules consult.
// The bidirectional pairing walk (directions 1+2 of the package contract)
// lives here, moved verbatim out of Run — Run assembles the inputs and owns
// the pre/post lifecycle (claim conflicts, scope hint, shadow capabilities,
// baseline lifecycle, final ordering); this type owns the per-canonical-group
// rule firing. Split for review surface: check.go sat one line under the
// repo's 650-line BuildFlow budget (TODO #64), and the pairing loop is a
// self-contained phase with exactly these eight inputs. No API change: Run's
// signature and every rule ID stay byte-identical.
type pairer struct {
	root          string
	m             *manifest.Manifest
	opts          Options
	rep           *Report
	buckets       map[string]*bucket
	baselined     map[string]bool
	usedBaseline  map[string]bool
	extractedByID map[string]bool
	declaredByID  map[string]bool
}

// pairGroups walks every canonical group and fires the pairing rules in the
// documented order: exact IDs first (never ambiguous), then the 1:1 leftover
// as id-drift, then undeclared-side findings, then declared-side
// classification, then location drift (info — derived data).
func (p *pairer) pairGroups() {
	for _, canon := range sortedKeys(p.buckets) {
		b := p.buckets[canon]
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
			p.rep.add(Finding{
				Rule: RuleIDDrift, Severity: SeverityWarning, ID: id, Family: familyOf(p.m, id, s),
				Location: firstLocation(s.Locations),
				Message:  fmt.Sprintf("declared as %q but extracts as %q; update the ID form (same symbol, different receiver)", id, s.ID),
			})
			leftoversDeclared, leftoversExtracted = nil, nil
		}

		// 3. Anything still unpaired on the extraction side is undeclared.
		for _, s := range leftoversExtracted {
			if p.baselined[canon] {
				p.usedBaseline[canon] = true

				continue
			}
			if s.Confidence != extract.ConfidenceResolved {
				if p.opts.Advisory {
					sev := SeverityInfo
					if p.opts.StrictWeak {
						sev = SeverityError
					}
					p.rep.add(Finding{
						Rule: RuleWeakSite, Severity: sev, ID: s.ID, Family: s.Family(),
						Location: firstLocation(s.Locations),
						Message:  fmt.Sprintf("weak-tier I/O site (%s) is not declared; declare it or convert the call to a resolved form", join(s.Calls)),
					})
				}

				continue
			}
			// The site whose dotted variant is declared is already covered by
			// the id-form error on the declaration; reporting undeclared too
			// would double-count one authoring mistake.
			if dotted, ok := match.DottedVariant(s.ID); ok && p.declaredByID[dotted] {
				continue
			}
			p.rep.add(Finding{
				Rule: RuleUndeclared, Severity: SeverityError, ID: s.ID, Family: s.Family(),
				Location: firstLocation(s.Locations),
				Message:  fmt.Sprintf("undeclared I/O site (%s); declare it in facts/claims or add it to baseline with a note", join(s.Calls)),
			})
		}

		// 4. Declared sites with no extraction partner: classify honestly.
		for _, id := range leftoversDeclared {
			site := p.m.Sites[id]
			if !site.Facts.Extractable {
				continue
			}
			// A dotted-subpackage ID can never pair: the extractor always
			// emits slash-package paths. Confirmed against the extracted set
			// so the unexported-type dotted form (internal/extract.extractor.
			// parseFile — "extractor" IS a type) stays legal. This is an
			// error, not drift: the declaration guards nothing while the
			// real call site goes unguarded (dnsblockd adoption lesson).
			if variant, ok := match.SlashVariant(id); ok && p.extractedByID[variant] {
				p.rep.add(Finding{
					Rule: RuleIDForm, Severity: SeverityError, ID: id, Family: site.Family,
					Location: firstLocation(site.Location),
					Message:  fmt.Sprintf("declared with a dotted subpackage; the extractor emits %q — the dotted form can never pair with an extraction", variant),
				})

				continue
			}
			if len(site.Location) == 0 {
				p.rep.add(Finding{
					Rule: RuleMissingLocation, Severity: SeverityWarning, ID: id, Family: site.Family,
					Message: "declared as extractable but carries no location; regenerate facts.cue",
				})
			}
			if len(b.extracted) > 0 {
				// The group exists but the pairing stayed ambiguous: two
				// different symbols share a base name, so a guess would lie.
				p.rep.add(Finding{
					Rule: RuleAmbiguousMatch, Severity: SeverityInfo, ID: id, Family: site.Family,
					Location: firstLocation(site.Location),
					Message:  fmt.Sprintf("cannot be paired unambiguously (%d declared, %d extracted here share a base symbol); declare the exact ID form", len(b.declared), len(b.extracted)),
				})

				continue
			}
			if symbolVisible(p.root, site.Location, match.Symbol(id)) {
				p.rep.add(Finding{
					Rule: RuleExtractionBlind, Severity: SeverityInfo, ID: id, Family: site.Family,
					Location: firstLocation(site.Location),
					Message:  "declared symbol exists but no watched I/O call extracts from it; mark facts.extractable=false or teach the extractor the seam",
				})

				continue
			}
			p.rep.add(Finding{
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
			for _, loc := range p.m.Sites[id].Location {
				if !hasString(s.Locations, loc) {
					p.rep.add(Finding{
						Rule: RuleLocationDrift, Severity: SeverityInfo, ID: id, Family: p.m.Sites[id].Family,
						Location: loc,
						Message:  "declared location differs from the extraction (line numbers are derived data); regenerate facts.cue",
					})
				}
			}
		}
	}
}
