# httputil — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — composable HTTP server middleware; io-surface has no HTTP surface of its own.**
> Latest version: **v1.8.0** (root; master 16 commits ahead) · module `github.com/larsartmann/httputil`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

Server middleware and lifecycle helpers: CORS, client-IP extraction, response
recording, chaining, security headers, CSP nonces, request IDs, panic
recovery, timeouts, structured logging, compression, request-bomb protection,
rate limiting, CSRF (nosurf), health checks — pure stdlib `net/http` with four
deliberate dependencies.

## Should io-surface use it?

No. The whole point of io-surface is that a tree's I/O surface is small and
declared; the tool itself practices that doctrine — it performs zero network
I/O, so there is no handler chain to wrap. Notably, io-surface's own README
positions it as the tool you run _because_ you added `httputil`-style surface
to your app: the extractor classifies `httputil` imports via its import
tables when scanning _consumers_, which is the only relationship between the
two projects.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg httputil go.mod go.sum                  # no hits
rg -n 'net/http' --glob '*.go' .           # none in the binary's own code
```
