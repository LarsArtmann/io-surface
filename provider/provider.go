// Package provider registers io-surface as a BuildFlow DAG tool via the
// go-finding toolsdk contract — the same self-registration pattern as
// cmdguard-lint. BuildFlow discovers this Spec (toolsdk.All) whenever it
// imports this package, so `buildflow` (and ./bin/dev) runs the I/O gate as
// a first-class pipeline tool with zero BuildFlow-side glue.
//
// Detection runs the full in-process pipeline (extract + check.Run) against
// the working directory's own manifest (docs/architecture/io-surface). A
// working directory without a manifest is a clean no-op: BuildFlow sweeps
// the whole fleet, and repos that never adopted io-surface must not fail.
//
// Severity mapping honors the manifest's recorded enforcement policy (D4):
// gate policy passes severities through unchanged; audit policy demotes
// error findings to warning so an adopting repo's posture — recorded in ITS
// manifest, reviewable in ITS diffs — controls whether BuildFlow fails.
package provider

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/toolsdk"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/manifest"
)

// Provider is the BuildFlow tool spec for io-surface. ModuleFanOut is on:
// multi-module workspaces gate every module against its own manifest.
//
//nolint:gochecknoglobals // toolsdk contract: specs self-register as package-level vars
var Provider = toolsdk.Register(toolsdk.Spec{
	Name:         "io-surface",
	Description:  "I/O surface manifest gate: undeclared I/O sites, stale claims, baseline lifecycle, location drift",
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
			// No manifest in this module: io-surface is not adopted here.
			// A manifest that EXISTS but fails validation still reports.
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		rep, err := check.Run(dir, m, check.Options{})
		if err != nil {
			return nil, err
		}

		return convert(rep, m.EnforcementPolicy()), nil
	}),
	Repair: nil,
})

// convert maps a check report into go-finding findings. Rule IDs stay
// verbatim (`io-surface/undeclared-site`) so BuildFlow suppression files and
// SARIF consumers see the same vocabulary as the CLI. Under the audit
// policy, error findings demote to warning — the recorded posture governs,
// exactly as `check` does with its exit code.
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
