# go-credentials-store — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — encrypted-at-rest credentials store with event sourcing and admin UI; io-surface holds no secrets, by design.**
> Latest version: **v0.3.1** (module proxy confirms, 2026-11-08 — plan M7; the local clone's newest tag is v0.2.1 and its README reference to a published v0.3.1 was correct, the clone is just behind) · module `github.com/larsartmann/go-credentials-store`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

Event-sourced, XChaCha20-Poly1305-sealed storage for API keys, Playwright
session blobs, and passwords, with lifecycle audit (created/rotated/probed/
used/revealed), a single-operator loopback admin UI over SSE, Prometheus
metrics — built on go-cqrs-lite's `system` root + `metaengine`.

## Should io-surface use it?

No — and the rejection is doctrinal, not incidental: io-surface is a gate
that must run hermetically in CI with zero secrets (gofmt-clean claim: no
credentials anywhere in the tree; the BuildFlow provider runs with repo
access only). There is no API to key, no session to persist, and adopting a
credentials store would itself add the I/O-and-secrets surface the tool
exists to make explicit.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-credentials-store go.mod go.sum      # no hits
rg -n 'password|apikey|api_key' --glob '*.go' internal/ cmd/   # nothing to store
go list -m github.com/larsartmann/go-credentials-store@latest   # v0.3.1 (proxy-confirmed 2026-11-08)
```
