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

	b.WriteString("## Taxonomy at a glance\n\n")
	b.WriteString("| Family | Sites | Claimed | With rationale |\n")
	b.WriteString("| --- | ---: | ---: | ---: |\n")
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
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d |\n", fam, n, claimed[fam], rationale[fam])
	}
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
		b.WriteString("| Site | Location | Resource | Trigger | Discipline |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
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
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n",
				id, loc, cell(s.Claims.Resource), cell(s.Claims.Trigger), cell(disc))
		}
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

	if len(m.Scope.Out) > 0 {
		b.WriteString("## Deliberately out of scope\n\n")
		for _, entry := range m.Scope.Out {
			fmt.Fprintf(&b, "- %s\n", entry)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// cell keeps empty table cells from collapsing the row.
func cell(s string) string {
	if s == "" {
		return "—"
	}

	return s
}
