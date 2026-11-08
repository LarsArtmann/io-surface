# go-output — adoption & utilization review

> **Status: IN GRAPH, not imported** — indirect via `cmdguard/v4` (`go.mod` carries 12 lines: root `v0.38.4 // indirect` + 11 submodules)
> **Verdict: correctly NOT adopted. io-surface's output is a byte-pinned contract, and every machine format that matters already has a dedicated path.**
> Latest version: **v0.38.7** (root module, proxy confirms) · io-surface carries v0.38.4 indirect
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What the library offers

"Write your data once, render it anywhere": a `Table`/`Tree`/`Graph` data
model with 16 render formats (markdown, CSV/TSV, JSON serialization, HTML,
TUI, d2, PlantUML, daghtml, …), a type-safe `Format` enum for runtime
`--format` flags, zero-config color, and NOM-style real-time progress
visualization. Root module is dependency-light (`golang.org/x/term` only);
renderers live in isolated submodules.

## io-surface today (evidence)

Four bespoke printers own every human-facing byte, deliberately:

- `printReport` / `printLintReport` (`cmd/io-surface/commands.go:168-180, 541-553`) — aligned two-line findings format.
- `printStats` (commands.go:387-412) — the per-package histogram.
- `printGitHubAnnotations` (commands.go:560-579) — `::error/::warning/::notice` workflow commands.
- Machine formats: `--format json` via `encoding/json`, `--format sarif` via `go-finding` (`internal/emit`), and the doc-embedding path via `internal/render` — all byte-pinned by goldens (`cmd/io-surface/testdata/lint-*.golden`) and parsed by CI.

## Why not adopt

| Capability                                   | Fit    | Reason                                                                                                                      |
| -------------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------- |
| Terminal table rendering (`printStats` etc.) | ⚪ N/A | the text format is a stability contract, pinned by tests and grepped by humans/CI; restyling is churn without a user asking |
| `Format` enum + runtime dispatch             | ⚪ N/A | io-surface already has its own `--format` switch (text/json/sarif/github) — a second enum would be a split brain            |
| markdown/CSV/HTML renderers                  | ⚪ N/A | findings' machine formats are JSON + SARIF (what CI consumes); a markdown/CSV findings export has zero recorded demand      |
| NOM progress visualization                   | ⚪ N/A | runs are sub-second; no long-running workflow to visualize                                                                  |
| Color/themes                                 | ⚪ N/A | gate output is deliberately plain (CI logs, piping); color would break byte-pinned goldens for zero value                   |

The strongest argument _for_ adoption — it is already in the module graph via
cmdguard, so a direct import adds no new module — is exactly why this is
worth writing down: the cost is one direct require line, and the benefit is
still zero until a real rendering need exists. The minimal-deps doctrine
(README markets "the AST walk stays stdlib-only"; four direct deps is a
stated feature) wins.

## Recommendation

**Do not adopt.** Revisit only if pilot users ask for a findings export
format the tool cannot produce today (e.g. an HTML report or CSV of
`check` findings) — then `go-output/markdown` or `/delimited` would be the
honest first stop rather than hand-rolling.

## How to verify

```bash
go mod why -m github.com/larsartmann/go-output      # cmd/io-surface → cmdguard/v4 → go-output
rg -n 'go-output' --glob '*.go' .                    # no hits (not imported)
ls cmd/io-surface/testdata/                          # byte-pinned goldens the printers must keep
go list -m github.com/larsartmann/go-output@latest   # v0.38.7
```
