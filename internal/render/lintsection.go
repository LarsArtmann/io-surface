package render

import (
	"fmt"
	"strings"

	"github.com/larsartmann/io-surface/internal/check"
)

// LintSection renders the detector-radar appendix for the inventory doc: one
// document, both faces (gate + radar). The findings table trusts the caller's
// order — lint.Run delivers findings sorted by severity rank, then rule, then
// ID (check.SortFindings), so this renderer stays a pure formatter and the
// section is deterministic byte-for-byte.
//
// Advisory by construction: the appendix informs the same readers the gate
// tables do and never gates a build (lint exit policy, D4). An empty report
// renders a single explicit no-findings line — an absent section would be
// indistinguishable from a stale block.
func LintSection(rep *check.Report) string {
	var b strings.Builder
	b.WriteString("## Lint radar (findings)\n\n")
	b.WriteString("> Advisory: undocumented-bug detectors below the gate contract. Rules ship at\n")
	b.WriteString("> info until fleet evidence promotes one (policy D4); this section never fails\n")
	b.WriteString("> a build.\n\n")

	if len(rep.Findings) == 0 {
		b.WriteString("No detector findings.\n\n")

		return b.String()
	}
	_, _ = fmt.Fprintf(&b, "> %d finding(s): %d error, %d warning, %d info — sorted by severity, then rule.\n\n",
		len(rep.Findings), rep.Stats.Errors, rep.Stats.Warnings, rep.Stats.Infos)

	t := newTable(
		[]string{"Severity", "Rule", "Location", "Message"},
		make([]bool, 4),
	)
	for _, f := range rep.Findings {
		loc := f.Location
		if loc == "" {
			loc = cell("")
		} else {
			loc = "`" + loc + "`"
		}
		t.row(string(f.Severity), "`"+f.Rule+"`", loc, esc(f.Message))
	}
	b.WriteString(t.render())
	b.WriteString("\n")

	return b.String()
}

// esc keeps cell content from breaking the GFM table: a raw pipe would start
// a new column, so it is escaped the way a formatter would round-trip it.
func esc(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
