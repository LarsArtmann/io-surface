package extract

// Detector rules: pure functions over one resolved call capture that emit
// observations. Every rule is data-driven (tables in tables.go, D6) and fires
// at info severity until fleet evidence promotes it (D4). The registry and
// the metadata table below are the two halves of the rule contract; a parity
// test pins them together.

import "slices"

// detector is one registered rule: name + the function that produces its
// observations for a single capture.
type detector struct {
	rule string
	run  func(e *extractor, c *capture) []Observation
}

// ruleDef is the published half of the rule contract: the stable name, the
// current severity, and the finding message builder the lint face uses.
type ruleDef struct {
	Name        string
	Severity    string // "info" until T17 evidence-gated promotion (D4)
	Description string
	Message     func(o Observation) string
}

// detectors is the execution registry (order = observation emission order per
// capture; global ordering is imposed by sortedObservations).
var detectors = []detector{
	{rule: RuleErrDropped, run: detectErrDropped},
}

// detectorRules is the metadata registry. Message receives the observation's
// Evidence clause and returns the complete human-facing finding message.
var detectorRules = []ruleDef{
	{
		Name:        RuleErrDropped,
		Severity:    "info",
		Description: "a watched call's error return is discarded",
		Message: func(o Observation) string {
			return o.Evidence + " — the error return is unchecked; handle it or ignore deliberately with `_ =`"
		},
	},
}

// detectorRule looks up metadata by rule name.
func detectorRule(name string) (ruleDef, bool) {
	for _, r := range detectorRules {
		if r.Name == name {
			return r, true
		}
	}

	return ruleDef{}, false
}

// detectErrDropped implements D8 exactly: a blank error slot in a named
// assignment is flagged (`x, _ := os.ReadFile(p)`), a fully blank assignment
// is a deliberate ignore (`_ = os.Remove(p)`) and stays silent, bare
// statement calls drop everything and are flagged, while returned, deferred
// (cleanup defers legitimately ignore), and expression-embedded calls never
// fire.
func detectErrDropped(e *extractor, c *capture) []Observation {
	errPos, watched := errResults[c.sel]
	if !watched {
		return nil
	}
	switch c.shape {
	case shapeBare:
		return []Observation{e.emit(c, RuleErrDropped, evidenceOf(&e.fset, c.node)+" (results discarded)")}
	case shapeAssign:
		if len(c.blanks) == c.slots {
			return nil // D8: every slot blank is a deliberate ignore
		}
		for _, p := range errPos {
			if containsInt(c.blanks, c.slotIdx+p) {
				return []Observation{e.emit(c, RuleErrDropped, evidenceOf(&e.fset, c.parent))}
			}
		}
	}
	return nil
}

func containsInt(list []int, v int) bool {
	return slices.Contains(list, v)
}
