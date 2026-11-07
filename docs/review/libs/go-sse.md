# go-sse — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — SSE transport, fan-out, and replay; io-surface is not a server and streams nothing.**
> Latest version: **v0.7.0** (2026-11-04; master 1 commit ahead; `sseparse`/`ssetest` submodules) · module `github.com/larsartmann/go-sse`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

Server-Sent Events plumbing: wire-format types and an allocation-minimized
serializer, connection `Stream` handling (heartbeats, Last-Event-ID),
generic `Broadcaster[T]` fan-out, `EventStore` reconnection replay, and
predicate push-down filters.

## Should io-surface use it?

No. The binary never holds a connection open — batch in, findings out, exit
0/1/2. Even the CI-facing outputs (SARIF, GitHub annotations) are
write-once stdout formats, not streams. Nothing subscribes, nothing
replays, nothing broadcasts.

Adoption trigger: a watch/live mode pushing re-check results to an editor —
that would build on this plus [go-filewatcher](go-filewatcher.md); neither
is planned (see both reports' triggers).

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-sse go.mod go.sum                    # no hits
```
