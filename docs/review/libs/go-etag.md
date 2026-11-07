# go-etag — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — two-sided HTTP ETag middleware/transport; io-surface serves no HTTP and makes no requests.**
> Latest version: **v0.7.0** (2026-11-03; master 1 commit ahead) · modules: root (deprecated alias), `go-etag/server`, `go-etag/client`, `go-etag/entitytag`, `go-etag/metrics`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

RFC 7232 both sides: server middleware that generates ETags and answers
`If-None-Match` with 304, plus a client `http.RoundTripper` implementing a
correct conditional-GET cache (RFC 9111 freshening, invalidation, bounded
memory).

## Should io-surface use it?

No. io-surface opens no sockets in either direction: the extractor is
stdlib-only AST walking (`internal/extract`), the manifest loads from disk,
and even the capability engine is run externally and pinned (`absorb`,
README "run externally and pinned — io-surface never imports" it). There is
no `http.Server` and no `http.Client` in the entire binary.

Adoption trigger: a served mode (e.g. editor integration exposing check
results over HTTP) — not planned.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg -n 'net/http' --glob '*.go' internal/ cmd/ provider/   # no server/client code
rg go-etag go.mod go.sum                                    # no hits
```
