# go-cqrs-lite — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — CQRS + event-sourcing composition root; io-surface is stateless and deterministic by design.**
> Latest version: **system/v4.17.0** (newest nested tag `catalog/v4.14.0`, 2026-11-07) · modules `github.com/larsartmann/go-cqrs-lite/{system,metaengine,event,command,…}/v4` (42 modules)
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

The composable CQRS/event-sourcing library: the `system` composition root
(`system.New(ctx, DomainConfig, DeploymentConfig)` auto-wiring store, bus,
projections, snapshots, timers, health), the cost-based `metaengine`
planner over interchangeable engines (memory, sqlite, turso, pebble, bbolt,
badger, duckdb, postgres), and the low-level event/command/decider modules.

## Should io-surface use it?

No. io-surface's "events" are findings — write-once, printed, exit. There is
no state to source: extraction is a pure function of the tree, the manifest
is declarative CUE, and determinism (same tree → same findings) is the
product. Event-sourcing the manifest would add stores, projections, and
engines — a second, heavier state model next to CUE for zero capability.

The genuine relationship is already implemented and ships today: the
extractor's import tables classify `go-cqrs-lite/system/v4` as
`cqrs-system` (`internal/extract/tables.go:141`), so io-surface _audits_
CQRS systems as a first-class target.

## Recommendation

**Do not adopt.** Keep the tool-to-target relationship one-directional.

## How to verify

```bash
rg go-cqrs-lite go.mod go.sum              # no hits (tables.go mentions are string keys, not imports)
rg -n 'cqrs-system' internal/extract/tables.go
```
