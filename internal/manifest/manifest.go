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
	// Owner is who vouches for the suppression (optional; team or handle).
	Owner string `json:"owner,omitempty"`
	// Expires is a YYYY-MM-DD date after which the gate fails on the entry:
	// grandfathering must not silently outlive its justification.
	Expires string `json:"expires,omitempty"`
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
	// list fail unification when it conflicts with a declared default. The
	// unified error is opaque ("incompatible list lengths"), so the actual
	// violation entries are recovered from the conjuncts and reported.
	if err = value.Err(); err != nil {
		return nil, constraintError(dir, value, err)
	}
	// Explicit emptiness enforcement: a disciplines.cue that only computes
	// `violations` (no conflicting default anywhere) unifies cleanly, so the
	// entries must be checked and reported directly.
	if entries := violationEntries(value); len(entries) > 0 {
		return nil, constraintError(dir, value, nil)
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

// Violation is one constraint failure produced by a disciplines.cue rule.
// The id/msg shape is the cross-repo convention (see the dnsblockd and
// gogenfilter manifests); anything undecodable is skipped rather than
// inventing a schema every repo must adopt.
type Violation struct {
	ID  string `json:"id"`
	Msg string `json:"msg"`
}

var violationsPath = cue.ParsePath("violations")

// constraintError reports failed manifest constraints. When the underlying
// failure is a unification conflict the entries live in a non-default
// conjunct, so violationEntries walks the operands; either way the user sees
// the actual id/msg violations instead of CUE's "incompatible list lengths".
func constraintError(dir string, value cue.Value, uniErr error) error {
	entries := violationEntries(value)
	if len(entries) == 0 {
		if uniErr == nil {
			return nil
		}

		return fmt.Errorf("%w in %s: %v", ErrConstraints, dir, uniErr)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "in %s", dir)
	if uniErr != nil {
		fmt.Fprintf(&b, ": %v", uniErr)
	}
	fmt.Fprintf(&b, " — %d constraint violation(s):", len(entries))
	seen := map[string]bool{}
	for _, v := range entries {
		key := v.ID + "|" + v.Msg
		if seen[key] {
			continue
		}
		seen[key] = true
		if v.ID != "" {
			fmt.Fprintf(&b, "\n  - %s: %s", v.ID, v.Msg)
		} else {
			fmt.Fprintf(&b, "\n  - %s", v.Msg)
		}
	}

	return fmt.Errorf("%w %s", ErrConstraints, b.String())
}

// violationEntries decodes the violations list, falling back to the top-level
// conjuncts when the unified value is erroneous: the offending operand still
// decodes even though the whole value does not.
func violationEntries(value cue.Value) []Violation {
	var out []Violation
	field := value.LookupPath(violationsPath)
	if field.Err() == nil {
		if err := field.Decode(&out); err == nil && len(out) > 0 {
			return out
		}
	}
	// The field errs on conflicting conjuncts (the "incompatible list
	// lengths" shape); decompose the field itself — every conjunct still
	// decodes, and the violating one carries the entries. A decode can be
	// partial when the conflict keeps a comprehension binding non-concrete:
	// keep whatever decoded (the msg always survives; the id survives
	// whenever CUE can resolve it).
	if _, operands := field.Expr(); operands != nil {
		for _, o := range operands {
			var entries []Violation
			if err := o.Decode(&entries); err == nil || len(entries) > 0 {
				out = append(out, entries...)
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	// File-level conjuncts of the whole value: when the conflict sits inside
	// one file's multiple declarations, that operand is skipped above, so
	// also probe per-file conjuncts that carry a clean violations field.
	if _, operands := value.Expr(); operands != nil {
		for _, o := range operands {
			sub := o.LookupPath(violationsPath)
			if sub.Err() != nil {
				continue
			}
			var entries []Violation
			if err := sub.Decode(&entries); err == nil {
				out = append(out, entries...)
			}
		}
	}

	return out
}

// Validate reports schema-level problems the CUE types cannot express cheaply:
// an unknown family, an unknown discipline tag, a malformed ID. These are hard
// errors: the manifest cannot be trusted until they are fixed.
//
// Missing locations on extractable sites are NOT here — they are findings
// (check.RuleMissingLocation), because a real manifest can carry them while the
// next audit regenerates facts.cue.
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
