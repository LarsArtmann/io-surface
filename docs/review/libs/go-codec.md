# go-codec — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — CBOR/JSON payload codec for the event-sourcing stack; io-surface serializes findings with encoding/json and manifest data with CUE.**
> Latest version: **v0.3.3** (2026-10-29; master 2 commits ahead) · module `github.com/larsartmann/go-codec`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

The `Codec` contract at the center of the CQRS/event-sourcing family:
CBOR/JSON/Raw payload encoding, `BufferEncoder` zero-allocation hot paths,
`ObservableCodec`, `AutoDetect`, deterministic variants required by signing —
consumed by go-cqrs-lite's storage/event/signing/encryption modules.

## Should io-surface use it?

No. io-surface's serialization surfaces are exactly two, both already owned
by a deliberate choice:

- Findings/report JSON (`--format json`, `extract`) — `encoding/json` on plain structs, pinned by goldens. A payload-codec abstraction over one format would be ceremony.
- The manifest itself — CUE, loaded in-process by `cuelang.org/go` (`internal/manifest`), because the manifest is a _constraint language_, not a payload.

There are no event payloads, no stores, no signing, no hot paths (runs are
sub-second over at most thousands of sites).

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-codec go.mod go.sum                  # no hits
rg -n 'encoding/json' --glob '*.go' cmd/ internal/ | head   # the JSON surface that exists
```
