package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/emit"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/render"
	"github.com/larsartmann/io-surface/internal/scaffold"
)

// checkFlags are the flags local to `io-surface check`.
type checkFlags struct {
	Manifest string `flag:"manifest" help:"manifest package directory (default <root>/docs/architecture/io-surface)"`
	JSON     bool   `flag:"json" help:"emit the report as JSON"`
	Format   string `flag:"format" help:"output format: text, json, or sarif (--json is shorthand for --format json)"`
	Advisory bool   `flag:"advisory" default:"true" help:"include weak-tier advisory findings"`
	Strict   bool   `flag:"strict" help:"treat warnings as failures"`
}

type initFlags struct {
	Force bool `flag:"force" help:"refresh facts.cue in an existing manifest (never touches claims/baseline/schema)"`
}

type renderFlags struct {
	Manifest string `flag:"manifest" help:"manifest package directory (default <root>/docs/architecture/io-surface)"`
	Out      string `flag:"out" help:"write to FILE instead of stdout"`
	Doc      string `flag:"doc" help:"apply the inventory into FILE between the io-surface markers (in place, prose preserved)"`
	Check    bool   `flag:"check" help:"with --doc: report drift instead of rewriting (exit 1 when the block is stale)"`
}

type extractFlags struct {
	IncludeTests bool `flag:"include-tests" help:"include *_test.go files (report only)"`
}

// stdout is the command output sink. It is a package variable so the
// end-to-end tests can capture what a real run writes without touching file
// descriptors.
var stdout io.Writer = os.Stdout

func runCheck(_ context.Context, root *rootFlags, flags checkFlags) error {
	m, err := manifest.Load(manifestDir(root.Root, flags.Manifest))
	if err != nil {
		return err
	}

	// A manifest that fails validation is not gated — it must be fixed — but
	// the caller still gets a full report, so --json holds for every gate
	// outcome and the exit-code decision happens after emission, uniformly.
	var rep *check.Report
	if vErrs := m.Validate(); len(vErrs) > 0 {
		rep = check.ValidationReport(m, vErrs)
	} else {
		if rep, err = check.Run(root.Root, m, check.Options{Advisory: flags.Advisory, Dirs: parseDirs(root.Dirs)}); err != nil {
			return err
		}
	}

	// One emission decision for every gate outcome; the exit-code decision
	// happens strictly after, so machines always get a parseable stdout.
	useJSON := flags.JSON || flags.Format == "json"
	switch {
	case flags.Format == "sarif":
		data, sErr := emit.SARIF(rep, version)
		if sErr != nil {
			return sErr
		}
		if _, err = stdout.Write(data); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout)
	case useJSON:
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err = enc.Encode(rep); err != nil {
			return err
		}
	default:
		printReport(rep, stdout)
	}

	if rep.Failing() || (flags.Strict && rep.Stats.Warnings > 0) {
		return errFindings
	}

	return nil
}

func printReport(rep *check.Report, w io.Writer) {
	for _, f := range rep.Findings {
		loc := ""
		if f.Location != "" {
			loc = "  " + f.Location
		}
		fmt.Fprintf(w, "%-7s %-32s %s\n", f.Severity, f.Rule, f.ID+loc)
		fmt.Fprintf(w, "        %s\n", f.Message)
	}
	fmt.Fprintf(w, "io-surface: %d declared, %d extracted (%d resolved, %d token), %d baselined — %d errors, %d warnings, %d infos\n",
		rep.Stats.Declared, rep.Stats.Extracted, rep.Stats.Resolved, rep.Stats.Token,
		rep.Stats.Baselined, rep.Stats.Errors, rep.Stats.Warnings, rep.Stats.Infos)
}

func runInit(_ context.Context, root *rootFlags, flags initFlags) error {
	sites, err := extract.Extract(root.Root, extract.Options{Dirs: parseDirs(root.Dirs)})
	if err != nil {
		return err
	}
	if len(sites) == 0 {
		return fmt.Errorf("no I/O sites found under %s/cmd or %s/internal", root.Root, root.Root)
	}
	dir, err := scaffold.Write(root.Root, sites, time.Now().UTC().Format("2006-01-02"), flags.Force)
	if err != nil {
		return err
	}
	fmt.Printf("io-surface: wrote %s (%d sites, %d families)\n", dir, len(sites), countFamilies(sites))
	fmt.Printf("next: io-surface check --root %s\n", root.Root)

	return nil
}

func countFamilies(sites []*extract.Site) int {
	seen := map[string]bool{}
	for _, s := range sites {
		if fam := s.Family(); fam != "" {
			seen[fam] = true
		}
	}

	return len(seen)
}

func runRender(_ context.Context, root *rootFlags, flags renderFlags) error {
	m, err := manifest.Load(manifestDir(root.Root, flags.Manifest))
	if err != nil {
		return err
	}

	if flags.Doc != "" {
		return applyDoc(flags.Doc, m, flags.Check)
	}

	docStr := render.Markdown(m)
	if flags.Out == "" {
		fmt.Fprint(stdout, docStr)

		return nil
	}

	return os.WriteFile(flags.Out, []byte(docStr), 0o644)
}

// applyDoc is the doc-embedding path: rewrite the block in place, or verify
// it when checking. Drift names the divergence so the author knows WHAT went
// stale, and the fix hint names the exact command.
func applyDoc(path string, m *manifest.Manifest, checkOnly bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if checkOnly {
		stored, fresh, err := render.CheckDoc(string(data), m)
		if err != nil {
			return err
		}
		if stored == fresh {
			return nil
		}
		storedLines, freshLines := storedLines(stored), storedLines(fresh)
		fmt.Fprintf(os.Stderr, "io-surface: generated block in %s is stale\n", path)
		for i := range max(len(storedLines), len(freshLines)) {
			var w, g string
			if i < len(storedLines) {
				w = storedLines[i]
			}
			if i < len(freshLines) {
				g = freshLines[i]
			}
			if w != g {
				fmt.Fprintf(os.Stderr, "  line %d:\n  stored: %s\n  fresh:  %s\n", i+1, w, g)
				break
			}
		}
		fmt.Fprintf(os.Stderr, "io-surface: fix with `io-surface render --doc %s`\n", path)

		return errFindings
	}
	updated, err := render.ApplyToDoc(string(data), m)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(updated), 0o644)
}

func storedLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}

	return strings.Split(s, "\n")
}

func runExtract(_ context.Context, root *rootFlags, flags extractFlags) error {
	sites, err := extract.Extract(root.Root, extract.Options{IncludeTests: flags.IncludeTests, Dirs: parseDirs(root.Dirs)})
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(map[string]any{"generator": "io-surface " + version + " (io-surface/v1)", "sites": sites})
}

// parseDirs splits a comma-separated --dirs value; empty means the default
// cmd,internal walk.
func parseDirs(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

// manifestDir resolves --manifest, defaulting to <root>/docs/architecture/io-surface.
func manifestDir(root, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	return filepath.Join(root, scaffold.Dir)
}
