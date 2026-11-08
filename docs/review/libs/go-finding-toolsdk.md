# go-finding/toolsdk — adoption & utilization review

> **Status: USED — direct dependency** (`github.com/larsartmann/go-finding/toolsdk v1.16.0`)
> **Verdict: the entire integration surface is two Go files, and the SDK contract is used exactly as designed — spec self-registration, no BuildFlow imports, module fan-out riding the converter's WorkDir agreement.**
> **Adoption score: 100/100 (high, scored 2026-11-08)** — 6 of 6 applicable capabilities fully leveraged, 0 partial
> Latest version: **v1.16.0** (2026-10-11, proxy confirms `@latest`; sibling train to go-finding v1.15.1) · io-surface pins v1.16.0 (plan M2)
> Reviewed: 2026-11-08 · io-surface `docs/review/libs/` series (added by remainder plan N6 — closes the #122 gap with the go-finding report)

## What the library offers

The self-registration plugin contract for BuildFlow providers. A tool repo
calls `toolsdk.Register(Spec{...})` at package init; BuildFlow discovers the
spec via a blank import plus `ToolsFromSDK`/`toolsdk.All()` and converts it to
its internal `domain.Tool` — one-way, in BuildFlow's
`tools/providers/sdk_registry.go::ToolFromSpec`. The boundary is deliberate:
plain strings, go-finding types, and function values only, so a tool NEVER
imports BuildFlow itself. `Spec` carries Name, Description, Trigger,
DependsOn, Inputs, ModuleFanOut, Options, SwitchCases, Detect, Repair;
`Register` validates eagerly (v1.16.0 added Trigger validation — a typo'd
`Language: "golang"` used to silently never match; now it panics at startup —
plus typed option accessors `StringFromContext`/`IntFromContext`/`BoolFromContext`
with tool-declared fallbacks).

## io-surface today (evidence)

The whole integration is `provider/provider.go` (one file) plus the blank
import in BuildFlow's `tools/providers/sdk_imports.go` (the other). Four
symbols total (`rg 'toolsdk\.' provider/`):

| Symbol                  | Where                        | What it buys                                                                                              |
| ----------------------- | ---------------------------- | --------------------------------------------------------------------------------------------------------- |
| `toolsdk.Register(Spec{...})` | `provider/provider.go:47` | self-registration at init — zero BuildFlow-side per-tool code beyond the blank import                      |
| `toolsdk.OnGoModule()`  | `provider/provider.go:53`    | trigger: exactly once per Go module, the fleet-sweep granularity                                            |
| `toolsdk.Spec` fields   | `provider/provider.go:47-77` | `ModuleFanOut: true` (per-module gating in workspaces), `Inputs: ["**/*.go", "docs/architecture/io-surface/*.cue"]` (cache keys), `Detect` as a `finding.NamedDetectorFunc`, `Repair: nil` (gate never mutates user code) |
| `toolsdk.All()`         | (BuildFlow side, not ours)   | discovery — listed here because the contract's other half lives in `sdk_registry.go`, which we do not own     |

The consumer contract for `ModuleFanOut` is honored verbatim: the provider's
Detect reads the working directory from `finding.WorkingDirFromContext(ctx)`
and never from the process CWD, and does not self-wrap with its own
WorkDir/progress adapters (spec.go:38-44 forbids exactly that; BuildFlow's
`detectorWithWorkDir` does the wrapping). The two-faces description names both
the gate and the radar so `buildflow list steps`/`explain io-surface` tell the
truth about what the step will do.

Not used, by design: `Options` + typed accessors (the step is flag-free —
every option io-surface honors is a CLI flag on the `io-surface` binary, not a
BuildFlow option value), `DependsOn` (the gate must run on whatever tree state
exists; ordering it after `go-fix` would hide the truth from the gate),
`SwitchCases` (no conditional repair — Repair is nil), `OnGoFiles()` (wrong
granularity: one run per module, not per file).

## Capability inventory

| # | Capability                                      | Status             | Evidence / reason                                                                                       |
| - | ----------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------- |
| 1 | Self-registration (`Register` at init)          | 🟢 Fully leveraged | `provider/provider.go:47`; one-file integration proof; `go vet` + panic-on-invalid keeps the spec honest  |
| 2 | Trigger correctness (`OnGoModule`)              | 🟢 Fully leveraged | v1.16.0's Trigger validation now guards the exact silent-miss failure mode; io-surface's trigger is valid by construction |
| 3 | `ModuleFanOut` + WorkDir agreement              | 🟢 Fully leveraged | `ModuleFanOut: true`; Detect reads the context dir only; verified end-to-end 2026-11-08 (both-fires evidence, TODO #125: gate + radar faces through `buildflow -s io-surface`) |
| 4 | `Inputs` globs as cache keys                    | 🟢 Fully leveraged | both glob sets declared; result cache hits observed (57% → 100% across runs) — the cache keys on exactly these |
| 5 | Detect capability (finding.Detector)            | 🟢 Fully leveraged | `NamedDetectorFunc` returning gate+radar findings merged under one enforcement policy                     |
| 6 | Version currency                                | 🟢 Fully leveraged | v1.16.0 = `@latest` (proxy check 2026-11-08); same train as go-finding v1.15.1                            |
| 7 | Options channel (`WithOption`/typed accessors)  | ⚪ N/A             | flag-free step; CLI flags own configuration — adding Options would split one config surface into two      |
| 8 | `DependsOn` / `SwitchCases` / `Repair`          | ⚪ N/A             | read-only gate by doctrine; conditional-repair topology has no consumer here                              |

Score derivation (skill rubric): N=8 rows, rows 7–8 excluded as N/A → A=6
applicable; fully=6, partial=0 → round(100 × 6/6) = **100 (high; scored
2026-11-08)**. The integration is the library's intended shape at minimum
surface: if a future spec needs Options or SwitchCases, the SDK already
validates and documents them — the extension is additive, not corrective.

## Recommendation

No change. Two watch items, both informational:

1. **`SwitchCases` validation is deferred upstream** (spec.go notes the
   Suffix/rule-ID checks land with the first real consumer) — irrelevant
   while `Repair: nil`, which is a doctrine constraint, not a TODO.
2. **Blank-import half of the loop:** the provider ships only when BuildFlow's
   `sdk_imports.go` blank-imports the published tag — the release protocol's
   "both halves ship together" rule (AGENTS.md). A provider change without a
   release is invisible to the fleet; that bit us once already (the stale
   system binary predating the v0.4.0 pin reported 0 radar findings silently,
   BuildFlow#81 class).

## How to verify

```bash
go list -m github.com/larsartmann/go-finding/toolsdk@latest   # v1.16.0 = pin
rg -o 'toolsdk\.[A-Za-z]+' provider/ --no-heading | sed 's/.*://' | sort | uniq -c   # the 4-symbol usage map above
grep -n "io-surface/provider" /home/lars/projects/BuildFlow/tools/providers/sdk_imports.go   # the blank-import half (v0.4.0 pin)
buildflow list steps | grep io-surface                        # step registered with the two-face description
go test ./provider/                                           # provider package green
```
