// Package render turns a manifest into the human-facing inventory document.
//
// This is the "auto-documentor" face, and it is deliberately a RENDERER, not a
// generator: facts come from the manifest (so they cannot go stale silently),
// while the prose in claims.rationale is authored by a human and passed through
// verbatim. A generated facts table alone would be a worse grep; the pairing is
// the product.
package render

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/larsartmann/io-surface/internal/manifest"
)

// Markdown renders the whole inventory: header, family summary, per-family
// tables, and the scope OUT list.
func Markdown(m *manifest.Manifest) string {
	var b strings.Builder
	b.WriteString("# I/O Surface Inventory\n\n")
	b.WriteString("> GENERATED from the io-surface manifest by `io-surface render`. Do not edit by hand:\n")
	b.WriteString("> edit `facts.cue` (regenerate) or `claims.cue` (judgment) and re-render.\n")
	fmt.Fprintf(&b, ">\n> Audited %s at `%s` — %d sites, %d location claims.\n\n",
		m.Audit.Date, m.Audit.Commit, len(m.Sites), m.Audit.Claims)
	b.WriteString(sections(m))

	return b.String()
}

// sections is the generated body shared by Markdown (full document) and Block
// (embedded between doc markers): taxonomy summary, per-family tables with
// judgments, and the scope OUT list.
func sections(m *manifest.Manifest) string {
	var b strings.Builder
	b.WriteString("## Taxonomy at a glance\n\n")
	tax := newTable(
		[]string{"Family", "Sites", "Claimed", "With rationale"},
		[]bool{false, true, true, true},
	)
	counts := m.CountByFamily()
	claimed := map[string]int{}
	rationale := map[string]int{}
	for _, s := range m.Sites {
		if s.Claims.Resource != "" || s.Claims.Trigger != "" {
			claimed[s.Family]++
		}
		if s.Claims.Rationale != "" {
			rationale[s.Family]++
		}
	}
	for _, fam := range manifest.Families {
		n := counts[fam]
		if n == 0 {
			continue
		}
		tax.row("`"+fam+"`", strconv.Itoa(n), strconv.Itoa(claimed[fam]), strconv.Itoa(rationale[fam]))
	}
	b.WriteString(tax.render())
	b.WriteString("\n")

	byFamily := map[string][]string{}
	for id, s := range m.Sites {
		byFamily[s.Family] = append(byFamily[s.Family], id)
	}
	for _, fam := range manifest.Families {
		ids := byFamily[fam]
		if len(ids) == 0 {
			continue
		}
		sort.Strings(ids)
		fmt.Fprintf(&b, "## `%s` (%d)\n\n", fam, len(ids))
		sites := newTable(
			[]string{"Site", "Location", "Resource", "Trigger", "Discipline", "Crit", "Durab"},
			make([]bool, 7),
		)
		for _, id := range ids {
			s := m.Sites[id]
			loc := ""
			if len(s.Location) > 0 {
				loc = "`" + s.Location[0] + "`"
			}
			disc := ""
			if len(s.Claims.Discipline) > 0 {
				disc = strings.Join(s.Claims.Discipline, ", ")
			}
			crit, durab := s.Claims.Criticality, s.Claims.Durability
			if crit == "" || crit == manifest.CriticalityNormal {
				crit = ""
			}
			if durab == "" || durab == manifest.DurabilityRecoverable {
				durab = ""
			}
			sites.row("`"+id+"`", loc, cell(s.Claims.Resource), cell(s.Claims.Trigger), cell(disc), cell(crit), cell(durab))
		}
		b.WriteString(sites.render())
		b.WriteString("\n")

		withProse := []string{}
		for _, id := range ids {
			if s := m.Sites[id]; s.Claims.Rationale != "" || s.Claims.Exception != "" {
				withProse = append(withProse, id)
			}
		}
		if len(withProse) > 0 {
			b.WriteString("### Judgments\n\n")
			for _, id := range withProse {
				s := m.Sites[id]
				fmt.Fprintf(&b, "- **`%s`**", id)
				if s.Claims.Exception != "" {
					fmt.Fprintf(&b, " — exception: %s", s.Claims.Exception)
				}
				if s.Claims.Rationale != "" {
					fmt.Fprintf(&b, " — %s", s.Claims.Rationale)
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	if len(m.Capabilities) > 0 {
		b.WriteString(capabilitiesSection(m))
	}

	if len(m.Scope.Out) > 0 {
		b.WriteString("## Deliberately out of scope\n\n")
		for _, entry := range m.Scope.Out {
			fmt.Fprintf(&b, "- %s\n", entry)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// capabilitiesSection renders the engine-observed reachability facts:
// package → direct/deep capability sets, with the human judgment that answers
// a shadow (the baseline note, verbatim — it is the only package-keyed
// judgment the manifest carries). Advisory tier: this table informs, it never
// gates.
func capabilitiesSection(m *manifest.Manifest) string {
	var b strings.Builder
	b.WriteString("## Reachability (capabilities)\n\n")
	if m.CapabilityAudit != nil {
		fmt.Fprintf(&b, "> Engine-observed by %s %s of `%s` (%s): what each package can REACH, beyond its declared sites.\n",
			m.CapabilityAudit.Engine, m.CapabilityAudit.Version, m.CapabilityAudit.Commit, m.CapabilityAudit.Date)
	}
	b.WriteString("> `direct` = through the package's own code (high signal); `deep` adds transitive\n")
	b.WriteString("> dependency paths (stdlib linkage reaches everywhere and carries no per-package signal).\n")
	b.WriteString("> Advisory: a package whose direct capability has no covered family is a `shadow-capability` info.\n\n")

	baseline := map[string]string{}
	for _, entry := range m.Baseline {
		baseline[entry.ID] = entry.Note
	}
	caps := newTable(
		[]string{"Package", "Direct", "Deep (transitive)", "Judgment"},
		make([]bool, 4),
	)
	for _, c := range m.Capabilities {
		caps.row("`"+c.Pkg+"`", joinCaps(c.Direct), joinCaps(c.Deep), cell(baseline[c.Pkg]))
	}
	b.WriteString(caps.render())
	b.WriteString("\n")

	return b.String()
}

// joinCaps renders a capability list for a table cell.
func joinCaps(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = "`" + n + "`"
	}

	return strings.Join(quoted, ", ")
}

// cell keeps empty table cells from collapsing the row.
func cell(s string) string {
	if s == "" {
		return "—"
	}

	return s
}
