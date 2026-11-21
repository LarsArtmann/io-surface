# Domain Language — io-surface

The ubiquitous language for this repo. One term, one meaning, one home.
When code and this file disagree, code wins and this file gets fixed.
Every definition names where it lives so the next reader can verify
instead of trust (`grep` + file:line).

## The core nouns

| Term                  | Definition                                                                                                                                                                                                                     | Lives in                                    |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------- |
| **Site**              | One production code location that touches I/O — the atomic unit of everything. Identified by an **ID** (see conventions). Has exactly one **family**.                                                                           | `internal/extract` (extraction), `schema.cue` |
| **Family**            | A site's dominant I/O role. Closed enum of 10: `file-read`, `file-write`, `embed`, `database`, `listener`, `outbound`, `stream-copy`, `entropy`, `ipc-subprocess`, `env`. Adding one is a schema version bump (taxonomy freeze). | `internal/manifest/manifest.go:26`            |
| **Discipline**        | A cross-cutting policy tag a claim puts on a site. Closed enum of 7: `atomic-write`, `single-writer`, `buffered-write`, `wal-pragmas`, `bounded`, `pooled`, `no-domain-labels`.                                                 | `internal/manifest/manifest.go:32`            |
| **Manifest**          | The CUE package at `docs/architecture/io-surface/` — the declaration the gate checks code against. Four files, one job each (below).                                                                                            | `internal/manifest`, `internal/scaffold`      |
| **Facts**             | Machine-extracted site data (call tokens, location, family). Regenerate freely; mechanical commits only. Never hand-edit locations.                                                                                             | `schema.cue` ("the load-bearing wall")        |
| **Claims**            | Hand-written judgment attached to a site: `resource`, `trigger`, `discipline`, `exception`, `rationale`, plus v1.2 `criticality`/`durability`. Never overwritten by tooling.                                                    | `schema.cue`, `internal/manifest`             |
| **Baseline**          | Grandfathered undeclared sites: `{id, note, date}` (+ optional `owner`/`expires`). Only shrinks — an entry suppressing nothing warns (`unused-baseline`), an expired entry errors (`baseline-expired`).                          | `internal/check/check.go` (rule consts)       |
| **Capabilities**      | Per-package reachability from an external SSA engine (Capslock), folded in by `absorb`. Closed-enum names `FILES`/`NETWORK`/`EXEC`, per package as `direct` (own code) and `deep` (transitive). Advisory, never gate-failing.    | `schema.cue:108`, `internal/capability`       |
| **Gate**              | The bidirectional check: every declared site still exists, every extracted site is declared. Exit 1 on error findings. Distinct from a linter — it enforces a documented contract, not style.                                   | `internal/check`                              |
| **Radar (lint face)** | Detector findings over ANY Go tree without a manifest. Every rule ships at `info`; exit 0 unless `--enforce`. The gate wants a manifest; the bugs don't.                                                                        | `internal/lint`                               |
| **Finding**           | One rule violation with rule ID, severity, location. The stable contract: rule IDs become SARIF rule IDs and suppression keys — renaming is breaking.                                                                         | `internal/check/check.go:40`, `internal/emit`  |

## Extraction tiers

| Tier         | Meaning                                                                            | Can fail a build? |
| ------------ | ---------------------------------------------------------------------------------- | ----------------- |
| **resolved** | Import-qualified calls through resolved imports (`os.ReadFile`, `http.Get`).        | Yes — the only tier with error teeth. |
| **token**    | Method-name heuristics on unresolvable receivers (`.Do`, `.Listen`, `DialContext`). | No — advisory by construction (`--strict-weak` opts a repo in). |
| **seam**     | A same-package wrapper (`atomicWriteFile`) owning the raw call; attribution lands on the wrapper, not its callers. Manifest `via` values override the built-in table. | n/a — attribution, not confidence. |

**Hermetic**: the extractor is stdlib-only (`go/parser` + `ast.Inspect`, no
`go/packages`, no module downloads), so it runs in a Nix sandbox with
`GOPROXY=off`. Type-fact resolution is out of scope BY DESIGN — such sites are
declared `facts.extractable: false` instead of guessed.

## The verbs

| Command         | Domain meaning                                                                                       |
| --------------- | ---------------------------------------------------------------------------------------------------- |
| `init`          | Snapshot the tree into a manifest — green by construction. `--force` refreshes `facts.cue` ONLY.      |
| `check`         | Run the bidirectional gate.                                                                           |
| `extract`       | Dump raw extraction as JSON (debugging/counts).                                                       |
| `render`        | Render the manifest as inventory markdown; `--doc` embeds into a hand-written doc behind BEGIN/END markers; `--lint` appends the radar appendix. |
| `lint`          | Run the detectors (radar) without a manifest.                                                         |
| `absorb`        | Fold a pinned Capslock JSON into `capabilities.cue` — that file ONLY.                                 |
| `baseline add`  | The ONE sanctioned machine write into a judgment file: append a grandfathering entry surgically.       |

## Rule vocabulary (the findings)

Gate rules (`internal/check/check.go:40-56`, 15 IDs): `undeclared-site`,
`stale-claim`, `id-drift`, `id-form`, `location-drift`, `missing-location`,
`extraction-blind`, `weak-site`, `ambiguous-match`, `unused-baseline`,
`baseline-expired`, `scope-hint`, `shadow-capability`, `manifest-validation`,
`claim-conflict`. Only `error` severity fails the gate; severity is policy
(`enforcement: "gate" | "audit"` in facts.cue, `--enforce` overrides one run).

Detector rules (`internal/extract/observations.go:35-44`, 10 IDs):
`err-dropped`, `closer-missing`, `ctx-dropped`, `defer-in-loop`,
`client-no-timeout`, `stat-then-open`, `io-in-loop`, `write-non-atomic`,
`tiny-buffer`, `ignore-invalid`. Promotion above `info` is evidence-gated:
fleet false-positive measurement (FP≈0) or it does not happen
(`docs/planning/2026-11-07_promotion-policy.md`).

Suppression: `//io-surface:ignore <rule> <reason>` silences one finding at one
site, reason mandatory; a malformed directive emits `ignore-invalid` instead of
suppressing. File-level ignores do not exist — that bluntness is a baseline entry.

## ID conventions (`schema.cue` header, fixture-verified)

```text
pkg.Func            internal/dns.estimatedLines
pkg.Type.Method     internal/server.AllowlistManager.Save
pkg.Var             internal/server.openAPISpec        (go:embed)
pkg.Func#aspect     internal/config.Load#file          (one symbol, two families)
area.topic          logging.sink-topology              (no single symbol)
```

Matching is receiver-insensitive (`pkg.Type.Method` ≡ `pkg.Method`,
`internal/match`), but exact IDs pair BEFORE canonical ones, so two types with
a same-named method are never silently merged (`internal/match/collision_test.go`).
IDs are repo/root-relative.

## Cross-cutting terms

- **Taxonomy freeze** — families, disciplines, capability names are closed
  enums; growing one is a schema version bump (`io-surface/v1` → `v1.1` →
  `v1.2`), never a casual edit. Unknown contract versions refuse to load
  (`ErrUnsupportedVersion`).
- **Only shrinks** — the baseline never grows silently; staleness is a warning,
  expiry is an error.
- **Green by construction** — `init` snapshots reality, so a fresh manifest
  cannot fail; the gate only fails on I/O added AFTER the snapshot.
- **Evidence-gated promotion** — detector severity moves only on measured fleet
  FP data, never on intuition; the fleetscan corpus
  (`docs/evidence/fleetscan-*.json`) is the measurement.
- **Canonical key** — receiver-insensitive matching means one baseline entry
  may hide same-base-name sites; `ambiguous-match` names the shape.
- **Two faces, one tool** — the gate (manifest required, error teeth) and the
  radar (manifestless, advisory); `enforcement` posture travels in every SARIF
  result (`go-finding/meta/enforcement`).

## What the words are NOT (anti-language)

- **"Surface" is not API surface.** It is the I/O contact patch: files,
  sockets, processes, env, entropy. An API review is a different activity.
- **"Gate" is not a linter run.** Linters judge style; the gate enforces a
  declared contract bidirectionally and fails the build on silence.
- **"Claim" is not a comment.** It is queryable CUE data that survives
  refactors and is checked against facts (`claim-conflict` flags a lying claim).
- **"Baseline" is not suppression-at-all-costs.** It is deliberate, dated,
  owned grandfathering that only shrinks.
- **"Detector" is not a gate rule.** Detectors find bugs anywhere at info;
  gate rules enforce the manifest contract. Different severity economics.
