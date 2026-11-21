# go-finding run-level propertyBag capability packet (TODO #133) — VERIFIED 2026-11-21

Status: verify-before-filing Gate 5 (already exists?) FAILED to find the
feature — it does NOT exist upstream, so the request is legitimate. go-finding
is the owner's OWN repo, so "filing" = opening an issue on
LarsArtmann/go-finding; the packet below is the issue draft, owner-gated like
#50 (no posting without an explicit order).

## Source-level verification (what exists today)

At go-finding master `bfd32c7` (`v1.15.1-89-gbfd32c7`, 89 commits past the
newest tag v1.15.1; io-surface consumes go-finding v1.15.1 / toolsdk v1.16.0):

- `sarif_types.go:63-66` — the run object carries only tool + results:

  ```go
  type sarifRun struct {
  	Tool    sarifTool     `json:"tool"`
  	Results []sarifResult `json:"results"`
  }
  ```

- `sarif_types.go:96` — `Properties map[string]any` exists ONLY per result
  (`sarifResult`), with the doctrine comment at lines 81-86 documenting the
  map[string]any boundary as "the ONLY place in go-finding where map[string]any
  is used".
- `sarifLog` (lines 56-60) has no top-level properties either.

So `runs[].properties` (SARIF 2.1.0 allows a propertyBag on every object,
§3.10.1) is absent from the model at master.

## The consumer need (io-surface's workaround today)

`internal/emit/emit.go:34-42`: every finding carries the run's enforcement
posture (gate/audit/lint) as a per-result SARIF property
`go-finding/meta/enforcement`, because go-finding has no run propertyBag.
The comment already names the migration trigger: "If go-finding grows a run
propertyBag, migrate there and drop this key in a minor release
(CHANGELOG note)." Run-level is the honest home — the value is per-RUN
(one enforcement posture per invocation), and per-result copies survive
consumer-side filtering only by redundancy: N results × 1 unchanged value.

A run-level bag would also give the tool-level rules array a home
(`runs[].tool.driver.rules` + `run.properties.ruleDescriptions`), retiring
the second per-result workaround (`properties.ruleDescription`,
emit.go:28-32).

## Proposed shape (minimal)

```go
type sarifRun struct {
	Tool       sarifTool      `json:"tool"`
	Results    []sarifResult  `json:"results"`
	Properties map[string]any `json:"properties,omitempty"`
}
```

Plus one setter on the report/export path (e.g. `report.SetRunProperty(k, v)`
or an export-option `WithRunProperties(map[string]any)`) — the exact seam is
go-finding's call; the requirement is only: survives the
`Report → sarifLog` export and, symmetrically, the import round-trip.

## FILING DRAFT (owner gate: do not post without explicit order)

Proposed title:
`run-level propertyBag: carry per-run metadata once instead of per-result`

```markdown
> [!NOTE]
> This filing was drafted by GLM-5.3-flash (via Crush) from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

TL;DR: `sarifRun` has no `properties` field, so per-run metadata (an enforcement posture, a scan root, a tool version tag) must today be copied onto every result. Ask: add `runs[].properties` (SARIF 2.1.0 allows a propertyBag on every object) with an export/import round-trip.

## Problem

`io-surface` (a consumer of go-finding v1.15.1) emits findings whose RUN carries one enforcement posture — `gate`, `audit`, or `lint` — but go-finding's SARIF model has nowhere to put run-level facts:

`// sarif_types.go:63 (master bfd32c7)`

```go
type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}
```

While `sarifResult` already has `Properties map[string]any` (`sarif_types.go:96`). So the posture rides every result as `go-finding/meta/enforcement`, and a second consumer fact (rule descriptions) rides every result as `go-finding/meta/ruleDescription` — N results × 1 unchanged value, and both copies survive consumer-side result filtering only by redundancy.

## Goal

As a consumer, I want to write:

```go
report.SetRunProperty("go-finding/meta/enforcement", "gate")
```

(or the equivalent export option) and see it once under `runs[0].properties` in the SARIF output — and read it back on import.

## Delta

|                    | today                                | desired                             |
| ------------------ | ------------------------------------ | ----------------------------------- |
| per-run metadata   | copied onto every result's `properties` | one `runs[].properties` bag      |
| result filtering   | metadata survives only by redundancy | run bag independent of results      |
| import round-trip  | per-result keys only                 | run bag preserved (`omitempty` when empty — zero diff for existing users) |

Constraint honored either way: `map[string]any` stays confined to the SARIF layer per the doctrine comment at `sarif_types.go:81-86` — the public `Finding.Metadata` remains `map[string]string`; run properties would live on the export/import path like the existing result properties.

💘 Generated with Crush
```

## Follow-through after the feature lands (do NOT start without the owner)

1. go-finding release train (version bump + tag per go-release protocol).
2. Bump io-surface's `go-finding` require, migrate emit: `enforcement` and
   `ruleDescription` move to the run bag; drop the per-result keys.
3. Re-pin lint goldens if the SARIF shape changes; CHANGELOG entry per
   Keep a Changelog; then the provider release so BuildFlow picks it up
   (both faces ship through that loop — docs/release-protocol.md).
