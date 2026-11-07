# go-health — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — Kubernetes three-probe SDK for samber/do services; io-surface has no probes, no container, no long-running process.**
> Latest version: **v0.10.0** (2026-11-04; master 1 commit ahead) · module `github.com/larsartmann/go-health`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

Turns the K8s liveness/readiness/startup pattern into a single `Probe` type
over samber/do v2 containers: critical/non-critical classification,
background caching, shutdown awareness, metrics, middleware, federation.

## Should io-surface use it?

No. The tool's health model is its exit code — 0 clean / 1 findings /
2 usage or internal (`cmd/io-surface/main.go:10` documents the contract).
That is the correct health contract for a batch gate: the caller (CI,
BuildFlow via the provider in `provider/`) decides what to do with it. There
is nothing alive to probe mid-run.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-health go.mod go.sum                 # no hits
```
