package extract

// Mutation corpus (superb plan T10, decision D3): recall is a test, not a
// claim. Every mutator injects the bug CLASS by hand into a clean source —
// not the detector's own pattern shape — and the rule must fire on the
// mutated tree and stay silent on the clean one. Recall is pinned at 100%
// per rule: any miss fails CI. The corpus is small and hand-built on
// purpose; growing it is how a future rule earns its floor.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// mutationCase is one bug injection: before is clean, after carries the
// class. Both compile-parse only (never built), same file layout.
type mutationCase struct {
	name   string
	rule   string
	class  string // the injected bug class, in one clause
	file   string // repo-relative path inside the temp tree
	before string
	after  string
}

func mutationCorpus() []mutationCase {
	mainPkg := "package main\n\nimport (\n\t\"context\"\n\t\"io\"\n\t\"net/http\"\n\t\"os\"\n\t\"os/exec\"\n)\n\n"

	return []mutationCase{
		{
			name:   "err-dropped/blank-assign",
			rule:   RuleErrDropped,
			class:  "error slot blanked in a two-value assign",
			file:   "cmd/app/main.go",
			before: mainPkg + "func load(p string) ([]byte, error) {\n\tdata, err := os.ReadFile(p)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn data, nil\n}\n",
			after:  mainPkg + "func load(p string) ([]byte, error) {\n\tdata, _ := os.ReadFile(p)\n\n\treturn data, nil\n}\n",
		},
		{
			name:   "err-dropped/bare-call",
			rule:   RuleErrDropped,
			class:  "checked call becomes a bare statement",
			file:   "cmd/app/main.go",
			before: mainPkg + "func flush(p string) error {\n\treturn os.WriteFile(p, []byte(\"x\"), 0o644)\n}\n",
			after:  mainPkg + "func flush(p string) {\n\tos.WriteFile(p, []byte(\"x\"), 0o644)\n}\n",
		},
		{
			name:   "closer-missing/defer-removed",
			rule:   RuleCloserMissing,
			class:  "the handle is opened, then never used or closed",
			file:   "cmd/app/main.go",
			before: mainPkg + "func head(p string) ([]byte, error) {\n\tf, err := os.Open(p)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tdefer f.Close()\n\n\tdata, err := io.ReadAll(f)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn data, nil\n}\n",
			after:  mainPkg + "func head(p string) ([]byte, error) {\n\tf, err := os.Open(p)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn nil, nil\n}\n",
		},
		{
			name:   "closer-missing/http-body",
			rule:   RuleCloserMissing,
			class:  "the resp.Body close duty is dropped",
			file:   "cmd/app/main.go",
			before: mainPkg + "func status(url string) (int, error) {\n\tresp, err := http.Get(url)\n\tif err != nil {\n\t\treturn 0, err\n\t}\n\tdefer resp.Body.Close()\n\n\treturn resp.StatusCode, nil\n}\n",
			after:  mainPkg + "func status(url string) (int, error) {\n\tresp, err := http.Get(url)\n\tif err != nil {\n\t\treturn 0, err\n\t}\n\n\treturn resp.StatusCode, nil\n}\n",
		},
		{
			name:   "ctx-dropped/command-variant",
			rule:   RuleCtxDropped,
			class:  "the context-aware variant is swapped for the plain one",
			file:   "cmd/app/main.go",
			before: mainPkg + "func gitStatus(ctx context.Context) error {\n\treturn exec.CommandContext(ctx, \"git\", \"status\").Run()\n}\n",
			after:  mainPkg + "func gitStatus(ctx context.Context) error {\n\treturn exec.Command(\"git\", \"status\").Run()\n}\n",
		},
		{
			name:   "defer-in-loop/defer-moved-into-loop",
			rule:   RuleDeferInLoop,
			class:  "per-item close deferred inside the loop body",
			file:   "cmd/app/main.go",
			before: mainPkg + "func readAll(paths []string) error {\n\tfor _, p := range paths {\n\t\tif err := readOne(p); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\n\treturn nil\n}\n\nfunc readOne(p string) error {\n\tf, err := os.Open(p)\n\tif err != nil {\n\t\treturn err\n\t}\n\tdefer f.Close()\n\n\t_, err = io.ReadAll(f)\n\n\treturn err\n}\n",
			after:  mainPkg + "func readAll(paths []string) error {\n\tfor _, p := range paths {\n\t\tf, err := os.Open(p)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tdefer f.Close()\n\n\t\tif _, err := io.ReadAll(f); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\n\treturn nil\n}\n",
		},
		{
			name:   "io-in-loop/open-moved-into-loop",
			rule:   RuleIOInLoop,
			class:  "a once-outside read becomes a per-iteration re-read",
			file:   "cmd/app/main.go",
			before: mainPkg + "func render(p string, n int) error {\n\tdata, err := os.ReadFile(p)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\tfor i := 0; i < n; i++ {\n\t\t_ = len(data)\n\t}\n\n\treturn nil\n}\n",
			after:  mainPkg + "func render(p string, n int) error {\n\tfor i := 0; i < n; i++ {\n\t\tif _, err := os.ReadFile(p); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\n\treturn nil\n}\n",
		},
		{
			name:   "client-no-timeout/timeout-dropped",
			rule:   RuleClientNoTimeout,
			class:  "the Timeout field removed from the client literal",
			file:   "cmd/app/main.go",
			before: mainPkg + "func fetch(url string) (*http.Response, error) {\n\tc := &http.Client{Timeout: 10}\n\n\treturn c.Get(url)\n}\n",
			after:  mainPkg + "func fetch(url string) (*http.Response, error) {\n\tc := &http.Client{}\n\n\treturn c.Get(url)\n}\n",
		},
		{
			name:   "write-non-atomic/rename-dropped",
			rule:   RuleWriteNonAtomic,
			class:  "temp+rename replaced by a direct final-path write",
			file:   "cmd/app/main.go",
			before: mainPkg + "func save(p string) error {\n\ttmp := p + \".tmp\"\n\tif err := os.WriteFile(tmp, []byte(\"x\"), 0o644); err != nil {\n\t\treturn err\n\t}\n\n\treturn os.Rename(tmp, p)\n}\n",
			after:  mainPkg + "func save(p string) error {\n\treturn os.WriteFile(p, []byte(\"x\"), 0o644)\n}\n",
		},
		{
			name:   "tiny-buffer/buffer-shrunk",
			rule:   RuleTinyBuffer,
			class:  "a page-sized loop buffer shrunk below the syscall floor",
			file:   "cmd/app/main.go",
			before: mainPkg + "func slurp(f *os.File, n int) {\n\tbuf := make([]byte, 4096)\n\n\tfor i := 0; i < n; i++ {\n\t\t_, _ = io.ReadFull(f, buf)\n\t}\n}\n",
			after:  mainPkg + "func slurp(f *os.File, n int) {\n\tbuf := make([]byte, 64)\n\n\tfor i := 0; i < n; i++ {\n\t\t_, _ = io.ReadFull(f, buf)\n\t}\n}\n",
		},
	}
}

// TestMutationRecall runs the corpus: every mutator's after-tree must fire
// its rule, every before-tree must stay clean. Recall is pinned at 100% per
// rule — a regression fails CI (D3). The percentage table is printed for the
// published numbers.
func TestMutationRecall(t *testing.T) {
	corpus := mutationCorpus()
	hits := map[string]int{}
	total := map[string]int{}
	for _, mc := range corpus {
		mc := mc
		t.Run(mc.name, func(t *testing.T) {
			afterObs := extractObservationsFromSource(t, mc.after)
			if !observationsContain(afterObs, mc.rule) {
				t.Errorf("MISSED: mutated tree (%s) did not fire %s; recall regression", mc.class, mc.rule)
			}
			beforeObs := extractObservationsFromSource(t, mc.before)
			for _, o := range beforeObs {
				if o.Rule == mc.rule {
					t.Errorf("clean tree fired %s (%s at %s) — the mutation is not the bug-class boundary", mc.rule, o.Call, o.Location)
				}
			}
			total[mc.rule]++
			if observationsContain(afterObs, mc.rule) {
				hits[mc.rule]++
			}
		})
	}

	rules := make([]string, 0, len(total))
	for r := range total {
		rules = append(rules, r)
	}
	sort.Strings(rules)
	for _, r := range rules {
		pct := 100 * hits[r] / total[r]
		t.Logf("recall %-18s %d/%d = %d%%", r, hits[r], total[r], pct)
		if pct != 100 {
			t.Errorf("rule %s recall %d%% below the pinned floor (100%%)", r, pct)
		}
	}
}

// extractObservationsFromSource writes one source file into a fresh temp
// tree (cmd/app layout, the default scan dir) and runs the observing
// extraction on it.
func extractObservationsFromSource(t *testing.T, src string) []Observation {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "cmd", "app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, obs, err := ExtractWithObservations(root, Options{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	return obs
}

func observationsContain(obs []Observation, rule string) bool {
	for _, o := range obs {
		if o.Rule == rule {
			return true
		}
	}

	return false
}
