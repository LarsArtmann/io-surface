> [!NOTE]
> This filing was drafted by GLM-5.3-flash (via Crush) from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

TL;DR: the purge collector treats ANY line carrying a non-empty `~~…~~` pair as a struck row — it never reads the row's status cell, so stray double-tilde fragments on OPEN rows get collected and purged as done work. Ask: a semantic guard that refuses (or warns on) struck lines whose row is not DONE.

## Problem

`collectStruck` collects "every line carrying a ~~…~~ wrap with non-empty task text", and its extractor looks only at the first `~~` pair on the line:

`// internal/docspurge/docspurge_collect.go:14 + :33 + :53 (master c81685b, 2026-11-22)`

```go
func collectStruck(content string) []struckRow {
	// ...
	row, ok := splitStrike(line, i+1)   // no status-cell input anywhere
```

```go
func splitStrike(line string, lineNo int) (struckRow, bool) {
	first := strings.Index(line, "~~")
	// ...
	if strings.TrimSpace(task) == "" && verdict == "" {
		return struckRow{}, false
	}
```

The row's status column (`🔴 TODO` / `🟢 DONE`) is never consulted. A strike that covers only part of a task cell, or stray tilde noise anywhere in a row, still collects.

## The incident

io-surface's 2026-11-21 purge: a concurrent session had left `~~`-fragments (strike-adjacent edit noise like a partially struck row number) on 6 OPEN TODO rows. `docs-gates purge . --apply` collected 165 rows — the 6 open ones among them — moved them to the CHANGELOG as DONE bullets, and deleted them from TODO_LIST. Recovered from git (`b6e88ec`), but only because the loss was noticed; the collector had no signal that anything unusual happened.

## Ask

A semantic guard in the collector: when the struck line's row status cell is present and not DONE — or when the `~~` pair does not start at the task-cell boundary — refuse to collect (strict mode) or collect with a loud per-row warning (advisory mode, mirroring how absorb drift stays advisory). The status cell is already a stable table shape the fleet's other gates parse, so the collector has the information it needs one cell to the right.

💘 Generated with Crush
