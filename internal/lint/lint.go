// Package lint is the manifestless face of io-surface: extraction plus
// detector observations, no CUE manifest required. It exists so a repo that
// has never declared its I/O surface still gets real bug findings today —
// the fleet scan showed 181 of 182 projects have no manifest, and requiring
// onboarding before any value would hide the detectors behind ceremony.
//
// The report reuses check.Report and emit unchanged (SARIF/JSON for free)
// with Enforcement "lint": machine consumers can tell the faces apart. Exit
// policy mirrors the shipped enforcement vocabulary in reverse: lint never
// fails by default (everything ships at info until fleet evidence promotes a
// rule — D4); --enforce fails on error-severity findings (D5).
package lint

import (
	"fmt"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
)

// Options controls a lint run.
type Options struct {
	// Dirs limits extraction (default cmd, internal) — same contract as check.
	Dirs []string
	// Ignore skips files matching these extraction globs (generated code).
	Ignore []string
	// IncludeTests also scans *_test.go files.
	IncludeTests bool
}

// RuleID is the stable finding rule ID for a detector rule name (D7).
func RuleID(name string) string { return "io-surface/" + name }

// Run extracts the tree under root with observations and converts them into
// a check.Report.
func Run(root string, opts Options) (*check.Report, error) {
	sites, obs, err := extract.ExtractWithObservations(root, extract.Options{
		IncludeTests: opts.IncludeTests,
		Dirs:         opts.Dirs,
		Ignore:       opts.Ignore,
	})
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", root, err)
	}

	rep := &check.Report{Enforcement: "lint"}
	rep.Stats.Extracted = len(sites)
	for _, s := range sites {
		if s.Confidence == extract.ConfidenceResolved {
			rep.Stats.Resolved++
		} else {
			rep.Stats.Token++
		}
	}
	for _, o := range obs {
		def, ok := extract.DetectorRule(o.Rule)
		if !ok {
			return nil, fmt.Errorf("observation %s/%s has no rule metadata (registry drift)", o.Rule, o.Location)
		}
		f := check.Finding{
			Rule:     RuleID(o.Rule),
			Severity: severityOf(def.Severity),
			ID:       o.SiteID,
			Location: o.Location,
			Message:  def.Message(o),
		}
		rep.Findings = append(rep.Findings, f)
		switch f.Severity {
		case check.SeverityError:
			rep.Stats.Errors++
		case check.SeverityWarning:
			rep.Stats.Warnings++
		default:
			rep.Stats.Infos++
		}
	}

	return rep, nil
}

func severityOf(s string) check.Severity {
	switch s {
	case "error":
		return check.SeverityError
	case "warning":
		return check.SeverityWarning
	}

	return check.SeverityInfo
}
