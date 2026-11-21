package main

// The check face's byte-pinned text report (TODO #168): printReport's prose
// — header, severity/rule/ID table, message lines, the stats summary — had
// no golden, so a presentation regression shipped silently while only the
// lint face was byte-pinned. The fixture is the shared miniature tree with
// its scaffolded manifest plus one deliberately undeclared site, so the
// golden pins the findings table AND the summary. Intentional printReport
// changes re-run with -update.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGolden_CheckOutput(t *testing.T) {
	root := t.TempDir()
	scaffoldFixture(t, root)
	late := filepath.Join(root, "cmd", "late")
	if err := os.MkdirAll(late, 0o755); err != nil {
		t.Fatalf("mkdir late: %v", err)
	}
	site := "package late\n\nimport \"os\"\n\nfunc Late() ([]byte, error) {\n\treturn os.ReadFile(\"late.bin\")\n}\n"
	if err := os.WriteFile(filepath.Join(late, "late.go"), []byte(site), 0o644); err != nil {
		t.Fatalf("write late site: %v", err)
	}

	out, err := runApp(t, "check", "--root", root)
	if !errors.Is(err, errFindings) {
		t.Fatalf("check: %v (an undeclared site must fail the gate)\n%s", err, out)
	}

	goldenPath := "testdata/check-text.golden"
	if *updateGoldens {
		if err := os.WriteFile(goldenPath, []byte(out), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (first run: go test ./cmd/io-surface -run TestGolden_CheckOutput -update)", err)
	}
	if out != string(want) {
		t.Errorf("check text output drifted from the golden (re-run `go test ./cmd/io-surface -run TestGolden_CheckOutput -update` after an intentional change):\n--- want ---\n%s\n--- got ---\n%s", want, out)
	}
}
