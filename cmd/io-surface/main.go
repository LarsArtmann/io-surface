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

	v4 "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
)

const version = "0.1.0"

func main() {
	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "io-surface: %v\n", err)
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

var errFindings = errors.New("gate findings")

// rootFlags are the persistent flags every subcommand inherits.
type rootFlags struct {
	Root string `flag:"root" default:"." help:"repository root"`
	Dirs string `flag:"dirs" help:"comma-separated directories to scan (default cmd,internal)"`
}

// newApp wires the command tree. Batteries bring strict validation, error
// hints, and the auto-registered version/doctor/describe commands.
func newApp() (*v4.CLI[rootFlags], error) {
	return v4.NewApp(
		"io-surface",
		"The I/O surface manifest gate (io-surface/v1)",
		rootFlags{},
		func(app *v4.CLI[rootFlags]) error {
			checkCmd, err := v4.NewCommand("check", checkFlags{}, runCheck,
				v4.WithShort("Compare extracted I/O sites against the manifest (exit 1 on error findings)"),
				v4.WithLong("Extracts every I/O call site from the tree and pairs it against the committed manifest. "+
					"Exits 1 when error findings exist (or warnings with --strict); always emits a full report first, so machines get parseable output for every gate outcome."),
				v4.WithExample("io-surface check --root ."),
			)
			if err != nil {
				return err
			}

			initCmd, err := v4.NewCommand("init", initFlags{}, runInit,
				v4.WithShort("Scaffold a manifest from the extracted I/O sites"),
				v4.WithLong("Writes facts.cue (the mechanical snapshot), and on first run claims.cue, baseline.cue and schema.cue. "+
					"With --force it refreshes ONLY facts.cue; judgment files are never overwritten."),
				v4.WithExample("io-surface init --root . --force"),
			)
			if err != nil {
				return err
			}

			renderCmd, err := v4.NewCommand("render", renderFlags{}, runRender,
				v4.WithShort("Render the manifest as markdown"),
				v4.WithLong("Renders the manifest to stdout or --out FILE. With --doc FILE it rewrites the block between the "+
					"io-surface:begin/end markers in place, preserving the surrounding prose; --check reports drift (exit 1) instead of rewriting."),
				v4.WithExample("io-surface render --doc docs/architecture/IO.md --check"),
			)
			if err != nil {
				return err
			}

			extractCmd, err := v4.NewCommand("extract", extractFlags{}, runExtract,
				v4.WithShort("Extract I/O sites as JSON (report only)"),
				v4.WithLong("Runs the hermetic AST walk and prints the sites as JSON. This is the raw extraction half of check, without a manifest."),
				v4.WithExample("io-surface extract --root ."),
			)
			if err != nil {
				return err
			}

			absorbCmd, err := v4.NewCommand("absorb", absorbFlags{}, runAbsorb,
				v4.WithShort("Ingest a pinned Capslock report as capability facts (advisory)"),
				v4.WithLong("Folds a Capslock JSON report into per-package capability sets and writes capabilities.cue — that file ONLY; "+
					"claims, baseline and facts are never machine-written. With --check it reports advisory drift between the committed capabilities "+
					"and a fresh report; capability findings never exit 1 (promotion to a gate is an enforcement-policy decision, not this command's)."),
				v4.WithExample("io-surface absorb --capslock capslock-report.json"),
			)
			if err != nil {
				return err
			}

			return v4.AddCommands(app, checkCmd, initCmd, renderCmd, extractCmd, absorbCmd)
		},
		v4.WithCLIVersion(version),
		v4.WithBatteries(),
	)
}
