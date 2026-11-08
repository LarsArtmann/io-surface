// Package manifest loads an io-surface CUE manifest package (io-surface/v1,
// v1.1, and v1.2): schema, facts, claims, disciplines, baseline, scope, and —
// since v1.1 — the engine-derived capabilities section.
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
	"github.com/larsartmann/io-surface/internal/capability"
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

// Dir is the conventional manifest location inside a repository (relative to
// the repo root).
const Dir = "docs/architecture/io-surface"

// Claims are human judgment: never overwritten by tooling.
type Claims struct {
	Resource   string   `json:"resource"`
	Trigger    string   `json:"trigger"`
	Discipline []string `json:"discipline"`
	Exception  string   `json:"exception"`
	Rationale  string   `json:"rationale"`
	// Severity-modulation inputs (v1.2). Zero values mean "unclaimed"; the
	// detector severity modulation treats unclaimed as the schema defaults
	// (normal / recoverable) without manifest edits.
	Criticality string `json:"criticality,omitempty"`
	Durability  string `json:"durability,omitempty"`
}

// Criticality/durability vocabularies (schema v1.2). The CUE closed enums in
// schema.cue are the standard; these exist so the Go side can validate and
// the detector modulation can escalate without stringly-typed guessing.
const (
	CriticalityLow      = "low"
	CriticalityNormal   = "normal"
	CriticalityCritical = "critical"

	DurabilityEphemeral   = "ephemeral"
	DurabilityRecoverable = "recoverable"
	DurabilityDurable     = "durable"
)

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
	// Version is the declared contract version ("" in v1 manifests).
	Version string
	// Enforcement is the recorded gate policy (D4): "gate" — error-severity
	// findings fail the build; "audit" — findings are reported but never
	// fail (adoption posture). "" normalizes to "gate". The choice lives in
	// the manifest so an opt-out is a committed, reviewable diff, never a
	// silent CI edit.
	Enforcement string
	// Capabilities are the v1.1 reachability facts (nil when absent).
	Capabilities []Capability
	// CapabilityAudit is the provenance of the last absorb run (nil when the
	// manifest carries no capabilities yet).
	CapabilityAudit *CapabilityAudit
}

// Enforcement policy values (D4).
const (
	EnforcementGate  = "gate"
	EnforcementAudit = "audit"
)

// ContractVersionMax is the newest manifest contract this tool fully
// understands (schema.cue's `version` enum, TODO #116). SupportedVersions is
// the closed set Load accepts; anything else is refused loudly instead of
// mis-decoded, because sites written for a newer contract can carry fields
// and vocabulary this tool has no meaning for.
const SupportedVersions = "io-surface/v1, io-surface/v1.1, io-surface/v1.2"

// versionsSupported is the lookup set behind SupportedVersions; keep in sync
// with the closed enum in internal/scaffold/schema.cue (the schema refuses
// unknown versions at unification when present — this guard covers manifests
// whose schema.cue is stale or missing).
var versionsSupported = map[string]bool{
	"io-surface/v1":   true,
	"io-surface/v1.1": true,
	"io-surface/v1.2": true,
}

// checkVersion refuses manifests declaring a contract version beyond the
// shipped set. Absent stays legal: v1 manifests predate the field.
func checkVersion(version string) error {
	if version == "" || versionsSupported[version] {
		return nil
	}

	return fmt.Errorf("%w: %q (this tool supports %s; upgrade io-surface, or pin the repo to the matching release)", ErrUnsupportedVersion, version, SupportedVersions)
}

// EnforcementPolicy returns the manifest's recorded policy, normalized:
// an absent field means "gate" (the birth decision — new undeclared sites
// fail; a repo opts out by changing the manifest, visibly).
func (m *Manifest) EnforcementPolicy() string {
	if m.Enforcement == EnforcementAudit {
		return EnforcementAudit
	}

	return EnforcementGate
}

// Capability is one package's reachability set observed by the external
// engine. Deep is always a superset of Direct: deep includes transitive
// dependency paths, direct only the package's own code.
type Capability struct {
	Pkg    string   `json:"pkg"`
	Direct []string `json:"direct"`
	Deep   []string `json:"deep"`
}

// CapabilityAudit records which engine run produced the capabilities.
type CapabilityAudit struct {
	Engine  string `json:"engine"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
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
		return nil, fmt.Errorf("decoding sites in %s: %w (facts.cue and claims.cue both contribute to sites — check the file you edited last)", dir, err)
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
	// v1.1 sections are optional: a v1 manifest loads unchanged.
	_ = value.LookupPath(cue.ParsePath("version")).Decode(&m.Version)
	_ = value.LookupPath(cue.ParsePath("enforcement")).Decode(&m.Enforcement)
	if err := checkVersion(m.Version); err != nil {
		return nil, fmt.Errorf("loading %s: %w", dir, err)
	}
	if err = value.LookupPath(cue.ParsePath("capabilities")).Decode(&m.Capabilities); err != nil {
		m.Capabilities = nil
	}
	var capAudit CapabilityAudit
	if err = value.LookupPath(cue.ParsePath("capabilityAudit")).Decode(&capAudit); err != nil {
		m.CapabilityAudit = nil
	} else {
		m.CapabilityAudit = &capAudit
	}

	return m, nil
}

// Violation is one constraint failure produced by a disciplines.cue rule.
// The id/msg shape is the cross-repo convention (see the dnsblockd and
// gogenfilter manifests); anything undecodable is skipped rather than
// inventing a schema every repo must adopt.
type Violation struct {
	ID   string `json:"id"`
	Msg  string `json:"msg"`
	Path string `json:"path,omitempty"` // CUE path of the entry (T11): anchors the violation even when id/msg decode empty
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
		key := v.ID + "|" + v.Msg + "|" + v.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		where := ""
		if v.Path != "" {
			where = " (at " + v.Path + ")"
		}
		switch {
		case v.ID != "":
			fmt.Fprintf(&b, "\n  - %s: %s%s", v.ID, v.Msg, where)
		case v.Msg != "":
			fmt.Fprintf(&b, "\n  - %s%s", v.Msg, where)
		default:
			// id and msg both decode empty (a foreign violations schema):
			// the CUE path is then the only locator the user has.
			fmt.Fprintf(&b, "\n  - constraint at %s", where)
		}
	}

	return fmt.Errorf("%w %s", ErrConstraints, b.String())
}

// violationEntries recovers the actual violations from a value whose
// unification failed: one deterministic walk, widest scope first. Each stage
// stops as soon as entries decode, and every stage dedupes on id+msg so the
// overlapping shapes a conflict produces (whole-value operands, field
// operands, per-file conjuncts) cannot double-report one violation.
//
// Stage 1 is the plain decoded list (the common, non-conflicting shape).
// Stage 2 decomposes the FIELD: on "incompatible list lengths" every conjunct
// still decodes, and the violating one carries the entries. A decode can be
// partial when the conflict keeps a comprehension binding non-concrete — the
// msg always survives, the id whenever CUE can resolve it. Stage 3 probes
// per-file conjuncts of the whole value for a clean violations field, for the
// shape where the conflict sits inside one file's multiple declarations.
func violationEntries(value cue.Value) []Violation {
	seen := map[string]bool{}
	// contentful drops decode artifacts: a violations-shaped value that is a
	// list CONCAT (dnsblockd's disciplines.cue) exposes its comprehension
	// sub-lists as operands, and decoding those against []Violation yields
	// contentless structs that look like violations but name nothing (the
	// pre-T11 "opaque ` - .` entry" bug). A real violation always carries an
	// id or a msg; an entry with neither is noise, not a finding.
	contentful := func(entries []Violation) []Violation {
		kept := entries[:0]
		for _, v := range entries {
			if v.ID != "" || v.Msg != "" {
				kept = append(kept, v)
			}
		}

		return kept
	}
	// entriesWithPaths decodes one list-typed value element-wise when the
	// list resolves (per-entry CUE paths survive), falling back to a whole
	// decode annotated with the list's own path (the conflict hides the
	// elements but the locator still names the violations field).
	entriesWithPaths := func(v cue.Value) ([]Violation, bool) {
		if elems := listElements(v); len(elems) > 0 {
			var entries []Violation
			for _, el := range elems {
				var one Violation
				if err := el.Decode(&one); err != nil {
					continue
				}
				one.Path = el.Path().String()
				entries = append(entries, one)
			}
			if len(entries) > 0 {
				return entries, true
			}
		}
		var entries []Violation
		if err := v.Decode(&entries); err != nil && len(entries) == 0 {
			return nil, false
		}
		if p := v.Path().String(); p != "" {
			for i := range entries {
				if entries[i].Path == "" {
					entries[i].Path = p
				}
			}
		}

		return entries, len(entries) > 0
	}

	var out []Violation
	field := value.LookupPath(violationsPath)
	if entries, ok := entriesWithPaths(field); ok {
		return deduped(out, contentful(entries), seen)
	}
	if _, operands := field.Expr(); operands != nil {
		for _, o := range operands {
			if entries, ok := entriesWithPaths(o); ok {
				out = append(out, entries...)
			}
		}
	}
	if len(contentful(out)) > 0 {
		return contentful(out)
	}
	if _, operands := value.Expr(); operands != nil {
		for _, o := range operands {
			sub := o.LookupPath(violationsPath)
			if sub.Err() != nil {
				continue
			}
			if entries, ok := entriesWithPaths(sub); ok {
				out = append(out, entries...)
			}
		}
	}

	return contentful(out)
}

// listElements returns the element values of a list-typed CUE value, empty
// when the value is not a resolvable list (conflict shapes).
func listElements(v cue.Value) []cue.Value {
	it, err := v.List()
	if err != nil {
		return nil
	}
	var out []cue.Value
	for it.Next() {
		out = append(out, it.Value())
	}

	return out
}

// deduped appends entries that the seen-set has not recorded yet, marking
// them as it goes. One dedupe, one place — the single source of the
// id+msg identity constraintError prints.
func deduped(out, entries []Violation, seen map[string]bool) []Violation {
	for _, v := range entries {
		key := v.ID + "|" + v.Msg
		if !seen[key] {
			seen[key] = true
			out = append(out, v)
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
	if e := m.Enforcement; e != "" && e != EnforcementGate && e != EnforcementAudit {
		errs = append(errs, fmt.Errorf("enforcement: unknown policy %q (want %q or %q)", e, EnforcementGate, EnforcementAudit))
	}

	errs = append(errs, m.validateCapabilities()...)

	return errs
}

// validateCapabilities enforces the v1.1 invariants Go must hold even when an
// older schema.cue (without the closed enum) would let bad data unify: names
// inside the closed vocabulary, deep ⊇ direct, and one entry per package.
func (m *Manifest) validateCapabilities() []error {
	var errs []error
	seen := map[string]bool{}
	for _, c := range m.Capabilities {
		if c.Pkg == "" {
			errs = append(errs, fmt.Errorf("capabilities: empty pkg"))
		}
		if seen[c.Pkg] {
			errs = append(errs, fmt.Errorf("capabilities: duplicate pkg %q", c.Pkg))
		}
		seen[c.Pkg] = true
		deep := map[string]bool{}
		for _, n := range c.Deep {
			if !capability.Valid(n) {
				errs = append(errs, fmt.Errorf("capabilities %s: unknown capability %q", c.Pkg, n))
			}
			deep[n] = true
		}
		for _, n := range c.Direct {
			if !deep[n] {
				errs = append(errs, fmt.Errorf("capabilities %s: direct %q missing from deep", c.Pkg, n))
			}
			if !capability.Valid(n) {
				errs = append(errs, fmt.Errorf("capabilities %s: unknown capability %q", c.Pkg, n))
			}
		}
	}

	return errs
}

// CapabilityFor returns the reachability set for pkg, or nil.
func (m *Manifest) CapabilityFor(pkg string) *Capability {
	for i := range m.Capabilities {
		if m.Capabilities[i].Pkg == pkg {
			return &m.Capabilities[i]
		}
	}

	return nil
}

// CountByFamily returns site counts per family (declared view).
func (m *Manifest) CountByFamily() map[string]int {
	out := map[string]int{}
	for _, s := range m.Sites {
		out[s.Family]++
	}

	return out
}
