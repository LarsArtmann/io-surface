# go-atomic-write — adoption & utilization review

> **Status: USED — direct dependency** (`github.com/larsartmann/go-atomic-write v0.6.2`)
> **Verdict: leveraged to the max, including the advanced verified-write path; bump SHIPPED 2026-11-08 (were two free patch bumps).**
> **Adoption score: 90/100 (high, re-scored at the 2026-11-08 bump; was 80 pre-bump)** — 4 of 5 applicable capabilities fully leveraged, 1 partial (opportunistic)
> Latest version: **v0.6.2** (2026-11-04, module proxy confirms) · io-surface pins **v0.6.2** (bumped from v0.6.0, plan M2)
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What the library offers

Atomic file writes for Go: stage to a unique temp file, fsync, rename into
place — a reader never sees a half-written file. Public surface (v0.6.x):
`Write` (inherits mode), `WriteWithPerm` (explicit mode), `WriteFunc`/
`WriteFuncVerified` (streaming via func), `WriteIfChanged` (skips identical
content), `FingerprintFromBytes` + `WriteVerified` (xxhash64 optimistic-
concurrency guard, `ErrConcurrentModification` sentinel), and MkdirAll of the
parent directory on every write entrypoint (unreleased, master). v0.6.1 fixed
a temp-file leak when the final rename fails; v0.6.2 was website/CI work; the
`WriteFuncWithPerm` variant is unreleased on master.

io-surface is one of the reference consumers: its own extractor even models
the atomic-write API surface as first-class detection tables
(`internal/extract/tables.go:291-293` — `atomicwrite.Write`, `WriteWithPerm`,
`WriteVerified`, `WriteFunc`, `WriteIfChanged` are all known call shapes in
the `atomicwrite` seam family).

## io-surface today (evidence)

Five write sites, all through the library, none raw:

| Site                                                               | Call                                                                                               | Why this variant                                                                                                                       |
| ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/io-surface/commands.go:244` (absorb → capabilities.cue)       | `WriteWithPerm(…, 0o644)`                                                                          | machine-written file must land with canonical perms                                                                                    |
| `cmd/io-surface/commands.go:294` (render --out)                    | `WriteWithPerm(…, 0o644)`                                                                          | same                                                                                                                                   |
| `internal/scaffold/scaffold.go:80` (init, all four manifest files) | `WriteWithPerm(…, 0o644)`                                                                          | manifest files are committed; mode must be stable for CI diffs                                                                         |
| `cmd/io-surface/commands.go:305,338` (render --doc)                | `FingerprintFromBytes` + `WriteVerified`                                                           | in-place rewrite of a human-edited doc: the fingerprint guard turns a lost update into an error instead of silent clobber              |
| `internal/manifest/baseline_write.go:45,62-63` (baseline add)      | `FingerprintFromBytes` + `WriteVerified` + `errors.Is(err, atomicwrite.ErrConcurrentModification)` | surgical insertion into a judgment file; concurrent-edit race is caught and re-mapped to a domain error telling the operator to re-run |

That last row is the textbook use of the library's advanced path: the only
sanctioned machine write into `baseline.cue` is guarded against concurrent
modification and the sentinel is handled explicitly with a user-facing fix
message — exactly the library's intended contract.

## Capability inventory

| # | Capability                                                        | Status             | Evidence / reason                                                                                                                                                                                                                                                                      |
| - | ----------------------------------------------------------------- | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Atomic staged writes with explicit perms (`WriteWithPerm`)        | 🟢 Fully leveraged | all machine-written files, 3 call sites, consistent 0o644                                                                                                                                                                                                                              |
| 2 | Optimistic concurrency (`FingerprintFromBytes` + `WriteVerified`) | 🟢 Fully leveraged | both in-place rewrite paths guarded (applyDoc, AppendBaseline)                                                                                                                                                                                                                         |
| 3 | `ErrConcurrentModification` sentinel handling                     | 🟢 Fully leveraged | baseline_write.go:63 remaps to `ErrBaseline` with re-run guidance                                                                                                                                                                                                                      |
| 4 | Streaming writers (`WriteFunc`, unreleased `WriteFuncWithPerm`)   | ⚪ N/A             | every write source is an in-memory `[]byte` (CUE builder / markdown string); nothing streams                                                                                                                                                                                           |
| 5 | `WriteIfChanged` (skip identical content)                         | 🟡 Partially used  | `render --doc` rewrites identical bytes when fresh==stored… actually the `--check` path (CI) compares before writing; the write path only runs on demand. Residual impact: mtime-only churn. The variant is worth adopting only if `--doc` rewrites ever become scheduled              |
| 6 | Version currency                                                  | 🟢 Fully leveraged | Pinned **v0.6.2** since 2026-11-08 (plan M2 bump, gates green). v0.6.1's temp-leak fix (staged `.tmp` not removed when rename fails) now hardens the `render --out` paths; the unreleased MkdirAll fix is irrelevant here (scaffold MkdirAlls the manifest dir itself, scaffold.go:66) |

Score derivation (skill rubric): N=6 rows, row 4 excluded as N/A → A=5
applicable; fully=4 (rows 1,2,3,6), partial=1 (row 5) →
round(100 × (4 + 0.5)/5) = **90 (high; re-scored 2026-11-08 at the M2 bump —
was 80 while the pin lagged)**. The one partial is opportunistic, not
corrective.

## Recommendation

**Bumped v0.6.0 → v0.6.2 on 2026-11-08** (plan M2, full gates green) — the
temp-leak fix now protects the `--out`/`--doc` paths. No API changes in the
gap. Beyond that, usage is at
the library's ceiling for this kind of tool: every write it should own, it
owns, with the strongest correct variant per site. Revisit `WriteIfChanged`
only if a scheduled `render --doc` job ever appears.

## How to verify

```bash
rg -n 'atomicwrite\.' cmd/ internal/ --glob '!*_test.go'   # 8 call sites (5 writes + fingerprint/read companions) + 3 extractor table keys
go list -m github.com/larsartmann/go-atomic-write           # v0.6.2 pinned (bumped 2026-11-08, plan M2)
go list -m github.com/larsartmann/go-atomic-write@latest    # v0.6.2
awk '/^## \[0.6.1\]/,/^## \[0.6.0\]/' /home/lars/projects/go-atomic-write/CHANGELOG.md
go test ./internal/manifest/ -run AppendBaseline -v          # concurrent-modification path is tested
```
