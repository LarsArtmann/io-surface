package absorb

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/larsartmann/io-surface/internal/capability"
	"github.com/larsartmann/io-surface/internal/manifest"
)

const fixtureReport = `{
  "capabilityInfo": [
    {"capabilityName": "FILES", "capabilityType": "CAPABILITY_TYPE_DIRECT", "packageDir": "example.com/repo/internal/store", "depPath": "x", "path": []},
    {"capabilityName": "NETWORK", "capabilityType": "CAPABILITY_TYPE_DIRECT", "packageDir": "example.com/repo/internal/store", "depPath": "x", "path": []},
    {"capabilityName": "EXEC", "capabilityType": "CAPABILITY_TYPE_TRANSITIVE", "packageDir": "example.com/repo/internal/store", "depPath": "x", "path": []},
    {"capabilityName": "REFLECT", "capabilityType": "CAPABILITY_TYPE_TRANSITIVE", "packageDir": "example.com/repo/internal/store", "depPath": "x", "path": []},
    {"capabilityName": "FILES", "capabilityType": "CAPABILITY_TYPE_TRANSITIVE", "packageDir": "cuelang.org/go/cue", "depPath": "x", "path": []},
    {"capabilityName": "UNANALYZED", "capabilityType": "CAPABILITY_TYPE_TRANSITIVE", "packageDir": "example.com/repo/internal/render", "depPath": "x", "path": []}
  ]
}`

func writeGoMod(t *testing.T, module string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestModulePath(t *testing.T) {
	dir := writeGoMod(t, "example.com/repo")
	got, err := ModulePath(dir)
	if err != nil {
		t.Fatalf("ModulePath: %v", err)
	}
	if got != "example.com/repo" {
		t.Errorf("ModulePath = %q", got)
	}
	if _, err := ModulePath(t.TempDir()); err == nil {
		t.Error("missing go.mod must error")
	}
}

func TestRun(t *testing.T) {
	root := writeGoMod(t, "example.com/repo")
	res, err := Run(root, []byte(fixtureReport), Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Pkgs != 1 {
		t.Fatalf("Pkgs = %d, want 1 (third-party and unmapped-only packages omitted)", res.Pkgs)
	}
	set := res.Sets[0]
	if set.Pkg != "internal/store" {
		t.Errorf("pkg = %q", set.Pkg)
	}
	if want := []capability.Name{"FILES", "NETWORK"}; !reflect.DeepEqual(set.Direct, want) {
		t.Errorf("direct = %v, want %v", set.Direct, want)
	}
	if want := []capability.Name{"EXEC", "FILES", "NETWORK"}; !reflect.DeepEqual(set.Deep, want) {
		t.Errorf("deep = %v, want %v", set.Deep, want)
	}
	if res.Dropped["REFLECT"] != 1 || res.Dropped["UNANALYZED"] != 1 {
		t.Errorf("dropped = %v, want REFLECT:1 UNANALYZED:1", res.Dropped)
	}
}

func TestRunRejectsUnknownType(t *testing.T) {
	root := writeGoMod(t, "example.com/repo")
	bad := `{"capabilityInfo": [{"capabilityName": "FILES", "capabilityType": "CAPABILITY_TYPE_HOLOGRAPHIC", "packageDir": "example.com/repo/internal/x"}]}`
	if _, err := Run(root, []byte(bad), Options{}); err == nil || !strings.Contains(err.Error(), "unknown capability type") {
		t.Fatalf("want unknown-type error, got %v", err)
	}
}

func TestRunModuleOverride(t *testing.T) {
	res, err := Run(t.TempDir(), []byte(fixtureReport), Options{ModulePath: "example.com/repo"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Pkgs != 1 {
		t.Fatalf("Pkgs = %d", res.Pkgs)
	}
}

func TestCUEBytes(t *testing.T) {
	root := writeGoMod(t, "example.com/repo")
	res, err := Run(root, []byte(fixtureReport), Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	body := string(CUE(res, Audit{Engine: "capslock", Version: "v0.3.4", Commit: "abc1234", Date: "2026-11-06"}))
	for _, want := range []string{
		`engine:  "capslock"`,
		`version: "v0.3.4"`,
		`commit:  "abc1234"`,
		`pkg: "internal/store", direct: ["FILES", "NETWORK"], deep: ["EXEC", "FILES", "NETWORK"]`,
		"package io_surface",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("CUE body missing %q\n---\n%s", want, body)
		}
	}
	if strings.Contains(body, "REFLECT") || strings.Contains(body, "claims:") || strings.Contains(body, "baseline:") {
		t.Error("capabilities.cue must carry only capability facts, never judgment sections")
	}
}

func TestResolveAuditDefaults(t *testing.T) {
	got := ResolveAudit("", Audit{Version: "v1"})
	if got.Engine != "capslock" || got.Version != "v1" || got.Date == "" {
		t.Errorf("ResolveAudit = %+v", got)
	}
	if got.Commit != "working-tree" && len(got.Commit) < 7 {
		t.Errorf("commit %q is neither the fallback nor a sha", got.Commit)
	}
}

func TestDrift(t *testing.T) {
	m := &manifest.Manifest{
		Capabilities: []manifest.Capability{
			{Pkg: "internal/store", Direct: []string{"FILES"}, Deep: []string{"FILES"}},
			{Pkg: "internal/ghost", Direct: nil, Deep: []string{"EXEC"}},
		},
	}
	res := &Result{Sets: []capability.Set{
		{Pkg: "internal/store", Direct: []capability.Name{"FILES", "NETWORK"}, Deep: []capability.Name{"FILES", "NETWORK"}},
		{Pkg: "internal/fresh", Deep: []capability.Name{"FILES"}},
	}}
	got := Drift(m, res)
	if len(got) != 3 {
		t.Fatalf("drift lines = %v, want 3", got)
	}
	if !strings.Contains(got[0], "internal/fresh") || !strings.Contains(got[0], "missing") {
		t.Errorf("first line should report the fresh package, got %q", got[0])
	}
	if !strings.Contains(got[1], "internal/ghost") || !strings.Contains(got[1], "no longer observed") {
		t.Errorf("second line should report the vanished package, got %q", got[1])
	}
	if !strings.Contains(got[2], "internal/store") || !strings.Contains(got[2], "≠") {
		t.Errorf("third line should report the changed package, got %q", got[2])
	}

	res.Sets = []capability.Set{
		{Pkg: "internal/store", Direct: []capability.Name{"FILES"}, Deep: []capability.Name{"FILES"}},
		{Pkg: "internal/ghost", Deep: []capability.Name{"EXEC"}},
	}
	if got := Drift(m, res); len(got) != 0 {
		t.Errorf("identical sets must produce no drift, got %v", got)
	}
}
