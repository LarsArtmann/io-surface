package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"runtime/debug"

	v4 "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	atomicwrite "github.com/larsartmann/go-atomic-write"
	"github.com/larsartmann/io-surface/internal/absorb"
	"github.com/larsartmann/io-surface/internal/check"
	"github.com/larsartmann/io-surface/internal/emit"
	"github.com/larsartmann/io-surface/internal/extract"
	"github.com/larsartmann/io-surface/internal/lint"
	"github.com/larsartmann/io-surface/internal/manifest"
	"github.com/larsartmann/io-surface/internal/render"
	"github.com/larsartmann/io-surface/internal/scaffold"
)

// checkFlags are the flags local to `io-surface check`.
type checkFlags struct {
	Manifest            string `flag:"manifest" help:"manifest package directory (default <root>/docs/architecture/io-surface)"`
	JSON                bool   `flag:"json" help:"emit the report as JSON"`
	Format              string `flag:"format" help:"output format: text, json, or sarif (--json is shorthand for --format json)"`
	Advisory            bool   `flag:"advisory" default:"true" help:"include weak-tier advisory findings"`
	Strict              bool   `flag:"strict" help:"treat warnings as failures"`
	Enforce             bool   `flag:"enforce" help:"fail on errors even when the manifest declares enforcement: audit"`
	StrictDirs          bool   `flag:"strict-dirs" help:"fail when declared sites fall outside --dirs instead of hinting"`
	StrictWeak          bool   `flag:"strict-weak" help:"fail when weak-tier (heuristic) I/O sites are undeclared instead of hinting"`
	AllowUnusedBaseline bool   `flag:"allow-unused-baseline" help:"unused-baseline findings never fail (exempt from --strict) while the baseline shrinks"`
}

type initFlags struct {
	Force bool `flag:"force" help:"refresh facts.cue in an existing manifest (never touches claims/baseline/schema)"`
}

type versionFlags struct {
	JSON bool `flag:"json" help:"emit {version, commit, date} as JSON"`
}

// runVersion keeps the one-line plain output byte-identical (TestExecute_
// VersionAndUnknownCommand pins it) and adds the --json provenance shape;
// the consumer command wins over cmdguard's auto-registered one.
func runVersion(_ context.Context, _ *rootFlags, flags versionFlags) error {
	if !flags.JSON {
		_, _ = fmt.Fprintln(stdout, "io-surface "+version)

		return nil
	}
	commit, date := "unknown", "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if s.Value != "" {
					commit = s.Value
				}
			case "vcs.time":
				if s.Value != "" {
					date = s.Value
				}
			}
		}
	}
	enc := json.NewEncoder(stdout)

	return enc.Encode(map[string]string{"version": version, "commit": commit, "date": date})
}

type baselineAddFlags struct {
	Note string `flag:"note" required:"true" help:"why this site stays undeclared (a baseline entry without a reason is how the baseline stops shrinking)"`
	Date string `flag:"date" help:"entry date YYYY-MM-DD (default: today)"`
}

type renderFlags struct {
	Manifest string `flag:"manifest" help:"manifest package directory (default <root>/docs/architecture/io-surface)"`
	Out      string `flag:"out" help:"write to FILE instead of stdout"`
	Doc      string `flag:"doc" help:"apply the inventory into FILE between the io-surface markers (in place, prose preserved)"`
	Check    bool   `flag:"check" help:"with --doc: report drift instead of rewriting (exit 1 when the block is stale)"`
}

type extractFlags struct {
	IncludeTests bool `flag:"include-tests" help:"include *_test.go files (report only)"`
	Stats        bool `flag:"stats" help:"print a per-package site histogram instead of JSON (spot outlier packages)"`
}

type absorbFlags struct {
	Manifest      string `flag:"manifest" help:"manifest package directory (default <root>/docs/architecture/io-surface)"`
	Capslock      string `flag:"capslock" required:"true" help:"path to the pinned Capslock JSON report"`
	Check         bool   `flag:"check" help:"compare a fresh report against the committed capabilities instead of writing (advisory output, never fails)"`
	Module        string `flag:"module" help:"override the module path (default: parsed from go.mod)"`
	EngineVersion string `flag:"engine-version" help:"engine version recorded in the capability audit"`
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
		if rep, err = check.Run(root.Root, m, check.Options{Advisory: flags.Advisory, Dirs: parseDirs(root.Dirs), Ignore: parseGlobs(root.Ignore), StrictDirs: flags.StrictDirs, StrictWeak: flags.StrictWeak}); err != nil {
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
		_, _ = fmt.Fprintln(os.Stdout)
	case useJSON:
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err = enc.Encode(rep); err != nil {
			return err
		}
	default:
		printReport(rep, stdout)
	}

	if rep.Failing() || (flags.Strict && rep.StrictWarnings(flags.AllowUnusedBaseline) > 0) {
		if m.EnforcementPolicy() == manifest.EnforcementAudit && !flags.Enforce {
			// Human-facing note: stdout in text mode, stderr in machine
			// modes (the JSON/SARIF stdout contract must stay parseable).
			note := fmt.Sprintf("io-surface: enforcement: audit — %d error(s) reported but not failing (the manifest records the policy; override with --enforce)\n", rep.Stats.Errors)
			if flags.Format == "sarif" || flags.JSON || flags.Format == "json" {
				_, _ = fmt.Fprint(os.Stderr, note)
			} else {
				_, _ = fmt.Fprint(stdout, note)
			}

			return nil
		}

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
		_, _ = fmt.Fprintf(w, "%-7s %-32s %s\n", f.Severity, f.Rule, f.ID+loc)
		_, _ = fmt.Fprintf(w, "        %s\n", f.Message)
	}
	_, _ = fmt.Fprintf(w, "io-surface: %d declared, %d extracted (%d resolved, %d token), %d baselined — %d errors, %d warnings, %d infos\n",
		rep.Stats.Declared, rep.Stats.Extracted, rep.Stats.Resolved, rep.Stats.Token,
		rep.Stats.Baselined, rep.Stats.Errors, rep.Stats.Warnings, rep.Stats.Infos)
}

func runInit(_ context.Context, root *rootFlags, flags initFlags) error {
	sites, err := extract.Extract(root.Root, extract.Options{Dirs: parseDirs(root.Dirs), Ignore: parseGlobs(root.Ignore)})
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

// runAbsorb ingests a pinned Capslock JSON report. Default mode replaces
// capabilities.cue (and ONLY that file — claims, baseline and facts are
// never machine-written); --check reports advisory drift instead.
func runAbsorb(_ context.Context, root *rootFlags, flags absorbFlags) error {
	report, err := os.ReadFile(flags.Capslock)
	if err != nil {
		return err
	}
	res, err := absorb.Run(root.Root, report, absorb.Options{ModulePath: flags.Module})
	if err != nil {
		return err
	}
	audit := absorb.ResolveAudit(root.Root, absorb.Audit{Version: flags.EngineVersion})
	dropped := totalDropped(res)

	dir := manifestDir(root.Root, flags.Manifest)
	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}

	if flags.Check {
		for _, line := range absorb.Drift(m, res) {
			_, _ = fmt.Fprintf(stdout, "%-7s %-32s %s\n", "info", absorb.RuleCapabilityDrift, line)
		}
		_, _ = fmt.Fprintf(stdout, "io-surface: %d packages with capability facts, %d unmapped edges dropped — %d drift (advisory; capabilities never fail the gate)\n",
			res.Pkgs, dropped, len(absorb.Drift(m, res)))
		printDropped(stdout, res)

		return nil
	}

	path := filepath.Join(dir, "capabilities.cue")
	if err = atomicwrite.WriteWithPerm(path, absorb.CUE(res, audit), 0o644); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "io-surface: wrote %s (%d packages, audit %s@%s of %s, %d unmapped edges dropped)\n",
		path, res.Pkgs, audit.Engine, audit.Version, audit.Commit, dropped)
	printDropped(stdout, res)

	return nil
}

func totalDropped(res *absorb.Result) int {
	total := 0
	for _, n := range res.Dropped {
		total += n
	}

	return total
}

// printDropped makes unmapped capability names LOUD: they mean the engine's
// vocabulary moved (probably a schema-version bump), never a silent skip.
func printDropped(w io.Writer, res *absorb.Result) {
	if len(res.Dropped) == 0 {
		return
	}
	names := make([]string, 0, len(res.Dropped))
	for name, count := range res.Dropped {
		names = append(names, fmt.Sprintf("%s×%d", name, count))
	}
	sort.Strings(names)
	_, _ = fmt.Fprintf(w, "io-surface: dropped unmapped capabilities (out-of-scope per ROADMAP Q5, or newer engine): %s\n", strings.Join(names, ", "))
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
		_, _ = fmt.Fprint(stdout, docStr)

		return nil
	}

	return atomicwrite.WriteWithPerm(flags.Out, []byte(docStr), 0o644)
}

// writeDocVerified is the doc seam's read-modify-write commit: it writes the
// regenerated block only if the file still holds the bytes the caller's
// fingerprint was captured from. A concurrent writer fails the command with
// re-run guidance instead of silently clobbering their edit (same contract as
// the baseline append path). Tested by TestApplyDoc_ConcurrentModification.
func writeDocVerified(path string, updated []byte, fp atomicwrite.Fingerprint) error {
	err := atomicwrite.WriteVerified(path, updated, fp)
	if err == nil {
		return nil
	}
	if errors.Is(err, atomicwrite.ErrConcurrentModification) {
		return fmt.Errorf("%w: %s changed since read (re-run the command to merge)", err, path)
	}

	return fmt.Errorf("writing %s: %w", path, err)
}

// applyDoc is the doc-embedding path: rewrite the block in place, or verify
// it when checking. Drift names the divergence so the author knows WHAT went
// stale, and the fix hint names the exact command.
func applyDoc(path string, m *manifest.Manifest, checkOnly bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fp := atomicwrite.FingerprintFromBytes(data)
	if checkOnly {
		stored, fresh, err := render.CheckDoc(string(data), m)
		if err != nil {
			return err
		}
		if stored == fresh {
			return nil
		}
		storedLines, freshLines := storedLines(stored), storedLines(fresh)
		_, _ = fmt.Fprintf(os.Stderr, "io-surface: generated block in %s is stale\n", path)
		for i := range max(len(storedLines), len(freshLines)) {
			var w, g string
			if i < len(storedLines) {
				w = storedLines[i]
			}
			if i < len(freshLines) {
				g = freshLines[i]
			}
			if w != g {
				_, _ = fmt.Fprintf(os.Stderr, "  line %d:\n  stored: %s\n  fresh:  %s\n", i+1, w, g)
				break
			}
		}
		_, _ = fmt.Fprintf(os.Stderr, "io-surface: fix with `io-surface render --doc %s`\n", path)

		return errFindings
	}
	updated, err := render.ApplyToDoc(string(data), m)
	if err != nil {
		return err
	}

	return writeDocVerified(path, []byte(updated), fp)
}

func storedLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}

	return strings.Split(s, "\n")
}

// parseGlobs splits the --ignore comma list into non-empty trimmed patterns.
func parseGlobs(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

func runExtract(_ context.Context, root *rootFlags, flags extractFlags) error {
	sites, err := extract.Extract(root.Root, extract.Options{IncludeTests: flags.IncludeTests, Dirs: parseDirs(root.Dirs), Ignore: parseGlobs(root.Ignore)})
	if err != nil {
		return err
	}
	if flags.Stats {
		return printStats(sites)
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(map[string]any{"generator": "io-surface " + version + " (io-surface/v1)", "sites": sites})
}

// printStats renders the per-package histogram: package, site count, and the
// resolved/token split — the outlier spotter for large trees.
func printStats(sites []*extract.Site) error {
	counts := map[string]int{}
	resolved := map[string]int{}
	for _, s := range sites {
		pkg := s.ID
		if i := strings.LastIndex(pkg, "."); i > 0 {
			pkg = pkg[:i]
		}
		counts[pkg]++
		if s.Confidence == extract.ConfidenceResolved {
			resolved[pkg]++
		}
	}
	pkgs := make([]string, 0, len(counts))
	for p := range counts {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	_, _ = fmt.Fprintf(stdout, "%-6s %-8s %s\n", "sites", "resolved", "package")
	for _, p := range pkgs {
		_, _ = fmt.Fprintf(stdout, "%-6d %-8d %s\n", counts[p], resolved[p], p)
	}
	_, _ = fmt.Fprintf(stdout, "%-6d %-8d TOTAL (%d packages)\n", len(sites), len(sites)-sumTokens(sites), len(pkgs))

	return nil
}

func sumTokens(sites []*extract.Site) int {
	n := 0
	for _, s := range sites {
		if s.Confidence == extract.ConfidenceToken {
			n++
		}
	}

	return n
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

// runBaselineAdd grandfathers one undeclared site: `io-surface baseline add
// <id> --note "why" [--date YYYY-MM-DD]`. It is the machine's only sanctioned
// write into baseline.cue (surgical insertion, comments preserved), replacing
// hand-edited CUE. The command exits 0 on success and prints the resulting
// gate status so the operator sees the effect immediately.
func runBaselineAdd(ctx context.Context, root *rootFlags, flags baselineAddFlags) error {
	args := v4.ArgsFromContext(ctx)
	if len(args) != 1 {
		return fmt.Errorf("baseline add takes exactly one site ID argument (usage: io-surface baseline add <pkg.Func> --note \"why\")")
	}
	id := args[0]

	dir := manifestDir(root.Root, "")
	if _, err := manifest.Load(dir); err != nil {
		return err
	}
	if err := manifest.AppendBaseline(dir, id, flags.Note, flags.Date); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "io-surface: baselined %s (note: %s, date: %s)\n", id, flags.Note, flags.Date)

	// Advisory feedback: a baselined ID that does not extract today is an
	// entry suppressing nothing — the unused-baseline warning waiting to
	// happen. Tell the operator now, not at the next gate run.
	sites, err := extract.Extract(root.Root, extract.Options{Dirs: parseDirs(root.Dirs), Ignore: parseGlobs(root.Ignore)})
	if err != nil {
		return err
	}
	found := false
	for _, s := range sites {
		if s.ID == id {
			found = true

			break
		}
	}
	if !found {
		_, _ = fmt.Fprintf(stdout, "io-surface: warning: %s does not extract from this tree today; the entry suppresses nothing until the site reappears\n", id)
	}

	return nil
}

// lintFlags are the flags local to `io-surface lint`.
type lintFlags struct {
	Format       string `flag:"format" help:"output format: text, json, sarif, or github (workflow annotations)"`
	Enforce      bool   `flag:"enforce" help:"exit 1 on error-severity findings (default: report only, exit 0)"`
	IncludeTests bool   `flag:"include-tests" help:"also scan *_test.go files"`
	Manifest     string `flag:"manifest" help:"manifest package directory for claim-based severity modulation (default <root>/docs/architecture/io-surface; absent manifest = no modulation)"`
}

// runLint is the manifestless face: extraction + detector observations, no
// CUE manifest required. Exit 0 unless --enforce and error findings exist
// (D5): rules ship at info, so the default can never surprise a CI.
func runLint(_ context.Context, root *rootFlags, flags lintFlags) error {
	rep, err := lint.Run(root.Root, lint.Options{Dirs: parseDirs(root.Dirs), Ignore: parseGlobs(root.Ignore), IncludeTests: flags.IncludeTests, Manifest: flags.Manifest})
	if err != nil {
		return err
	}

	switch flags.Format {
	case "sarif":
		data, sErr := emit.SARIF(rep, version)
		if sErr != nil {
			return sErr
		}
		if _, err = stdout.Write(data); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(os.Stdout)
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(rep)
	case "github":
		printGitHubAnnotations(rep, stdout)
	default:
		printLintReport(rep, stdout)
	}

	if flags.Enforce && rep.Failing() {
		return errFindings
	}

	return nil
}

func printLintReport(rep *check.Report, w io.Writer) {
	for _, f := range rep.Findings {
		loc := ""
		if f.Location != "" {
			loc = "  " + f.Location
		}
		_, _ = fmt.Fprintf(w, "%-7s %-32s %s\n", f.Severity, f.Rule, f.ID+loc)
		_, _ = fmt.Fprintf(w, "        %s\n", f.Message)
	}
	_, _ = fmt.Fprintf(w, "io-surface lint: %d sites (%d resolved, %d token) — %d errors, %d warnings, %d infos (report only; pass --enforce to fail on errors)\n",
		rep.Stats.Extracted, rep.Stats.Resolved, rep.Stats.Token,
		rep.Stats.Errors, rep.Stats.Warnings, rep.Stats.Infos)
}

// printGitHubAnnotations renders findings as GitHub workflow commands
// (https://docs.github.com/actions/reference/workflow-commands-for-github-actions):
// every finding becomes one annotation anchored at its file:line, info maps
// to ::notice. The exit-code contract is untouched — annotations decorate a
// report-only run; --enforce still owns the exit code.
func printGitHubAnnotations(rep *check.Report, w io.Writer) {
	for _, f := range rep.Findings {
		cmd := "notice"
		switch f.Severity {
		case check.SeverityError:
			cmd = "error"
		case check.SeverityWarning:
			cmd = "warning"
		}
		file, line := splitAnnotationLocation(f.Location)
		props := "title=" + githubEscape(f.Rule)
		if file != "" {
			props += ",file=" + githubEscape(file) + ",line=" + line
		}
		_, _ = fmt.Fprintf(w, "::%s %s::%s\n", cmd, props, githubEscape(f.Message))
	}
	_, _ = fmt.Fprintf(w, "::notice title=io-surface lint::%d sites (%d resolved, %d token) — %d errors, %d warnings, %d infos (report only; pass --enforce to fail on errors)\n",
		rep.Stats.Extracted, rep.Stats.Resolved, rep.Stats.Token,
		rep.Stats.Errors, rep.Stats.Warnings, rep.Stats.Infos)
}

// splitAnnotationLocation splits the repo-relative "path:line" location
// contract; an empty line (path-only location) still anchors the file.
func splitAnnotationLocation(loc string) (file, line string) {
	if loc == "" {
		return "", ""
	}
	i := strings.LastIndex(loc, ":")
	if i < 0 {
		return loc, ""
	}

	return loc[:i], loc[i+1:]
}

// githubEscape escapes the three characters the workflow-command data syntax
// reserves (newline, CR, percent).
func githubEscape(s string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")

	return r.Replace(s)
}
