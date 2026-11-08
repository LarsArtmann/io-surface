package emit

import (
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/io-surface/internal/absorb"
	"github.com/larsartmann/io-surface/internal/check"
)

func sampleReport() *check.Report {
	return &check.Report{
		Stats: check.Stats{Declared: 2, Extracted: 2, Errors: 1, Warnings: 1, Infos: 1},
		Findings: []check.Finding{
			{
				Rule:     check.RuleUndeclared,
				Severity: check.SeverityError,
				ID:       "internal/store.Persist",
				Family:   "file-write",
				Location: "internal/store/store.go:31",
				Message:  "undeclared I/O site (os.WriteFile)",
			},
			{
				Rule:     check.RuleLocationDrift,
				Severity: check.SeverityInfo,
				ID:       "cmd/io-surface.runRender",
				Location: "cmd/io-surface/main.go:228",
				Message:  "declared location differs from the extraction",
			},
			{
				Rule:     check.RuleManifestValidation,
				Severity: check.SeverityWarning,
				ID:       "manifest",
				Location: "",
				Message:  "no schema.cue",
			},
		},
	}
}

// TestToFindings_PreservesContract pins the mapping guarantees: rule IDs and
// severities survive verbatim (they are the suppression/SARIF contract), every
// finding gets a go-finding ID, and "path:line" parses into a position.
func TestToFindings_PreservesContract(t *testing.T) {
	fs := ToFindings(sampleReport())
	if len(fs) != 3 {
		t.Fatalf("got %d findings, want 3", len(fs))
	}

	first := fs[0]
	if string(first.Rule) != check.RuleUndeclared {
		t.Errorf("rule mutated: %q", first.Rule)
	}
	if first.Severity != "error" {
		t.Errorf("severity mutated: %q", first.Severity)
	}
	if first.ToolName != ToolName {
		t.Errorf("tool name: %q", first.ToolName)
	}
	if first.ID == "" {
		t.Error("go-finding ID not generated")
	}
	if first.Position.File != "internal/store/store.go" || first.Position.Line != 31 {
		t.Errorf("position: %+v", first.Position)
	}

	last := fs[2]
	if last.Position.Line != 0 {
		t.Errorf("location-less finding should carry line 0, got %d", last.Position.Line)
	}
}

// TestSARIF_DocumentShape parses the emitted document and asserts the parts
// GitHub's SARIF renderer keys on: tool driver name, one result per finding,
// ruleIds identical to the contract strings.
func TestSARIF_DocumentShape(t *testing.T) {
	data, err := SARIF(sampleReport(), "0.1.0")
	if err != nil {
		t.Fatalf("SARIF: %v", err)
	}

	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("SARIF output is not JSON: %v\n%s", err, data)
	}
	if len(doc.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(doc.Runs))
	}
	run := doc.Runs[0]
	if run.Tool.Driver.Name != ToolName {
		t.Errorf("driver name: %q", run.Tool.Driver.Name)
	}
	if len(run.Results) != 3 {
		t.Fatalf("results = %d, want 3", len(run.Results))
	}
	if run.Results[0].RuleID != check.RuleUndeclared {
		t.Errorf("ruleId changed: %q", run.Results[0].RuleID)
	}
	if run.Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI != "internal/store/store.go" {
		t.Errorf("uri: %q", run.Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI)
	}
}

// TestSARIF_EnforcementProperty pins the D4 posture contract (TODO #93):
// every result of a run carries the report's enforcement policy as the
// `go-finding/meta/enforcement` property, for BOTH postures, so a consumer
// can tell a gate run from an audit run off the artifact alone.
func TestSARIF_EnforcementProperty(t *testing.T) {
	for _, posture := range []string{"gate", "audit"} {
		rep := sampleReport()
		rep.Enforcement = posture
		data, err := SARIF(rep, "0.1.0")
		if err != nil {
			t.Fatalf("SARIF(%s): %v", posture, err)
		}

		var doc struct {
			Runs []struct {
				Results []struct {
					RuleID     string         `json:"ruleId"`
					Properties map[string]any `json:"properties"`
				} `json:"results"`
			} `json:"runs"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse SARIF(%s): %v", posture, err)
		}
		if len(doc.Runs) != 1 || len(doc.Runs[0].Results) != 3 {
			t.Fatalf("posture %s: want one run with 3 results", posture)
		}
		for _, res := range doc.Runs[0].Results {
			got, ok := res.Properties["go-finding/meta/enforcement"]
			if !ok {
				t.Errorf("posture %s, rule %s: no enforcement property", posture, res.RuleID)
				continue
			}
			if got != posture {
				t.Errorf("posture %s, rule %s: enforcement = %v", posture, res.RuleID, got)
			}
		}
	}
}

// TestToFindings_EnforcementRidesMetadata checks the source of the SARIF
// property: the report's posture lands in every finding's metadata, merged
// with (not clobbering) the detector rule description.
func TestToFindings_EnforcementRidesMetadata(t *testing.T) {
	rep := sampleReport()
	rep.Enforcement = "audit"
	rep.Findings[0].Rule = "io-surface/err-dropped" // detector rule → gets ruleDescription
	fs := ToFindings(rep)
	if len(fs) != 3 {
		t.Fatalf("got %d findings, want 3", len(fs))
	}
	first := fs[0]
	if first.Metadata["enforcement"] != "audit" {
		t.Errorf("enforcement metadata: %q", first.Metadata["enforcement"])
	}
	if first.Metadata["ruleDescription"] == "" {
		t.Error("ruleDescription lost when enforcement rides along")
	}

	rep.Enforcement = ""
	fs = ToFindings(rep)
	if _, ok := fs[0].Metadata["enforcement"]; ok {
		t.Error("empty enforcement must not emit the property")
	}
}

// TestSARIF_Deterministic matters for golden-style consumers and caches.
func TestSARIF_Deterministic(t *testing.T) {
	a, err := SARIF(sampleReport(), "0.1.0")
	if err != nil {
		t.Fatalf("SARIF a: %v", err)
	}
	b, err := SARIF(sampleReport(), "0.1.0")
	if err != nil {
		t.Fatalf("SARIF b: %v", err)
	}
	if string(a) != string(b) {
		t.Error("SARIF output is not deterministic")
	}
}

// TestSARIF_CapabilityRuleIDsFlowByteStable pins the v1.1 contract end to
// end: the advisory capability rules ride the same SARIF stream unchanged,
// and the whole document is byte-identical across runs (M55).
func TestSARIF_CapabilityRuleIDsFlowByteStable(t *testing.T) {
	rep := &check.Report{
		Stats: check.Stats{Infos: 2},
		Findings: []check.Finding{
			{
				Rule:     check.RuleShadowCapability,
				Severity: check.SeverityInfo,
				ID:       "internal/store",
				Message:  "reachability engine observes NETWORK in this package, but no declared site covers those families",
			},
			{
				Rule:     absorb.RuleCapabilityDrift,
				Severity: check.SeverityInfo,
				ID:       "internal/store",
				Message:  "committed deep [FILES] ≠ fresh deep [EXEC, FILES]",
			},
		},
	}
	first, err := SARIF(rep, "test")
	if err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	second, err := SARIF(rep, "test")
	if err != nil {
		t.Fatalf("SARIF again: %v", err)
	}
	if string(first) != string(second) {
		t.Error("SARIF output must be byte-stable for an identical report")
	}
	var doc struct {
		Runs []struct {
			Results []struct {
				RuleID string `json:"ruleId"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(first, &doc); err != nil {
		t.Fatalf("parse SARIF: %v", err)
	}
	if len(doc.Runs) != 1 || len(doc.Runs[0].Results) != 2 {
		t.Fatalf("want one run with 2 results, got %+v", doc)
	}
	got := map[string]bool{}
	for _, r := range doc.Runs[0].Results {
		got[r.RuleID] = true
	}
	if !got[check.RuleShadowCapability] || !got[absorb.RuleCapabilityDrift] {
		t.Errorf("capability ruleIds must pass through unchanged, got %v", got)
	}
}
