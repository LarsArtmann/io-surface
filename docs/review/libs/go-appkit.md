# go-appkit — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — the GOTH application framework (cqrs, httpx, realtime, otel, security, systemd…); io-surface is deliberately none of those things.**
> Latest version: **v0.12.0** (root; newest nested tag `httpx/v0.1.0`, 2026-11-07) · module `github.com/larsartmann/go-appkit` + 15 nested modules
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

The application starter kit for the house stack: `preset` (opinionated app
skeleton), `httpx` (HTTP plumbing), `cqrs` (dispatch over go-cqrs-lite),
`realtime` (SSE), `health`, `otel`, `security`, `errorpages`, `systemd`,
`config`, `flightrecorder` — everything a long-running service needs.

## Should io-surface use it?

No. Every module presupposes a service; io-surface is a hermetic CLI whose
value proposition is *not* being one. The correct-stack argument runs the
other way: io-surface is the gate you install INTO a go-appkit-style service
to declare its I/O surface. Its extractor even recognizes the house CQRS
stack by import path (`internal/extract/tables.go:141` maps
`go-cqrs-lite/system/v4` to a family) — the relationship is tool-to-target,
not dependency.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-appkit go.mod go.sum                 # no hits
rg -n 'cqrs-system' internal/extract/tables.go   # io-surface classifies the stack, doesn't use it
```
