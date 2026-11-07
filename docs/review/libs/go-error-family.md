# go-error-family — adoption & utilization review

> **Status: IN GRAPH, not imported** — indirect via `go-finding` (pulled by `internal/emit`), `go.mod` line `v0.12.0 // indirect`
> **Verdict: correctly NOT adopted. The protocol is already compiled into the binary at zero dependency cost, but no io-surface error path benefits from family classification.**
> Latest version: **v0.12.0** (2026-10-14; proxy confirms; master 6 commits ahead) · io-surface carries v0.12.0 indirect
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What the library offers

A structured error protocol for Go: **Family** classification (Rejection,
Conflict, Transient, Corruption, Infrastructure, Orchestration) with
`Classify(err)`, `IsRetryable(err)`, `ExitCode(err)` (sysexits-style, e.g. 75
EX_TEMPFAIL), and What/Why/Fix/WayOut structured user-facing errors via six
small interfaces (`Coded`, `Classified`, `Contextual`, `Retryable`,
`ExitCoder`, `HTTPStatuser`). Root module has zero third-party deps. Submodules:
`bridge` (samber/oops integration), `diagnose` (+ git/postgres rules), and
`agent` (experimental root-cause/fix-suggestion analyzer over diagnostic
context).

## io-surface today (evidence)

- Error model is deliberately tiny and fully documented: package sentinels in per-package `errors.go` (`manifest.ErrNotInitialized/ErrEmpty/ErrConstraints`, `scaffold.ErrExists`, `manifest.ErrBaseline`), errors wrap context with `%w`, exit codes are the pinned 0/1/2 gate contract (`cmd/io-surface/main.go:39-51`).
- No error in the system has a *behavioral* consumer: nothing retries, nothing maps errors to HTTP, nothing chooses messaging tone from classification. The only sentinel with switch-like behavior is cmdguard usage errors (rendered by cmdguard) and `atomicwrite.ErrConcurrentModification` (remapped once, `internal/manifest/baseline_write.go:63`).
- The house error style (what happened / why / how to fix) is already the convention everywhere (AGENTS.md Error Handling; flag help strings carry fix hints).

## Why not adopt — and when that flips

| Capability | Fit for io-surface | Reason |
|-----------|--------------------|--------|
| `Classify` / `IsRetryable` | ⚪ N/A | hermetic one-shot tool: no retries, no transient failures by design (determinism is the product) |
| `ExitCode(err)` | 🔴 would be harmful | io-surface's exit contract is 0 = clean / 1 = gate findings / 2 = usage or internal — pinned by CI and README; family exit codes (e.g. 75) would break every consumer |
| Structured What/Why/Fix errors | ⚪ N/A | error volume is small and already written in that style; the win would be marginal polish, at the cost of a new public error shape |
| `agent` submodule (fix suggestions) | ⚪ N/A | targets diagnostic-rule-driven RCA of runtime failures; io-surface findings already carry their own fix hints (`unused-baseline`, `location-drift`, `--enforce` notes) |
| `bridge` / `diagnose` | ⚪ N/A | no oops enrichment, no git/postgres failure domains |

The one honest nuance: because `go-finding` already depends on it, adopting the
interfaces later adds **zero** new modules to the graph. That makes the
trigger purely behavioral: if io-surface ever grows a subcommand whose
failures are retryable or environment-classified (e.g. a networked mode —
which ROADMAP explicitly does not plan), family classification becomes the
right tool. Until then it would be protocol ceremony over three sentinels.

## Recommendation

**Do not adopt.** Document the trigger above; re-review only if a
network/exec-facing mode is ever added.

## How to verify

```bash
go mod why -m github.com/larsartmann/go-error-family   # internal/emit → go-finding → go-error-family
rg -n 'errorfamily' --glob '*.go' .                     # no hits (not imported)
go list -m github.com/larsartmann/go-error-family@latest
```
