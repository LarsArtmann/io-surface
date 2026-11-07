# httputil/server_timing — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — W3C Server-Timing middleware for HTTP handlers; no handler exists to time.**
> Latest version: **server_timing/v1.0.3** · module `github.com/larsartmann/httputil/server_timing`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

`server_timing` middleware that emits W3C Server-Timing headers (request
phases as `Server-Timing: name;dur=…` metrics) for Go HTTP handlers.

## Should io-surface use it?

No. Subsumed by the [httputil](httputil.md) verdict: io-surface serves no
HTTP, so there is no response to attach timing headers to. Internal timing is
not a need either — `check` and `lint` are sub-second batch runs whose
duration is already visible to any caller that cares (`time io-surface check`).

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg server_timing go.mod go.sum             # no hits
```
