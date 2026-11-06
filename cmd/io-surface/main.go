// Command io-surface checks, scaffolds and renders an io-surface/v1 manifest:
// a machine-readable inventory of every production code location that touches
// I/O (files, databases, listeners, egress, streams, entropy, IPC, env).
//
// Exit codes: 0 = clean, 1 = gate findings, 2 = usage or internal error.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/render"
	"github.com/larsartmann/io-surface/internal/scaffold"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "check":
		err = runCheck(os.Args[2:], os.Stdout)
	case "init":
		err = runInit(os.Args[2:])
	case "render":
		err = runRender(os.Args[2:])
	case "extract":
		err = runExtract(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("io-surface %s\n", version)
	case "help", "-h", "--help":
		usage()
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "io-surface: %v\n", err)
		os.Exit(exitCode(err))
	}
}

// exitCode maps a sentinel to the CLI contract.
func exitCode(err error) int {
	if errors.Is(err, errFindings) {
		return 1
	}

	return 2
}

func usage() {
	fmt.Print(`io-surface — the I/O surface manifest gate (io-surface/v1)

usage:
  io-surface check   [--root DIR] [--manifest DIR] [--json] [--advisory] [--strict] [--dirs a,b]
  io-surface init    [--root DIR] [--force] [--dirs a,b]
  io-surface render  [--root DIR] [--manifest DIR] [--out FILE]
  io-surface extract [--root DIR] [--include-tests] [--dirs a,b]
  io-surface version

check exits 0 when clean, 1 when the gate has error findings, 2 on usage errors.
The manifest defaults to <root>/` + scaffold.Dir + `.
`)
}

var errFindings = errors.New("gate findings")

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

func runCheck(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	man := fs.String("manifest", "", "manifest package directory (default <root>/"+scaffold.Dir+")")
	asJSON := fs.Bool("json", false, "emit the report as JSON")
	advisory := fs.Bool("advisory", true, "include weak-tier advisory findings")
	strict := fs.Bool("strict", false, "treat warnings as failures")
	dirs := fs.String("dirs", "", "comma-separated directories to scan (default cmd,internal)")
	err := fs.Parse(args)
	if err != nil {
		return err
	}

	m, err := manifest.Load(manifestDir(*root, *man))
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
		if rep, err = check.Run(*root, m, check.Options{Advisory: *advisory, Dirs: parseDirs(*dirs)}); err != nil {
			return err
		}
	}

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err = enc.Encode(rep); err != nil {
			return err
		}
	} else {
		printReport(rep, stdout)
	}

	if rep.Failing() || (*strict && rep.Stats.Warnings > 0) {
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

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	force := fs.Bool("force", false, "refresh facts.cue in an existing manifest (never touches claims/baseline/schema)")
	dirs := fs.String("dirs", "", "comma-separated directories to scan (default cmd,internal)")
	err := fs.Parse(args)
	if err != nil {
		return err
	}

	sites, err := extract.Extract(*root, extract.Options{Dirs: parseDirs(*dirs)})
	if err != nil {
		return err
	}
	if len(sites) == 0 {
		return fmt.Errorf("no I/O sites found under %s/cmd or %s/internal", *root, *root)
	}
	dir, err := scaffold.Write(*root, sites, time.Now().UTC().Format("2006-01-02"), *force)
	if err != nil {
		return err
	}
	fmt.Printf("io-surface: wrote %s (%d sites, %d families)\n", dir, len(sites), countFamilies(sites))
	fmt.Printf("next: io-surface check --root %s\n", *root)

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

func runRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	man := fs.String("manifest", "", "manifest package directory")
	out := fs.String("out", "", "write to FILE instead of stdout")
	err := fs.Parse(args)
	if err != nil {
		return err
	}

	m, err := manifest.Load(manifestDir(*root, *man))
	if err != nil {
		return err
	}
	doc := render.Markdown(m)
	if *out == "" {
		fmt.Print(doc)

		return nil
	}

	return os.WriteFile(*out, []byte(doc), 0o644)
}

func runExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	includeTests := fs.Bool("include-tests", false, "include *_test.go files (report only)")
	dirs := fs.String("dirs", "", "comma-separated directories to scan (default cmd,internal)")
	err := fs.Parse(args)
	if err != nil {
		return err
	}

	sites, err := extract.Extract(*root, extract.Options{IncludeTests: *includeTests, Dirs: parseDirs(*dirs)})
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(map[string]any{"generator": "io-surface " + version + " (io-surface/v1)", "sites": sites})
}
