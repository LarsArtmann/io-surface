package capability

import (
	"reflect"
	"testing"
)

func TestValid(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"FILES", true},
		{"NETWORK", true},
		{"EXEC", true},
		// v1.3 closed ten: the runtime-power set is valid (stored at package level).
		{"REFLECT", true},
		{"RUNTIME", true},
		{"SYSTEM_CALLS", true},
		{"UNSAFE_POINTER", true},
		{"OPERATING_SYSTEM", true},
		{"READ_SYSTEM_STATE", true},
		{"MODIFY_SYSTEM_STATE", true},
		// Still outside the closed ten: dropped loudly.
		{"CGO", false},
		{"UNANALYZED", false},
		{"files", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := Valid(tt.name); got != tt.want {
			t.Errorf("Valid(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestRuntimePower(t *testing.T) {
	for _, n := range []Name{Reflect, Runtime, SystemCalls, UnsafePointer, OperatingSystem, ReadSystemState, ModifySystemState} {
		if !RuntimePower(n) {
			t.Errorf("RuntimePower(%q) = false, want true", n)
		}
	}
	for _, n := range []Name{Files, Network, Exec} {
		if RuntimePower(n) {
			t.Errorf("RuntimePower(%q) = true, want false (site-mapped cap)", n)
		}
	}
	if RuntimePower(Name("CGO")) {
		t.Error("RuntimePower(CGO) = true, want false (not in the closed ten)")
	}
}

func TestFamiliesFor(t *testing.T) {
	if got, want := FamiliesFor(Files), []string{"database", "embed", "file-read", "file-write"}; !reflect.DeepEqual(got, want) {
		t.Errorf("FamiliesFor(FILES) = %v, want %v", got, want)
	}
	if got, want := FamiliesFor(Network), []string{"listener", "outbound"}; !reflect.DeepEqual(got, want) {
		t.Errorf("FamiliesFor(NETWORK) = %v, want %v", got, want)
	}
	if got, want := FamiliesFor(Exec), []string{"ipc-subprocess"}; !reflect.DeepEqual(got, want) {
		t.Errorf("FamiliesFor(EXEC) = %v, want %v", got, want)
	}
}

func TestSets(t *testing.T) {
	edges := []Edge{
		{PackageDir: "github.com/larsartmann/io-surface/internal/extract", CapabilityName: "FILES", Depth: "DIRECT"},
		{PackageDir: "github.com/larsartmann/io-surface/internal/extract", CapabilityName: "NETWORK", Depth: "DIRECT"},
		{PackageDir: "github.com/larsartmann/io-surface/internal/extract", CapabilityName: "EXEC", Depth: "TRANSITIVE"},
		{PackageDir: "github.com/larsartmann/io-surface/internal/check", CapabilityName: "FILES", Depth: "TRANSITIVE"},
		{PackageDir: "github.com/larsartmann/io-surface/internal/check", CapabilityName: "REFLECT", Depth: "TRANSITIVE"},
		{PackageDir: "cuelang.org/go/cue", CapabilityName: "FILES", Depth: "DIRECT"},
		{PackageDir: "github.com/larsartmann/io-surface/internal/render", CapabilityName: "REFLECT", Depth: "TRANSITIVE"},
	}
	got := Sets("github.com/larsartmann/io-surface", edges)

	// v1.3: REFLECT stores, so all three first-party packages appear.
	if len(got.Sets) != 3 {
		t.Fatalf("Sets len = %d (%v), want 3 first-party packages", len(got.Sets), got.Sets)
	}
	extractSet := got.Sets[1]
	if extractSet.Pkg != "internal/extract" {
		t.Errorf("second set pkg = %q, want internal/extract", extractSet.Pkg)
	}
	if want := []Name{Exec, Files, Network}; !reflect.DeepEqual(extractSet.Deep, want) {
		t.Errorf("extract deep = %v, want %v", extractSet.Deep, want)
	}
	if want := []Name{Files, Network}; !reflect.DeepEqual(extractSet.Direct, want) {
		t.Errorf("extract direct = %v, want %v", extractSet.Direct, want)
	}
	checkSet := got.Sets[0]
	if checkSet.Pkg != "internal/check" || !reflect.DeepEqual(checkSet.Deep, []Name{Files, Reflect}) {
		t.Errorf("check set = %+v, want pkg internal/check deep [FILES REFLECT] (runtime-power stores since v1.3)", checkSet)
	}
	if len(checkSet.Direct) != 0 {
		t.Errorf("check direct = %v, want empty", checkSet.Direct)
	}
	renderSet := got.Sets[2]
	if renderSet.Pkg != "internal/render" || !reflect.DeepEqual(renderSet.Deep, []Name{Reflect}) {
		t.Errorf("render set = %+v, want pkg internal/render deep [REFLECT]", renderSet)
	}
	if got.Dropped["REFLECT"] != 0 {
		t.Errorf("dropped REFLECT = %d, want 0 (stored since v1.3)", got.Dropped["REFLECT"])
	}
	if got.Dropped["FILES"] != 0 {
		t.Errorf("dropped FILES = %d, want 0", got.Dropped["FILES"])
	}
}

func TestSetsEmptyResultOmitted(t *testing.T) {
	got := Sets("github.com/larsartmann/io-surface", []Edge{
		{PackageDir: "github.com/larsartmann/io-surface/internal/render", CapabilityName: "UNANALYZED", Depth: "TRANSITIVE"},
	})
	if len(got.Sets) != 0 {
		t.Errorf("package with only unmapped caps must be omitted, got %v", got.Sets)
	}
	if len(got.Dropped) != 1 {
		t.Errorf("dropped = %v, want one entry", got.Dropped)
	}
}
