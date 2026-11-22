// Command io-surface checks, scaffolds and renders an io-surface/v1 manifest:
// a machine-readable inventory of every production code location that touches
// I/O (files, databases, listeners, egress, streams, entropy, IPC, env).
//
// The CLI is built with cmdguard: commands and typed flags are declared as
// structs, cmdguard owns help/error rendering and exits, and this package
// owns the one contract cmdguard cannot know — the exit-code mapping
// (0 clean, 1 gate findings, 2 usage or internal error).
//
// Exit codes: 0 = clean, 1 = gate findings, 2 = usage or internal error.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	v5 "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
)

const version = "0.6.0"

func main() {
	app, err := newApp()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "io-surface: %v\n", err)
		os.Exit(2)
	}

	// cmdguard prints the error exactly once; the returned error is for
	// exit-code mapping only.
	if err := app.Execute(context.Background()); err != nil {
		os.Exit(exitCode(err))
	}
}

// exitCode maps a sentinel to the CLI contract.
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	if errors.Is(err, errFindings) {
		return 1
	}

	return 2
}

// rootFlags are the persistent flags every subcommand inherits.
type rootFlags struct {
	Root   string `flag:"root" default:"." help:"repository root"`
	Dirs   string `flag:"dirs" help:"comma-separated directories to scan (default cmd,internal)"`
	Ignore string `flag:"ignore" help:"comma-separated glob patterns to skip, e.g. *_templ.go,zz_generated* (matched against repo-relative paths and file names)"`
}

// newApp wires the command tree. Batteries bring strict validation, error
// hints, and the auto-registered version/doctor/describe commands.
func newApp() (*v5.CLI[rootFlags], error) {
	return v5.NewApp(
		"io-surface",
		"The I/O surface manifest gate (io-surface/v1)",
		rootFlags{},
		func(app *v5.CLI[rootFlags]) error {
			checkCmd, err := v5.NewCommand("check", checkFlags{}, runCheck,
				v5.WithShort("Compare extracted I/O sites against the manifest (exit 1 on error findings)"),
				v5.WithLong("Extracts every I/O call site from the tree and pairs it against the committed manifest. "+
					"Exits 1 when error findings exist (or warnings with --strict); always emits a full report first, so machines get parseable output for every gate outcome."),
				v5.WithExample("io-surface check --root ."),
			)
			if err != nil {
				return err
			}

			initCmd, err := v5.NewCommand("init", initFlags{}, runInit,
				v5.WithShort("Scaffold a manifest from the extracted I/O sites"),
				v5.WithLong("Writes facts.cue (the mechanical snapshot), and on first run claims.cue, baseline.cue and schema.cue. "+
					"With --force it refreshes ONLY facts.cue; judgment files are never overwritten."),
				v5.WithExample("io-surface init --root . --force"),
			)
			if err != nil {
				return err
			}

			renderCmd, err := v5.NewCommand("render", renderFlags{}, runRender,
				v5.WithShort("Render the manifest as markdown"),
				v5.WithLong("Renders the manifest to stdout or --out FILE. With --doc FILE it rewrites the block between the "+
					"io-surface:begin/end markers in place, preserving the surrounding prose; --check reports drift (exit 1) instead of rewriting."),
				v5.WithExample("io-surface render --doc docs/architecture/IO.md --check"),
			)
			if err != nil {
				return err
			}

			extractCmd, err := v5.NewCommand("extract", extractFlags{}, runExtract,
				v5.WithShort("Extract I/O sites as JSON (report only)"),
				v5.WithLong("Runs the hermetic AST walk and prints the sites as JSON. This is the raw extraction half of check, without a manifest."),
				v5.WithExample("io-surface extract --root ."),
			)
			if err != nil {
				return err
			}

			lintCmd, err := v5.NewCommand("lint", lintFlags{}, runLint,
				v5.WithShort("Find I/O bug patterns without a manifest (report only)"),
				v5.WithLong("Runs the same hermetic extraction plus detector rules over dropped errors, missing closers, context-blind calls and loop-carried I/O. "+
					"No manifest is needed, so any Go tree can be linted; findings never fail by default — pass --enforce to exit 1 on error-severity findings. "+
					"Severity is evidence-gated: rules ship at info and are promoted only after fleet false-positive measurement. "+
					"Per-rule cards, known false-positive classes, and the //io-surface:ignore suppression grammar: see the README's detector rule cards."),
				v5.WithExample("io-surface lint --root ."),
			)
			if err != nil {
				return err
			}

			absorbCmd, err := v5.NewCommand("absorb", absorbFlags{}, runAbsorb,
				v5.WithShort("Ingest a pinned Capslock report as capability facts (advisory)"),
				v5.WithLong("Folds a Capslock JSON report into per-package capability sets and writes capabilities.cue — that file ONLY; "+
					"claims, baseline and facts are never machine-written. With --check it reports advisory drift between the committed capabilities "+
					"and a fresh report; capability findings never exit 1 (promotion to a gate is an enforcement-policy decision, not this command's)."),
				v5.WithExample("io-surface absorb --capslock capslock-report.json"),
			)
			if err != nil {
				return err
			}

			baselineAddCmd, err := v5.NewCommand("add", baselineAddFlags{}, runBaselineAdd,
				v5.WithShort("Grandfather one undeclared site: add <id> --note \"why\""),
				v5.WithLong("Appends {id, note, date} to baseline.cue — the machine's only sanctioned write into a judgment file, "+
					"surgical insertion with comments preserved. Refuses duplicates and empty notes; warns when the ID does not extract today. "+
					"The baseline only shrinks: every entry needs a reason and a date."),
				v5.WithExample("io-surface baseline add cmd/tool.LegacyPath --note \"removed next quarter\" --date 2026-12-01"),
			)
			if err != nil {
				return err
			}
			versionCmd, err := v5.NewCommand("version", versionFlags{}, runVersion,
				v5.WithShort("Print the version (--json adds commit and build date)"),
				v5.WithLong("Prints `io-surface <version>` like always; --json emits {version, commit, date} from the build info, so scripts and badges get the full provenance without shelling out to go version."),
				v5.WithExample("io-surface version --json"),
			)
			if err != nil {
				return err
			}

			baselineCmd, err := v5.NewParentCommand[rootFlags, baselineAddFlags]("baseline", "Grandfather undeclared sites (the baseline only shrinks).", baselineAddFlags{},
				v5.WithShort("Grandfather undeclared sites"),
				v5.WithSubcommands(baselineAddCmd),
			)
			if err != nil {
				return err
			}

			return v5.AddCommands(app, checkCmd, initCmd, renderCmd, extractCmd, lintCmd, absorbCmd, baselineCmd, versionCmd)
		},
		v5.WithCLIVersion(version),
		v5.WithBatteries(),
	)
}
