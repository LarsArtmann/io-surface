package main

// Contract tests moved out of main_test.go for the 650-line file gate:
// describe output shape, lint-flag surface, version JSON provenance.

import (
	"encoding/json"
	"testing"
)

// TestDescribe_Contract pins the machine-readable CLI contract (TODO #53):
// schema_version stays 1 within the major, and every user-facing command is
// present — agents and tooling drive the CLI through this document, so a
// silently missing command would break them.
func TestDescribe_Contract(t *testing.T) {
	out, err := runApp(t, "describe")
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	var doc struct {
		SchemaVersion string `json:"schema_version"`
		Name          string `json:"name"`
		Commands      []struct {
			Path     string `json:"path"`
			Commands []struct {
				Path  string `json:"path"`
				Flags []struct {
					Name string `json:"name"`
				} `json:"flags"`
			} `json:"commands"`
			Flags []struct {
				Name string `json:"name"`
			} `json:"flags"`
		} `json:"commands"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("describe output not JSON: %v\n%s", err, out)
	}
	if doc.SchemaVersion != "1" {
		t.Errorf("schema_version = %q, want 1 (additive within the major)", doc.SchemaVersion)
	}
	paths := map[string]bool{}
	for _, c := range doc.Commands {
		paths[c.Path] = true
		for _, sub := range c.Commands {
			paths[sub.Path] = true
		}
	}
	for _, want := range []string{"io-surface check", "io-surface init", "io-surface render", "io-surface extract", "io-surface lint", "io-surface absorb", "io-surface baseline", "io-surface baseline add", "io-surface version"} {
		if !paths[want] {
			t.Errorf("describe contract lost %q (has %d commands)", want, len(paths))
		}
	}
	// The lint face's contract surface (T15): its flags are part of the
	// machine-readable contract, so a renamed or dropped flag must break
	// this test, not a fleet CI step.
	var lintFlags map[string]bool
	for i := range doc.Commands {
		c := doc.Commands[i]
		if c.Path == "io-surface lint" {
			lintFlags = map[string]bool{}
			for _, fl := range c.Flags {
				lintFlags[fl.Name] = true
			}

			break
		}
		for _, sub := range c.Commands {
			if sub.Path == "io-surface lint" {
				lintFlags = map[string]bool{}
				for _, fl := range sub.Flags {
					lintFlags[fl.Name] = true
				}

				break
			}
		}
	}
	if lintFlags == nil {
		t.Fatal("io-surface lint not found in describe output")
	}
	for _, want := range []string{"format", "enforce", "include-tests", "manifest"} {
		if !lintFlags[want] {
			t.Errorf("describe contract lost lint flag --%s (fleet consumers depend on it)", want)
		}
	}
}

// TestVersionJSON pins the --json provenance shape: full keys even when the
// build carried no VCS stamp, and the plain output stays byte-identical.
func TestVersionJSON(t *testing.T) {
	out, err := runApp(t, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var doc map[string]string
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if doc["version"] != version {
		t.Errorf("version = %q, want %q", doc["version"], version)
	}
	if _, ok := doc["commit"]; !ok {
		t.Error("commit key missing (must exist even when unstamped)")
	}
	if _, ok := doc["date"]; !ok {
		t.Error("date key missing (must exist even when unstamped)")
	}
}
