// Package emit routes io-surface findings into the house go-finding model,
// so every consumer format the fleet already has (SARIF, LSP, suppression,
// go-finding's pipeline) works on the gate's output for free.
//
// The mapping is lossless for what matters and honest about what does not:
// rule IDs stay the stable contract strings ("io-surface/undeclared-site"
// becomes the SARIF ruleId unchanged), severities map 1:1 (both vocabularies
// are error/warning/info), and "path:line" locations become positions. The
// gate is deterministic, so every finding carries confidence 1.0 — the weak
// token tier's uncertainty is expressed by its rule (weak-site), not by a
// made-up confidence number.
package emit

import (
	"strconv"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/io-surface/internal/check"
)

// ToolName is the tool identity inside go-finding reports and SARIF output.
const ToolName = "io-surface"

// ToFindings converts a check report into go-finding findings.
func ToFindings(rep *check.Report) []finding.Finding {
	out := make([]finding.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		out = append(out, finding.NewFinding(
			finding.RuleName(f.Rule),
			ToolName,
			f.Message,
			severityOf(f.Severity),
			positionOf(f.Location),
			1.0,
		))
	}

	return out
}

// ToReport builds a go-finding Report from a check report.
func ToReport(rep *check.Report, version string) *finding.Report {
	return finding.NewReportFromFindings(finding.ToolInfo{Name: ToolName, Version: version}, ToFindings(rep))
}

// SARIF renders the check report as a SARIF document.
func SARIF(rep *check.Report, version string) ([]byte, error) {
	return ToReport(rep, version).ToSARIF()
}

func severityOf(s check.Severity) finding.Severity {
	switch s {
	case check.SeverityError:
		return finding.SeverityError
	case check.SeverityWarning:
		return finding.SeverityWarning

	default:
		return finding.SeverityInfo
	}
}

// positionOf parses a repo-relative "path:line" location. Locations without a
// line (the manifest itself, family-level findings) keep the path with line 0,
// which go-finding treats as "not set".
func positionOf(loc string) finding.Position {
	if loc == "" {
		return finding.Pos(finding.FilePath(ToolName), 0, 0)
	}
	path, line := loc, 0
	if i := strings.LastIndex(loc, ":"); i >= 0 {
		if n, err := strconv.Atoi(loc[i+1:]); err == nil {
			path, line = loc[:i], n
		}
	}

	return finding.Pos(finding.FilePath(path), line, 0)
}
