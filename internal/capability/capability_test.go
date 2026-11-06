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
		{"REFLECT", false},
		{"RUNTIME", false},
		{"files", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := Valid(tt.name); got != tt.want {
			t.Errorf("Valid(%q) = %v, want %v", tt.name, got, tt.want)
		}
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

func TestFamiliesUnion(t *testing.T) {
	got, err := Families([]Name{Network, Exec, Files})
	if err != nil {
		t.Fatalf("Families: %v", err)
	}
	want := []string{"database", "embed", "file-read", "file-write", "ipc-subprocess", "listener", "outbound"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Families union = %v, want %v", got, want)
	}
	if _, err := Families([]Name{"CGO"}); err == nil {
		t.Error("Families(CG0) should error on a name outside the closed enum")
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

	if len(got.Sets) != 2 {
		t.Fatalf("Sets len = %d (%v), want 2 first-party packages", len(got.Sets), got.Sets)
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
	if checkSet.Pkg != "internal/check" || !reflect.DeepEqual(checkSet.Deep, []Name{Files}) {
		t.Errorf("check set = %+v, want pkg internal/check deep [FILES] (REFLECT dropped)", checkSet)
	}
	if len(checkSet.Direct) != 0 {
		t.Errorf("check direct = %v, want empty", checkSet.Direct)
	}
	if got.Dropped["REFLECT"] != 2 {
		t.Errorf("dropped REFLECT = %d, want 2", got.Dropped["REFLECT"])
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
