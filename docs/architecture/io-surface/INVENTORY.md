# io-surface inventory

This repository's own I/O surface, rendered from the manifest in this
directory. The generated block below is owned by the renderer: edit
`facts.cue` (mechanical facts) or `claims.cue` (human judgment), then run
`go run ./cmd/io-surface render --root . --doc docs/architecture/io-surface/INVENTORY.md`.
CI fails when this block goes stale (`render --doc … --check`).

<!-- io-surface:begin -->

## Taxonomy at a glance

| Family           | Sites | Claimed | With rationale |
| ---------------- | ----: | ------: | -------------: |
| `file-read`      |     8 |       2 |              2 |
| `file-write`     |     5 |       2 |              2 |
| `embed`          |     1 |       0 |              0 |
| `ipc-subprocess` |     2 |       1 |              1 |

## `file-read` (8)

| Site                                         | Location                            | Resource                                                  | Trigger                                  | Discipline | Crit | Durab |
| -------------------------------------------- | ----------------------------------- | --------------------------------------------------------- | ---------------------------------------- | ---------- | ---- | ----- |
| `internal/absorb.ModulePath`                 | `internal/absorb/absorb.go:78`      | go.mod of the analyzed repo                               | every absorb run without --module        | —          | —    | —     |
| `internal/check.symbolVisible`               | `internal/check/location.go:28`     | —                                                         | —                                        | —          | —    | —     |
| `internal/extract.extractor.collectSeamDecl` | `internal/extract/extract.go:171`   | candidate Go source files during the seam-collection pass | pass A of every two-pass extraction walk | bounded    | —    | —     |
| `internal/extract.extractor.parseFile`       | `internal/extract/extract.go:234`   | —                                                         | —                                        | —          | —    | —     |
| `internal/extract.extractor.walk`            | `internal/extract/extract.go:94`    | —                                                         | —                                        | —          | —    | —     |
| `internal/extract.runExtract`                | `internal/extract/tables.go:107`    | —                                                         | —                                        | —          | —    | —     |
| `internal/manifest.Load`                     | `internal/manifest/manifest.go:192` | —                                                         | —                                        | —          | —    | —     |
| `internal/manifest.Manifest.Validate`        | `internal/manifest/manifest.go:402` | —                                                         | —                                        | —          | —    | —     |

### Judgments

- **`internal/absorb.ModulePath`** — a single small read to resolve the first-party package prefix; --module exists for manifests analyzed outside their repo
- **`internal/extract.extractor.collectSeamDecl`** — the substring prefilter gates the parse: a file is read and parsed only when it contains `func <watched-seam>(`, so the extra pass reads the tree once more but parses almost nothing — the cross-package seam index is worth one sequential read for caller-before-declarer resolution

## `file-write` (5)

| Site                               | Location                                 | Resource                                   | Trigger                             | Discipline | Crit | Durab |
| ---------------------------------- | ---------------------------------------- | ------------------------------------------ | ----------------------------------- | ---------- | ---- | ----- |
| `cmd/io-surface.applyDoc`          | `cmd/io-surface/commands.go:301`         | —                                          | —                                   | —          | —    | —     |
| `cmd/io-surface.runAbsorb`         | `cmd/io-surface/commands.go:215`         | Capslock JSON report passed via --capslock | every `io-surface absorb` run       | bounded    | —    | —     |
| `cmd/io-surface.runRender`         | `cmd/io-surface/commands.go:294`         | —                                          | —                                   | —          | —    | —     |
| `internal/manifest.AppendBaseline` | `internal/manifest/baseline_write.go:41` | baseline.cue of the manifest being edited  | every `io-surface baseline add` run | —          | —    | —     |
| `internal/scaffold.Write`          | `internal/scaffold/scaffold.go:61`       | —                                          | —                                   | —          | —    | —     |

### Judgments

- **`cmd/io-surface.runAbsorb`** — the operator names the artifact; the engine output is decoded in memory with no streaming copy, so an oversized report fails loudly at decode rather than silently truncating
- **`internal/manifest.AppendBaseline`** — the machine's only sanctioned write into a judgment file: a surgical insertion of one {id, note, date} entry, refusing duplicates and noteless entries so the baseline keeps its only-shrinks shape; the read-modify-write is single-file, unbuffered, and fails loudly on a missing file

## `embed` (1)

| Site                          | Location                           | Resource | Trigger | Discipline | Crit | Durab |
| ----------------------------- | ---------------------------------- | -------- | ------- | ---------- | ---- | ----- |
| `internal/scaffold.schemaCUE` | `internal/scaffold/scaffold.go:33` | —        | —       | —          | —    | —     |

## `ipc-subprocess` (2)

| Site                      | Location                            | Resource                                        | Trigger                                           | Discipline | Crit | Durab |
| ------------------------- | ----------------------------------- | ----------------------------------------------- | ------------------------------------------------- | ---------- | ---- | ----- |
| `internal/absorb.gitHead` | `internal/absorb/absorb.go:165`     | git HEAD + worktree status of the analyzed repo | every absorb run without an explicit audit commit | —          | —    | —     |
| `internal/scaffold.git`   | `internal/scaffold/scaffold.go:228` | —                                               | —                                                 | —          | —    | —     |

### Judgments

- **`internal/absorb.gitHead`** — provenance only: the capability audit must name the revision the engine actually analyzed; falls back to the literal working-tree outside git so sandboxed runs stay valid

## Reachability (capabilities)

> Engine-observed by capslock v0.3.4 of `5a6c289-dirty` (2026-11-06): what each package can REACH, beyond its declared sites.
> `direct` = through the package's own code (high signal); `deep` adds transitive
> dependency paths (stdlib linkage reaches everywhere and carries no per-package signal).
> Advisory: a package whose direct capability has no covered family is a `shadow-capability` info.

| Package             | Direct             | Deep (transitive)          | Judgment                                                                                                                                                                                                         |
| ------------------- | ------------------ | -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/io-surface`    | `FILES`, `NETWORK` | `EXEC`, `FILES`, `NETWORK` | capslock v0.3.4 direct-NETWORK edge here is an SSA conflation phantom (json/v2 stream writer unified with http.response; zero network use — blind-spot run 2026-11-06, false-positive class 2)                   |
| `internal/absorb`   | `EXEC`, `FILES`    | `EXEC`, `FILES`, `NETWORK` | —                                                                                                                                                                                                                |
| `internal/check`    | `FILES`            | `EXEC`, `FILES`, `NETWORK` | —                                                                                                                                                                                                                |
| `internal/emit`     | —                  | `EXEC`, `FILES`, `NETWORK` | —                                                                                                                                                                                                                |
| `internal/extract`  | `FILES`, `NETWORK` | `EXEC`, `FILES`, `NETWORK` | capslock v0.3.4 direct-NETWORK edges here are SSA range-func conflation (strings.FieldsSeq$1 -> net/http.ParseCookie$1; the package is hermetic by contract — blind-spot run 2026-11-06, false-positive class 1) |
| `internal/manifest` | `FILES`            | `EXEC`, `FILES`, `NETWORK` | —                                                                                                                                                                                                                |
| `internal/render`   | —                  | `EXEC`, `FILES`, `NETWORK` | —                                                                                                                                                                                                                |
| `internal/scaffold` | `EXEC`, `FILES`    | `EXEC`, `FILES`            | —                                                                                                                                                                                                                |

## Deliberately out of scope

- timers and tickers (scheduling)
- memory pooling
- kernel-internal I/O (epoll, page cache)

<!-- io-surface:end -->

Prose and context live around the block; only the generated section between
the markers is rewritten.
