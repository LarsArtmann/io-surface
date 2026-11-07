# cqrs-htmx — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — CQRS + HTMX web-app library (usermgmt, adminui, SSE-only since v5); no web app exists to build on it.**
> Latest version: **v4.18.1** (root `v4` module; newest nested tag `huma/v4.0.0`, 2026-11-07) · modules `github.com/larsartmann/cqrs-htmx/{v4,usermgmt,adminui,dashboardui,setup,…}`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

The web face of the house CQRS stack: CQRS command/query HTTP handlers,
HTMX/Datastar responses, event-sourced user management (WebAuthn/OAuth2/
TOTP), a ready-made admin UI, setup flows, SSE transport (WebSocket removed
in v5 per its ADR 0046).

## Should io-surface use it?

No. io-surface is not a web application and takes no user input beyond CLI
flags. Even its most seductive module — `usermgmt` — solves a problem
(authenticated multi-user access) that a local deterministic gate does not
have. Adoption would transitively pull go-cqrs-lite, templ, go-sse, and
go-datastar: four rejections compounding into one.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg cqrs-htmx go.mod go.sum                 # no hits
```
