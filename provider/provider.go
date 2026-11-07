// Package provider registers io-surface as a BuildFlow DAG tool via the
// go-finding toolsdk contract — the same self-registration pattern as
// cmdguard-lint. BuildFlow discovers this Spec (toolsdk.All) whenever it
// imports this package, so `buildflow` (and ./bin/dev) runs the I/O gate as
// a first-class pipeline tool with zero BuildFlow-side glue.
//
// Detection runs BOTH faces of the tool against the working directory:
//
//   - the manifest gate (extract + check.Run) when the module adopted
//     io-surface (docs/architecture/io-surface present), and
//   - the io-bug-radar detector sweep (lint.Run) ALWAYS — the detectors are
//     manifestless by design, so unadopted modules still get real bug
//     findings. Radar ships at info severity (D4): BuildFlow's findings gate
//     fails on errors, and radar errors only exist where a manifest claim
//     escalated one.
//
// Severity mapping honors the manifest's recorded enforcement policy (D4)
// for BOTH faces: gate policy passes severities through unchanged; audit
// policy demotes error findings to warning — an adopting repo's posture,
// recorded in ITS manifest, reviewable in ITS diffs, controls whether
// BuildFlow fails. A working directory without a manifest is radar only,
// never a step error.
package provider

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/toolsdk"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/lint"
	"github.com/larsartmann/io-surface/internal/manifest"
)

// Provider is the BuildFlow tool spec for io-surface. ModuleFanOut is on:
// multi-module workspaces gate every module against its own manifest.
//
//nolint:gochecknoglobals // toolsdk contract: specs self-register as package-level vars
var Provider = toolsdk.Register(toolsdk.Spec{
	Name:         "io-surface",
	Description:  "I/O surface manifest gate (undeclared sites, stale claims, baseline lifecycle, location drift) + io-bug-radar detector sweep (dropped errors, unclosed bodies, non-atomic writes, …; advisory info severity, manifest optional)",
	Trigger:      toolsdk.OnGoModule(),
	Inputs:       []string{"**/*.go", "docs/architecture/io-surface/*.cue"},
	ModuleFanOut: true,

	Detect: finding.NamedDetectorFunc("io-surface", func(ctx context.Context) ([]finding.Finding, error) {
		dir := finding.WorkingDirFromContext(ctx)
		if dir == "" {
			dir = "."
		}
		manifestDir := filepath.Join(dir, "docs", "architecture", "io-surface")

		m, err := manifest.Load(manifestDir)
		if errors.Is(err, manifest.ErrNotInitialized) {
			// No manifest in this module: io-surface is not adopted here, so
			// the gate has nothing to pair against — but the radar face needs
			// no manifest, and an unadopted module is exactly where its
			// advisory findings carry the most information.
			rep, lerr := lint.Run(dir, lint.Options{})
			if lerr != nil {
				return nil, lerr
			}

			return convert(rep, manifest.EnforcementGate), nil
		}
		if err != nil {
			return nil, err
		}

		gateRep, err := check.Run(dir, m, check.Options{})
		if err != nil {
			return nil, err
		}
		radarRep, err := lint.Run(dir, lint.Options{})
		if err != nil {
			return nil, err
		}

		// One policy governs both faces: audit demotion applies to gate
		// errors and to radar findings a critical claim escalated alike.
		return append(convert(gateRep, m.EnforcementPolicy()), convert(radarRep, m.EnforcementPolicy())...), nil
	}),
	Repair: nil,
})

// convert maps a check report into go-finding findings. Rule IDs stay
// verbatim (`io-surface/undeclared-site`, `io-surface/write-non-atomic`) so
// BuildFlow suppression files and SARIF consumers see the same vocabulary
// as the CLI, whichever face produced the finding. Under the audit policy,
// error findings demote to warning — the recorded posture governs, exactly
// as `check` does with its exit code.
func convert(rep *check.Report, policy string) []finding.Finding {
	out := make([]finding.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		sev := finding.Severity(f.Severity)
		if policy == manifest.EnforcementAudit && sev == finding.SeverityError {
			sev = finding.SeverityWarning
		}
		pos := finding.Position{}
		if loc := f.Location; loc != "" {
			if i := strings.LastIndex(loc, ":"); i > 0 {
				pos.File = finding.FilePath(loc[:i])
				if n := 0; parseLine(loc[i+1:], &n) {
					pos.Line = n
				}
			}
			if pos.File == "" {
				pos.File = finding.FilePath(loc)
			}
		}
		out = append(out, finding.NewFinding(
			finding.RuleName(f.Rule), finding.ToolName("io-surface"), f.Message,
			sev, pos, finding.ConfidenceHigh,
		))
	}

	return out
}

// parseLine parses a 1-based line number suffix; junk (the summary line's
// "--dirs" pseudo-location) leaves the position unset.
func parseLine(s string, out *int) bool {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 {
		return false
	}
	*out = n

	return true
}
