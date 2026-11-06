package emit

import (
	"encoding/json"
	"testing"

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
