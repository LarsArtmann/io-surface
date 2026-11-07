# go-health-dashboard — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — real-time Datastar dashboard for go-health probes; doubly inapplicable (no probes, no UI).**
> Latest version: **v0.13.2** (2026-11-05; master 13 commits ahead) · module `github.com/larsartmann/go-health-dashboard`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

A browser dashboard over [go-health](go-health.md): status banners, stat
cards, trend sparklines, severity-grouped service tables, sub-second SSE
updates, plus a standalone federation hub binary — content-negotiated so the
kubelet still gets JSON.

## Should io-surface use it?

No. Requires go-health probes (io-surface has none — see that report) and a
served UI (io-surface has none — terminal + markdown only). The nearest
io-surface analog, the human-facing inventory, already exists as
`render --doc` markdown embedded in architecture docs.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-health-dashboard go.mod go.sum       # no hits
```
