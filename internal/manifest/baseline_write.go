package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrBaseline is the sentinel for baseline.cue update failures (missing file,
// unparseable list shape, duplicate id).
var ErrBaseline = errors.New("baseline update failed")

// AppendBaseline adds one entry to baseline.cue by editing the existing list
// in place — a surgical text insertion that preserves the file's comments and
// formatting, because baseline.cue is a judgment file humans own. It is the
// machine's ONLY sanctioned write into a judgment file, and only for exactly
// this operation: grandfathering one undeclared site with a note and a date.
//
// The date must parse as YYYY-MM-DD (the same shape check's expiry logic
// consumes); an empty date defaults to today.
func AppendBaseline(dir, id, note, date string) error {
	if id == "" {
		return fmt.Errorf("%w: id is empty", ErrBaseline)
	}
	if note == "" {
		return fmt.Errorf("%w: note is empty (a baseline entry without a reason is how the baseline stops shrinking)", ErrBaseline)
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("%w: date %q is not YYYY-MM-DD: %v", ErrBaseline, date, err)
	}

	path := filepath.Join(dir, "baseline.cue")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%w: %s (run `io-surface init` to scaffold one)", ErrNotInitialized, dir)
	}

	text := string(raw)
	if dup := entryLine(text, id); dup >= 0 {
		return fmt.Errorf("%w: %s is already baselined at line %d — extend the existing note instead of stacking entries", ErrBaseline, id, dup+1)
	}

	closer := strings.Index(text, "]\n")
	if closer < 0 {
		closer = strings.LastIndex(text, "]")
	}
	if closer < 0 {
		return fmt.Errorf("%w: no baseline list found in %s — the scaffolded file always carries one", ErrBaseline, path)
	}

	entry := fmt.Sprintf("\t{\n\t\tid:   %q\n\t\tnote: %q\n\t\tdate: %q\n\t},\n", id, note, date)
	updated := text[:closer] + entry + text[closer:]
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("%w: writing %s: %v", ErrBaseline, path, err)
	}

	return nil
}

// entryLine returns the 0-based line of an existing `id: "<id>"` occurrence,
// or -1. It guards the one-entry-per-site invariant by scanning the source
// text (the committed file, not a loaded copy) so a caller cannot stack a
// duplicate that a later Load would silently carry.
func entryLine(text, id string) int {
	needle := fmt.Sprintf("id:   %q", id)
	for i, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needle) {
			return i
		}
	}

	return -1
}
