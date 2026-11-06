package render

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/manifest"
)

// update regenerates golden files: go test ./internal/render -update
var update = flag.Bool("update", false, "rewrite golden files")

// fixtureManifest covers every renderer branch: three populated families (one
// with full claims prose, one with an exception judgment, one bare), an empty
// family (embed — must be skipped), and a scope OUT list.
func fixtureManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Dir: "docs/architecture/io-surface",
		Sites: map[string]manifest.Site{
			"internal/store.LoadConfig": {
				Location: []string{"internal/store/store.go:9"},
				Family:   "file-read",
				Facts:    manifest.Facts{Call: []string{"os.ReadFile"}, Extractable: true},
				Claims: manifest.Claims{
					Resource:   "service config file",
					Trigger:    "process start",
					Discipline: []string{"read-only", "no-secret"},
					Rationale:  "config is read once at boot; paths come from a validated flag",
				},
			},
			"internal/store.Save": {
				Location: []string{"internal/store/store.go:31"},
				Family:   "file-write",
				Facts:    manifest.Facts{Call: []string{"os.WriteFile"}, Extractable: true},
				Claims: manifest.Claims{
					Resource:  "state file",
					Trigger:   "explicit user command",
					Exception: "atomic write handled by wrapper",
				},
			},
			"cmd/app.fetch": {
				Location: []string{"cmd/app/main.go:18"},
				Family:   "outbound",
				Facts:    manifest.Facts{Call: []string{"http.Get"}, Extractable: true},
			},
			"cmd/app.openDB": {
				Location: []string{"cmd/app/main.go:24"},
				Family:   "database",
				Facts:    manifest.Facts{Call: []string{"sql.Open"}, Extractable: true},
				Claims: manifest.Claims{
					Resource:   "local sqlite file",
					Trigger:    "command run",
					Discipline: []string{"local-only"},
				},
			},
		},
		Audit: manifest.Audit{Commit: "abc1234", Date: "2026-11-06", Claims: 3, Method: "manual audit"},
		Scope: manifest.Scope{Out: []string{
			"`testdata/` — fixture I/O never ships",
			"`vendor/` — third-party code, not ours to declare",
		}},
	}
}

func TestMarkdown_Golden(t *testing.T) {
	got := Markdown(fixtureManifest())

	golden := filepath.Join("testdata", "golden.md")
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run once with -update): %v", err)
	}
	if got != string(want) {
		t.Fatalf("render drift\n--- diff ---\n%s", lineDiff(string(want), got))
	}
}

// TestMarkdown_Deterministic pins that map iteration cannot leak into the
// document: two renders of one manifest must be byte-identical.
func TestMarkdown_Deterministic(t *testing.T) {
	first := Markdown(fixtureManifest())
	for range 10 {
		if again := Markdown(fixtureManifest()); again != first {
			t.Fatal("Markdown is not deterministic across runs")
		}
	}
}

// TestMarkdown_EmptyManifest covers the degenerate input: header and audit
// line only, no family sections, no OUT section.
func TestMarkdown_EmptyManifest(t *testing.T) {
	got := Markdown(&manifest.Manifest{Dir: "x", Audit: manifest.Audit{Date: "2026-11-06", Commit: "working-tree"}})
	for _, unwanted := range []string{"## `", "Deliberately out of scope", "| `"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("empty manifest rendered %q:\n%s", unwanted, got)
		}
	}
	if !strings.Contains(got, "# I/O Surface Inventory") {
		t.Error("empty manifest lost the header")
	}
}

// lineDiff prints the first divergence with context — enough to eyeball what
// drifted without a full diff dependency.
func lineDiff(want, got string) string {
	wantLines, gotLines := splitLines(want), splitLines(got)
	var b strings.Builder
	n := max(len(wantLines), len(gotLines))
	for i := range n {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			fmt.Fprintf(&b, "line %d\nwant: %q\ngot:  %q\n", i+1, w, g)
			if len(b.String()) > 2000 {
				break
			}
		}
	}
	fmt.Fprintf(&b, "(%d want lines, %d got lines)\n", len(wantLines), len(gotLines))

	return b.String()
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}

	return strings.Split(s, "\n")
}
