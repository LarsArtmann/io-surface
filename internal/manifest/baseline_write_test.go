package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baselineFixture = `// baseline.cue — grandfathered undeclared sites (the adoption escape hatch).
//
// The baseline only shrinks: every entry needs a note and a date.
package io_surface

baseline: [
	{
		id:   "old.Site"
		note: "legacy"
		date: "2026-01-01"
	},
]
`

func TestAppendBaseline(t *testing.T) {
	writeB := func(t *testing.T, text string) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "baseline.cue"), []byte(text), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		return dir
	}

	t.Run("appends preserving comments", func(t *testing.T) {
		dir := writeB(t, baselineFixture)
		if err := AppendBaseline(dir, "new.Site", "tracked in T1", "2026-06-01"); err != nil {
			t.Fatalf("append: %v", err)
		}
		raw, rErr := os.ReadFile(filepath.Join(dir, "baseline.cue"))
		if rErr != nil {
			t.Fatalf("read baseline: %v", rErr)
		}
		text := string(raw)
		if !strings.Contains(text, "// baseline.cue — grandfathered") {
			t.Error("header comment lost")
		}
		for _, want := range []string{"new.Site", "tracked in T1", "2026-06-01", `"old.Site"`} {
			if !strings.Contains(text, want) {
				t.Errorf("baseline.cue lost %q", want)
			}
		}
	})

	t.Run("empty date defaults to today", func(t *testing.T) {
		dir := writeB(t, baselineFixture)
		if err := AppendBaseline(dir, "new.Site", "why", ""); err != nil {
			t.Fatalf("append: %v", err)
		}
		raw, rErr := os.ReadFile(filepath.Join(dir, "baseline.cue"))
		if rErr != nil {
			t.Fatalf("read baseline: %v", rErr)
		}
		if !strings.Contains(string(raw), "date:") {
			t.Error("no date written")
		}
	})

	t.Run("duplicate id is refused", func(t *testing.T) {
		dir := writeB(t, baselineFixture)
		err := AppendBaseline(dir, "old.Site", "again", "")
		if !errors.Is(err, ErrBaseline) || !strings.Contains(err.Error(), "already baselined") {
			t.Errorf("err = %v, want ErrBaseline naming the duplicate", err)
		}
	})

	t.Run("empty note is refused", func(t *testing.T) {
		dir := writeB(t, baselineFixture)
		err := AppendBaseline(dir, "new.Site", "", "")
		if !errors.Is(err, ErrBaseline) || !strings.Contains(err.Error(), "note is empty") {
			t.Errorf("err = %v, want ErrBaseline naming the note", err)
		}
	})

	t.Run("bad date shape is refused", func(t *testing.T) {
		dir := writeB(t, baselineFixture)
		err := AppendBaseline(dir, "new.Site", "why", "01-06-2026")
		if !errors.Is(err, ErrBaseline) || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Errorf("err = %v, want ErrBaseline naming the date shape", err)
		}
	})

	t.Run("missing file is ErrNotInitialized-shaped", func(t *testing.T) {
		dir := t.TempDir()
		err := AppendBaseline(dir, "new.Site", "why", "")
		if !errors.Is(err, ErrNotInitialized) {
			t.Errorf("err = %v, want ErrNotInitialized", err)
		}
	})
}
