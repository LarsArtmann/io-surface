package extract

// Suppression: the per-site, reason-mandatory ignore directive for detector
// findings (superb plan T5, decision D2). A lint hit that is correct-as-actor
// (deliberate cleanup unlink, force-kill path) is silenced IN the source, one
// rule at a time, with the reason sitting in the review diff — the audit
// trail is the code history itself, so no only-shrinks tooling is needed.
//
// Grammar (own line above the call, or trailing on the call's first line):
//
//	//io-surface:ignore <rule> <reason…>
//
// A directive governs the flagged call on its own line or on the next line
// (the convention above-the-call). File-level ignores are rejected BY
// ABSENCE of a mechanism: too blunt to earn trust. A malformed directive
// (unknown rule, empty reason) never suppresses — it emits its own
// ignore-invalid finding so the mistake is loud, not silent.

import (
	"go/ast"
	"path/filepath"
	"strconv"
	"strings"
)

// IgnoreDirectivePrefix is the comment grammar's literal prefix.
const IgnoreDirectivePrefix = "io-surface:ignore"

// ignoreDirective is one parsed directive. valid=false entries are kept only
// to report the malformation; they never suppress.
type ignoreDirective struct {
	file   string // repo-relative, slash-separated (the location contract)
	line   int    // directive's own line
	rule   string
	reason string
	text   string // the raw comment, for the invalid finding's evidence
}

// collectIgnores scans one parsed file's comments for suppression directives
// and registers them (valid) or queues an ignore-invalid observation
// (malformed). Called from parseFile — the file is already parsed with
// ParseComments, so the collection costs no extra I/O. Plain Extract keeps
// the maps nil and the scan skipped.
func (e *extractor) collectIgnores(file *ast.File) {
	if e.ignores == nil {
		return
	}
	known := knownRuleNames()
	for _, group := range file.Comments {
		for _, cm := range group.List {
			for i, line := range strings.Split(cm.Text, "\n") {
				trimmed := directiveLine(line)
				if !strings.HasPrefix(trimmed, IgnoreDirectivePrefix) {
					continue
				}
				pos := e.fset.Position(cm.Pos())
				pos.Line += i
				dir := ignoreDirective{file: e.relPath(pos.Filename), line: pos.Line, text: trimmed}
				rest := strings.TrimSpace(strings.TrimPrefix(trimmed, IgnoreDirectivePrefix))
				rule, reason, _ := strings.Cut(rest, " ")
				reason = strings.TrimSpace(reason)
				if _, ok := known[rule]; !ok || reason == "" {
					dir.rule, dir.reason = rule, reason
					e.invalidIgnores = append(e.invalidIgnores, dir)

					continue
				}
				dir.rule, dir.reason = rule, reason
				e.ignores[dir.file] = append(e.ignores[dir.file], dir)
			}
		}
	}
}

// directiveLine strips comment markers from one source line and returns the
// directive text if the line carries the prefix (empty string otherwise).
func directiveLine(line string) string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "//")
	if strings.HasPrefix(s, "/*") {
		s = strings.TrimPrefix(s, "/*")
	}
	if strings.HasSuffix(strings.TrimSpace(s), "*/") {
		s = strings.TrimSuffix(strings.TrimSpace(s), "*/")
	}

	return strings.TrimSpace(s)
}

// filterSuppressed drops observations covered by a valid directive: same
// file, rule match, directive on the flagged line or the line directly above
// it. Returns the input slice when nothing is suppressed.
func (e *extractor) filterSuppressed(obs []Observation) []Observation {
	if len(e.ignores) == 0 {
		return obs
	}
	kept := make([]Observation, 0, len(obs))
	for _, o := range obs {
		if !e.suppressed(o) {
			kept = append(kept, o)
		}
	}

	return kept
}

// suppressed reports whether one observation is covered by a directive.
func (e *extractor) suppressed(o Observation) bool {
	file, lineStr, ok := strings.Cut(o.Location, ":")
	if !ok {
		return false
	}
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return false
	}
	for _, d := range e.ignores[file] {
		if d.rule == o.Rule && (d.line == line || d.line == line-1) {
			return true
		}
	}

	return false
}

// invalidIgnoreObservations renders every malformed directive as its own
// finding (one per directive, in scan order; sortedObservations normalizes).
func (e *extractor) invalidIgnoreObservations() []Observation {
	if len(e.invalidIgnores) == 0 {
		return nil
	}
	out := make([]Observation, 0, len(e.invalidIgnores))
	for _, d := range e.invalidIgnores {
		why := "unknown rule name"
		if d.reason == "" {
			why = "missing reason"
		}
		out = append(out, Observation{
			// SiteID anchors at the file (a directive is not tied to a
			// capture); the finding contract requires a non-empty ID.
			SiteID:   d.file,
			Rule:     RuleIgnoreInvalid,
			Call:     why,
			Location: d.file + ":" + strconv.Itoa(d.line),
			Evidence: d.text,
		})
	}

	return out
}

// relPath formats an absolute path as the repo-relative, slash-separated
// location path the manifest contract keys on (same formatting as locOf).
func (e *extractor) relPath(abs string) string {
	rel := abs
	if r, err := filepath.Rel(e.root, abs); err == nil {
		rel = filepath.ToSlash(r)
	}

	return rel
}

// knownRuleNames indexes the published rule vocabulary for directive
// validation.
func knownRuleNames() map[string]bool {
	out := map[string]bool{}
	for _, r := range DetectorRules() {
		out[r.Name] = true
	}

	return out
}
