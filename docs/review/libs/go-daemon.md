# go-daemon — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — unix-socket daemon lifecycle machinery; io-surface is a one-shot CLI that owns no socket.**
> Latest version: **v0.4.0** (2026-10-03; master 32 commits ahead) · module `github.com/LarsArtmann/go-daemon` (capital-L path)
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

Long-lived-daemon building blocks over unix domain sockets: `SocketServer`
with stale-socket cleanup and an ordered lifecycle hook contract, sd_notify
readiness/watchdog, graceful drain, socket HTTP transports, JSON/CBOR
content negotiation, WHATWG-conformant SSE parsing, and a flight recorder
with slow-request auto-trigger.

## Should io-surface use it?

No. io-surface's runtime model is: parse flags, walk files, print, exit.
No listener, no lifecycle beyond process exit, no client sessions. A
daemonized io-surface (LSP-style resident analyzer) is imaginable but has
no demand and would trade the tool's strongest property — hermetic,
deterministic, zero-socket CI behavior — for latency nobody measured as a
problem.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-daemon go.mod go.sum                 # no hits
rg -n 'net\.Listen|unix\.Socket' --glob '*.go' .   # no listener anywhere
```
