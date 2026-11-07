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
//
// When a manifest IS present, claims modulate severity (v1.2, plan T20):
// findings on `critical` sites escalate one step. The manifest is optional —
// modulation is a bonus the declared repos earn, never a requirement the
// undeclared ones pay.
package lint

import (
	"fmt"
	"path/filepath"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
)

// Options controls a lint run.
type Options struct {
	// Dirs limits extraction (default cmd, internal) — same contract as check.
	Dirs []string
	// Ignore skips files matching these extraction globs (generated code).
	Ignore []string
	// IncludeTests also scans *_test.go files.
	IncludeTests bool
	// Manifest overrides the manifest package directory (default
	// <root>/docs/architecture/io-surface). A missing or unreadable manifest
	// simply disables claim modulation — lint never requires it.
	Manifest string
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
	claims := claimsBySite(root, opts.Manifest)
	for _, o := range obs {
		def, ok := extract.DetectorRule(o.Rule)
		if !ok {
			return nil, fmt.Errorf("observation %s/%s has no rule metadata (registry drift)", o.Rule, o.Location)
		}
		sev := severityOf(def.Severity)
		c, declared := claims[o.SiteID]
		if declared && c.Criticality == manifest.CriticalityCritical {
			sev = escalate(sev)
		}
		// Durability weighs write-shaped findings (plan T21): a durable
		// claim escalates write-non-atomic one step; an ephemeral claim
		// drops it — the loss is declared harmless.
		if o.Rule == extract.RuleWriteNonAtomic && declared {
			switch c.Durability {
			case manifest.DurabilityDurable:
				sev = escalate(sev)
			case manifest.DurabilityEphemeral:
				continue
			}
		}
		f := check.Finding{
			Rule:     RuleID(o.Rule),
			Severity: sev,
			ID:       o.SiteID,
			Location: o.Location,
			Message:  def.Message(o),
		}
		if declared && c.Criticality == manifest.CriticalityCritical {
			f.Message += " [escalated: claims.criticality=critical]"
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

// claimsBySite loads the manifest when present and indexes its claims by
// site ID. Any load failure degrades to "no modulation" — the lint face
// cannot require a manifest.
func claimsBySite(root, manifestDir string) map[string]manifest.Claims {
	dir := manifestDir
	if dir == "" {
		dir = filepath.Join(root, manifest.Dir)
	}
	m, err := manifest.Load(dir)
	if err != nil || m == nil {
		return nil
	}
	out := make(map[string]manifest.Claims, len(m.Sites))
	for id, s := range m.Sites {
		out[id] = s.Claims
	}

	return out
}

// escalate moves a finding one severity step up the ladder; error stays
// error (the ladder has no ceiling above the gate).
func escalate(s check.Severity) check.Severity {
	switch s {
	case check.SeverityInfo:
		return check.SeverityWarning
	case check.SeverityWarning:
		return check.SeverityError
	}

	return s
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
