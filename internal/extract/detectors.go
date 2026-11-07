package extract

// Detector rules: pure functions over one resolved call capture that emit
// observations. Every rule is data-driven (tables in tables.go, D6) and fires
// at info severity until fleet evidence promotes it (D4). The registry and
// the metadata table below are the two halves of the rule contract; a parity
// test pins them together.

import ()

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
var detectors = []detector{}

// detectorRules is the metadata registry. Message receives the observation's
// Evidence clause and returns the complete human-facing finding message.
var detectorRules = []ruleDef{}

// detectorRule looks up metadata by rule name.
func detectorRule(name string) (ruleDef, bool) {
	for _, r := range detectorRules {
		if r.Name == name {
			return r, true
		}
	}

	return ruleDef{}, false
}
