# templ-components — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — templ + HTMX + Tailwind v4 UI component library; io-surface has no UI layer to compose.**
> Latest version: **v1.27.0** (root; `charts/echarts/v1.27.0` tagged 2026-11-04; master 87 commits ahead) · module `github.com/larsartmann/templ-components` (+ `htmx`, `datastar`, `icons`, `errorpage`, `utils`, `visualtest` submodules)
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

The house web UI kit: typed templ components (props structs, typed enums,
style lookups, icon paths), HTMX/Datastar action wiring, CSP-safe rendering,
dark-mode awareness, visual regression test helpers.

## Should io-surface use it?

No. io-surface renders exactly two human-facing things, neither HTML:
terminal findings (plain text, byte-pinned) and markdown inventories
(`internal/render`, embedded into docs via `render --doc`). Adopting a UI
kit would add templ codegen, Tailwind, and a components dependency to a
binary whose selling point is a stdlib-only core and four direct deps.

If io-surface ever grows a report *website*, the right split stays: generate
static markdown/HTML from the manifest (render's current job), not a live
component tree.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg templ-components go.mod go.sum          # no hits
rg -n '\.templ' --glob '*.templ' .         # no templ sources exist
```
