# go-datastar — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — server-side DataStar protocol over SSE; io-surface has no browser-facing surface.**
> Latest version: **v0.7.0** (root; `broadcast/v0.7.0` tagged 2026-11-07; master 4 commits ahead; `datastartest` submodule) · module `github.com/larsartmann/go-datastar`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

The DataStar hypermedia protocol as values: every patch is a first-class
value producing an `sse.Event`, composable with go-sse's broadcaster/replay
infrastructure — DOM patches, signal updates, script execution from server
to browser.

## Should io-surface use it?

No. There is no HTML, no browser client, no open connection — the human
interface is a terminal (and optional markdown rendering via `internal/render`).
Adoption would drag in [go-sse](go-sse.md) for a UI that does not exist.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-datastar go.mod go.sum               # no hits
```
