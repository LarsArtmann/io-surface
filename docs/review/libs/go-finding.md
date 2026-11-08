# go-finding — adoption & utilization review

> **Status: USED — direct dependency** (`github.com/larsartmann/go-finding v1.15.1` + `toolsdk v1.16.0`)
> **Verdict: leveraged to the max on the emit/detect faces this tool needs; the unused pipeline/LSP/analysis modules are owned by BuildFlow or out of scope by hermeticity.**
> **Adoption score: 100/100 (high, scored 2026-11-08)** — 6 of 6 applicable capabilities fully leveraged, 0 partial
> Latest version: **v1.15.1** core / **v1.16.0** toolsdk (same 2026-10-11/12 release train; proxy confirms both are `@latest` — the "skew" is per-sub-module SemVer, not lag) · io-surface pins exactly those (plan M2)
> Reviewed: 2026-11-08 · io-surface `docs/review/libs/` series (added by remainder plan N5 — this and the toolsdk report close the #122 gap)

## What the library offers

A unified data model and pipeline for Go static-analysis findings: one
`Finding` type (stable ID `tool:rule:file:line:col`, rule, tool, message,
severity, position, category, tags, fix strategy, suggestion code), `Report`
with computed summaries, SARIF 2.1.0 export **and import**, LSP diagnostic
conversion, text/markdown rendering, report merge/dedupe/cross-tool
correlation/diff, a Builder API with validation, plus three sub-modules:
`pipeline` (detect → triage → fix → verify loop with per-finding fix outcomes
and scoped rollback), `analysis` (go/analysis integration), and `toolsdk`
(BuildFlow tool-spec contract — reviewed separately in
`docs/review/libs/go-finding-toolsdk.md`). Unreleased on master: `Relativize`
(issue #47) rewrites absolute-path positions to root-relative and regenerates
IDs so checkout-independent IDs survive the CLI vs SDK path-form difference.

## io-surface today (evidence)

io-surface rides go-finding on two faces, and the mapping contract is written
down in `internal/emit/emit.go` (the "lossless for what matters" doc comment):

| Face          | Call sites                                                        | What rides the library                                                                                                                    |
| ------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `emit` (SARIF) | `internal/emit/emit.go:50-58` (`NewFinding`, `Metadata`), `:84` (`NewReportFromFindings`), `:89` (`ToSARIF`) | rule IDs pass through verbatim (`io-surface/undeclared-site` becomes the SARIF ruleId unchanged); severities map 1:1 (error/warning/info — both vocabularies have exactly these three in io-surface's contract); `path:line` locations become `Position` with line-0 = unset for manifest-level findings; confidence 1.0 everywhere (deterministic gate — weak-tier uncertainty is expressed by the `weak-site` RULE, never by a made-up confidence number) |
| result metadata | `emit.go:30-41` (`ruleDescription` T15 + `enforcement` D4/#93)    | both ride `Finding.Metadata` → SARIF result `properties` (`properties.ruleDescription`, `go-finding/meta/enforcement`). Result-level is DELIBERATE over run-level: go-finding has no run propertyBag today (the #133 upstream request), and result properties survive consumer-side filtering that would drop a run bag |
| provider (BuildFlow) | `provider/provider.go:50` (`NamedDetectorFunc`), `:54-56` (`WorkingDirFromContext` fallback to `"."`) | the detector is a `finding.Detector` on the shared working-dir context key — BuildFlow's `runner.WithWorkingDir` and the SDK read the SAME value (one propagation path, documented on both sides) |

Not used, by design (evidence for the score's N/A rows): `SARIF import` (we
only emit), `merge/dedupe/correlate/diff` (one deterministic tool, no
cross-tool baseline), `Builder` API (`NewFinding` already validates the
fields we set; the builder's richer taxonomy — category, tags, fix strategy —
would be ceremonial for gate output), `LSP` conversion (no editor client),
`pipeline`/`analysis` modules (BuildFlow owns the orchestration loop; the
hermetic stdlib-only extractor is precisely NOT a go/analysis pass — that
would import `golang.org/x/tools` into the gate core).

## Capability inventory

| # | Capability                                             | Status             | Evidence / reason                                                                                                                                                                    |
| - | ------------------------------------------------------ | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | Core model: `NewFinding`/`Report`/`ToolInfo`/`Position` | 🟢 Fully leveraged | emit + provider, 18 distinct symbols, mapping contract documented in the package comment (lossless for rule/severity/position)                                                        |
| 2 | SARIF 2.1.0 export                                     | 🟢 Fully leveraged | `emit.SARIF` → `report.ToSARIF()`; CI uploads the lint face as artifact; goldens byte-pin the output (`cmd/io-surface/testdata/lint-sarif.golden`)                                     |
| 3 | `Metadata` → SARIF result properties                   | 🟢 Fully leveraged | T15 rule descriptions + D4 enforcement posture (v0.5.0); the propertyBag migration path is pre-negotiated in the emit doc (drop the key in a minor release if go-finding grows a run bag) |
| 4 | Detector contract + working-dir context                | 🟢 Fully leveraged | provider's `NamedDetectorFunc` on the shared `WithWorkingDir` key; harness-verified 2026-11-08 (standalone `Detect` on a temp tree returned the expected 3 findings)                    |
| 5 | Severity vocabulary                                    | 🟢 Fully leveraged | 1:1 map in `severityOf` — io-surface's error/warning/info contract covers go-finding's levels it uses; `critical` would be a contract extension, not an underuse                        |
| 6 | Version currency                                       | 🟢 Fully leveraged | pins = `@latest` on both modules (verified 2026-11-08 via `go list -m …@latest`); same release train as toolsdk v1.16.0; `Relativize` is unreleased so nothing is being missed          |
| 7 | SARIF import / LSP / merge-correlate-diff / Builder / pipeline / analysis | ⚪ N/A            | see the by-design list above — each is either BuildFlow's job, an editor feature io-surface never ships, or violates the hermeticity doctrine                                            |

Score derivation (skill rubric): N=7 rows, row 7 excluded as N/A → A=6
applicable; fully=6, partial=0 → round(100 × 6/6) = **100 (high; scored
2026-11-08)**. First 100 in the series — and it is earned, not generous: the
library's single known limitation io-surface hits (no run-level propertyBag)
is already worked around at the right layer with a migration path on record.

## Recommendation

Keep on this exact train. Two standing items, both already tracked:

1. **#133 (upstream request):** file the run-level SARIF `propertyBag`
   request against go-finding (verify-before-filing applies — read
   `sarif_export.go` first to confirm the bag truly does not exist), then
   migrate `enforcement` per the emit doc's note.
2. **`Relativize` when released:** emit's contract already produces
   repo-relative paths (extracted IDs/locations are root-relative by the
   manifest contract), so nothing to do today; if the provider lane ever
   starts emitting absolute-path positions (a WorkingDir contract change),
   `Relativize` is the designated fix — no action until then.

## How to verify

```bash
go list -m github.com/larsartmann/go-finding@latest        # v1.15.1 = pin
go list -m github.com/larsartmann/go-finding/toolsdk@latest # v1.16.0 = pin
rg -o 'finding\.[A-Za-z]+' internal/ provider/ cmd/ --no-heading | sed 's/.*://' | sort | uniq -c | sort -rn   # the 18-symbol usage map above
go test ./internal/emit/ ./cmd/io-surface/ -run 'TestSARIF|TestToFindings|TestGolden'   # SARIF mapping + goldens green
grep -n 'enforcement' internal/emit/emit.go                # D4 posture metadata (this report's row 3)
```
