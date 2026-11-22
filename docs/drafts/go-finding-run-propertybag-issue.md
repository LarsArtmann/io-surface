> [!NOTE]
> This filing was drafted by GLM-5.3-flash (via Crush) from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

TL;DR: `sarifRun` has no `properties` field, so per-run metadata (an enforcement posture, a scan root, a tool version tag) must today be copied onto every result. Ask: add `runs[].properties` (SARIF 2.1.0 allows a propertyBag on every object) with an export/import round-trip.

## Problem

`io-surface` (a consumer of go-finding v1.15.1) emits findings whose RUN carries one enforcement posture — `gate`, `audit`, or `lint` — but go-finding's SARIF model has nowhere to put run-level facts:

`// sarif_types.go:63 (master bfd32c7, re-verified 2026-11-22)`

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

|                   | today                                   | desired                                                                   |
| ----------------- | --------------------------------------- | ------------------------------------------------------------------------- |
| per-run metadata  | copied onto every result's `properties` | one `runs[].properties` bag                                               |
| result filtering  | metadata survives only by redundancy    | run bag independent of results                                            |
| import round-trip | per-result keys only                    | run bag preserved (`omitempty` when empty — zero diff for existing users) |

Constraint honored either way: `map[string]any` stays confined to the SARIF layer per the doctrine comment at `sarif_types.go:81-86` — the public `Finding.Metadata` remains `map[string]string`; run properties would live on the export/import path like the existing result properties.

💘 Generated with Crush
