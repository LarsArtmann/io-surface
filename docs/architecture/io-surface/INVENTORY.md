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
| `file-read`      |     7 |       0 |              0 |
| `file-write`     |     4 |       0 |              0 |
| `embed`          |     1 |       0 |              0 |
| `ipc-subprocess` |     2 |       0 |              0 |

## `file-read` (7)

| Site                                   | Location                            | Resource | Trigger | Discipline |
| -------------------------------------- | ----------------------------------- | -------- | ------- | ---------- |
| `internal/absorb.ModulePath`           | `internal/absorb/absorb.go:78`      | —        | —       | —          |
| `internal/check.symbolVisible`         | `internal/check/location.go:23`     | —        | —       | —          |
| `internal/extract.Extract`             | `internal/extract/extract.go:111`   | —        | —       | —          |
| `internal/extract.extractor.parseFile` | `internal/extract/extract.go:276`   | —        | —       | —          |
| `internal/extract.extractor.walk`      | `internal/extract/extract.go:237`   | —        | —       | —          |
| `internal/manifest.Load`               | `internal/manifest/manifest.go:147` | —        | —       | —          |
| `internal/manifest.Manifest.Validate`  | `internal/manifest/manifest.go:334` | —        | —       | —          |

## `file-write` (4)

| Site                       | Location                           | Resource | Trigger | Discipline |
| -------------------------- | ---------------------------------- | -------- | ------- | ---------- |
| `cmd/io-surface.applyDoc`  | `cmd/io-surface/commands.go:241`   | —        | —       | —          |
| `cmd/io-surface.runAbsorb` | `cmd/io-surface/commands.go:155`   | —        | —       | —          |
| `cmd/io-surface.runRender` | `cmd/io-surface/commands.go:234`   | —        | —       | —          |
| `internal/scaffold.Write`  | `internal/scaffold/scaffold.go:56` | —        | —       | —          |

## `embed` (1)

| Site                          | Location                           | Resource | Trigger | Discipline |
| ----------------------------- | ---------------------------------- | -------- | ------- | ---------- |
| `internal/scaffold.schemaCUE` | `internal/scaffold/scaffold.go:30` | —        | —       | —          |

## `ipc-subprocess` (2)

| Site                      | Location                            | Resource | Trigger | Discipline |
| ------------------------- | ----------------------------------- | -------- | ------- | ---------- |
| `internal/absorb.gitHead` | `internal/absorb/absorb.go:165`     | —        | —       | —          |
| `internal/scaffold.git`   | `internal/scaffold/scaffold.go:221` | —        | —       | —          |

## Deliberately out of scope

- timers and tickers (scheduling)
- memory pooling
- kernel-internal I/O (epoll, page cache)

<!-- io-surface:end -->

Prose and context live around the block; only the generated section between
the markers is rewritten.
