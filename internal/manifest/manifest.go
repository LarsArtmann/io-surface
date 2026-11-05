// Package manifest loads an io-surface/v1 CUE manifest package: schema, facts,
// claims, disciplines, baseline, scope.
//
// The load is in-process (cuelang.org/go), so the checker needs no `cue` binary
// at runtime and stays hermetic in a sandbox. Loading doubles as constraint
// enforcement: a package whose `disciplines.cue` unifies a non-empty
// `violations` list fails to build, so `Load` returns that error before any
// check runs.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

// Families is the io-surface/v1 taxonomy. A site has exactly one.
var Families = []string{
	"file-read", "file-write", "embed", "database", "listener",
	"outbound", "stream-copy", "entropy", "ipc-subprocess", "env",
}

// Disciplines is the cross-cutting policy tag set (io-surface.md §10).
var Disciplines = []string{
	"atomic-write", "single-writer", "buffered-write", "wal-pragmas",
	"bounded", "pooled", "no-domain-labels",
}

// Facts are machine-extractable: regenerated freely by the extractor.
type Facts struct {
	Call        []string `json:"call"`
	Via         string   `json:"via"`
	Extractable bool     `json:"extractable"`
	Testinfra   bool     `json:"testinfra"`
}

// Claims are human judgment: never overwritten by tooling.
type Claims struct {
	Resource   string   `json:"resource"`
	Trigger    string   `json:"trigger"`
	Discipline []string `json:"discipline"`
	Exception  string   `json:"exception"`
	Rationale  string   `json:"rationale"`
}

// Site is one declared I/O surface entry.
type Site struct {
	Location []string `json:"location"`
	Family   string   `json:"family"`
	Facts    Facts    `json:"facts"`
	Claims   Claims   `json:"claims"`
}

// Audit records the snapshot the facts were taken at.
type Audit struct {
	Commit string `json:"commit"`
	Date   string `json:"date"`
	Claims int    `json:"claims"`
	Method string `json:"method"`
}

// Scope records what the inventory deliberately covers and excludes.
type Scope struct {
	In  []string `json:"in"`
	Out []string `json:"out"`
}

// BaselineEntry grandfathers an undeclared extraction finding.
type BaselineEntry struct {
	ID   string `json:"id"`
	Note string `json:"note"`
	Date string `json:"date"`
}

// Manifest is a loaded package.
type Manifest struct {
	Dir      string
	Audit    Audit
	Scope    Scope
	Sites    map[string]Site
	Baseline []BaselineEntry
}

// ids returns the declared site IDs, sorted.
func (m *Manifest) IDs() []string {
	out := make([]string, 0, len(m.Sites))
	for id := range m.Sites {
		out = append(out, id)
	}
	sort.Strings(out)

	return out
}

// BaselineIDs returns the grandfathered IDs (canonicalized elsewhere).
func (m *Manifest) BaselineIDs() []string {
	out := make([]string, 0, len(m.Baseline))
	for _, b := range m.Baseline {
		out = append(out, b.ID)
	}

	return out
}

// Load builds the CUE instance in dir and decodes the manifest.
//
// A missing directory is reported as ErrNotInitialized so callers can offer
// `init` instead of a stack-trace-style failure.
func Load(dir string) (*Manifest, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: %s (run `io-surface init` to scaffold one)", ErrNotInitialized, dir)
	}

	insts := load.Instances([]string{"."}, &load.Config{Dir: dir})
	if len(insts) != 1 {
		return nil, fmt.Errorf("loading %s: expected one instance, got %d", dir, len(insts))
	}
	inst := insts[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading %s: %w", dir, inst.Err)
	}

	value := cuecontext.New().BuildInstance(inst)
	// Constraint enforcement: disciplines.cue makes a non-empty violations
	// list fail unification, which surfaces here, before any check.
	if err = value.Err(); err != nil {
		return nil, fmt.Errorf("manifest constraints failed in %s: %w", dir, err)
	}

	m := &Manifest{Dir: dir}
	err = value.LookupPath(cue.ParsePath("sites")).Decode(&m.Sites)
	if err != nil {
		return nil, fmt.Errorf("decoding sites: %w", err)
	}
	if len(m.Sites) == 0 {
		return nil, fmt.Errorf("%w: %s declares no sites", ErrEmpty, dir)
	}
	// audit/scope/baseline are optional for a freshly scaffolded package.
	if err = value.LookupPath(cue.ParsePath("audit")).Decode(&m.Audit); err != nil {
		m.Audit = Audit{}
	}
	if err = value.LookupPath(cue.ParsePath("scope")).Decode(&m.Scope); err != nil {
		m.Scope = Scope{}
	}
	if err = value.LookupPath(cue.ParsePath("baseline")).Decode(&m.Baseline); err != nil {
		m.Baseline = nil
	}

	return m, nil
}

// Validate reports schema-level problems the CUE types cannot express cheaply:
// an unknown family, a missing location on an extractable site, an unknown
// discipline tag.
func (m *Manifest) Validate() []error {
	var errs []error
	famOK := map[string]bool{}
	for _, f := range Families {
		famOK[f] = true
	}
	discOK := map[string]bool{}
	for _, d := range Disciplines {
		discOK[d] = true
	}
	for _, id := range m.IDs() {
		s := m.Sites[id]
		if !famOK[s.Family] {
			errs = append(errs, fmt.Errorf("%s: unknown family %q", id, s.Family))
		}
		if s.Facts.Extractable && len(s.Location) == 0 {
			errs = append(errs, fmt.Errorf("%s: extractable site has no location", id))
		}
		for _, d := range s.Claims.Discipline {
			if !discOK[d] {
				errs = append(errs, fmt.Errorf("%s: unknown discipline %q", id, d))
			}
		}
		if strings.HasPrefix(id, "#") || strings.HasSuffix(id, ".") {
			errs = append(errs, fmt.Errorf("%s: malformed ID", id))
		}
	}
	if _, err := os.Stat(filepath.Join(m.Dir, "schema.cue")); err != nil {
		errs = append(errs, fmt.Errorf("%s: no schema.cue (is this an io-surface package?)", m.Dir))
	}

	return errs
}

// CountByFamily returns site counts per family (declared view).
func (m *Manifest) CountByFamily() map[string]int {
	out := map[string]int{}
	for _, s := range m.Sites {
		out[s.Family]++
	}

	return out
}
