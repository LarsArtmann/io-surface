# httputil/endpoints — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — self-describing `/endpoints` route listing for HTTP services; io-surface exposes CLI subcommands, not routes.**
> Latest version: **endpoints/v0.1.0** (2026-11-06, newest httputil-family tag) · module `github.com/larsartmann/httputil/endpoints`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

A separate httputil module that derives a self-describing `/endpoints` page
from Go 1.27's pattern-based `http.ServeMux` registrations — a service's
route inventory rendered for humans.

## Should io-surface use it?

No — same root cause as [httputil](httputil.md): no server. The conceptual
cousin it embodies (self-describing surface inventory) is io-surface's own
product goal, already satisfied for the CLI surface by cmdguard's
`describe` command (versioned JSON contract of every command and flag,
wired via `WithBatteries` in `cmd/io-surface/main.go:160`).

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg 'httputil/endpoints' go.mod go.sum      # no hits
go run ./cmd/io-surface describe | head    # the CLI-surface equivalent, already shipped
```
