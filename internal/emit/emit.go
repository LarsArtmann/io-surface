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
	"path"
	"strconv"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
)

// ToolName is the tool identity inside go-finding reports and SARIF output.
const ToolName = "io-surface"

// ToFindings converts a check report into go-finding findings. Run-level
// facts (the enforcement posture, the rule-description table) are NOT
// per-result anymore: since go-finding v1.16.0 they ride the run-level
// SARIF propertyBag once (see ToReport, go-finding#53). A finding here
// carries only its own face: rule, message, severity, position.
func ToFindings(rep *check.Report) []finding.Finding {
	out := make([]finding.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		fnd := finding.NewFinding(
			finding.RuleName(f.Rule),
			ToolName,
			f.Message,
			severityOf(f.Severity),
			positionOf(f.Rule, f.Location, rep.Enforcement),
			1.0,
		)
		out = append(out, fnd)
	}

	return out
}

// detectorRuleDescription looks up a detector rule's published description
// from its finding rule ID ("io-surface/err-dropped" → err-dropped).
func detectorRuleDescription(ruleID string) (string, bool) {
	name, ok := strings.CutPrefix(ruleID, "io-surface/")
	if !ok {
		return "", false
	}
	def, ok := extract.DetectorRule(name)
	if !ok {
		return "", false
	}

	return def.Description, true
}

// ToReport builds a go-finding Report from a check report. Run-level facts
// ride the run propertyBag (go-finding v1.16.0, #53): the enforcement
// posture ("gate"/"audit"/"lint") as `go-finding/meta/enforcement`, and
// the detector descriptions present in this run as a single
// `go-finding/meta/ruleDescriptions` map keyed by rule ID — one map instead
// of one copy of one description per result.
func ToReport(rep *check.Report, version string) *finding.Report {
	report := finding.NewReportFromFindings(finding.ToolInfo{Name: ToolName, Version: version}, ToFindings(rep))

	if rep.Enforcement != "" {
		report.SetRunProperty("go-finding/meta/enforcement", rep.Enforcement)
	}

	descriptions := make(map[string]string)
	for _, f := range rep.Findings {
		if def, ok := detectorRuleDescription(f.Rule); ok {
			descriptions[f.Rule] = def
		}
	}
	if len(descriptions) > 0 {
		report.SetRunProperty("go-finding/meta/ruleDescriptions", descriptions)
	}

	return report
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
// which go-finding treats as "not set". An entirely empty location (the
// missing-location rule, scope hints, baseline and validation findings) has
// no real path:line to anchor — see anchorFor.
func positionOf(rule, loc, enforcement string) finding.Position {
	if loc == "" {
		return finding.Pos(anchorFor(rule, enforcement), 0, 0)
	}
	path, line := loc, 0
	if i := strings.LastIndex(loc, ":"); i >= 0 {
		if n, err := strconv.Atoi(loc[i+1:]); err == nil {
			path, line = loc[:i], n
		}
	}

	return finding.Pos(finding.FilePath(path), line, 0)
}

// anchorFor decides where a location-less finding anchors in SARIF. An empty
// artifact URI is not an option: go-finding renders it as uri:"" with a
// zero region, which SARIF consumers reject, and the old fallback (a
// pseudo-file named "io-surface") sent clicks nowhere real. The honest
// anchor is the manifest file where the fix happens: baseline entries are
// edited in baseline.cue, everything else (a missing location to fill, a
// dirs-scope hint, a validation failure) is fixed in facts.cue. The lint
// face has no manifest — its defensive empty-location case (no current
// detector produces one) keeps the tool-identity pseudo-file.
func anchorFor(rule, enforcement string) finding.FilePath {
	if enforcement == "lint" {
		return finding.FilePath(ToolName)
	}
	if rule == check.RuleBaselineUnused || rule == check.RuleBaselineExpired {
		return finding.FilePath(path.Join(manifest.Dir, "baseline.cue"))
	}

	return finding.FilePath(path.Join(manifest.Dir, "facts.cue"))
}
