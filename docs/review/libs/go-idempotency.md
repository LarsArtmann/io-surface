# go-idempotency — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — solves at-least-once delivery; io-surface has no delivery, no commands, no store.**
> Latest version: **v0.3.2** (2026-11-03; master 5 commits ahead) · module `github.com/larsartmann/go-idempotency`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

Turns at-least-once delivery into at-most-once processing: clients attach a
stable idempotency key; `CheckAndRecord(key, ttl)` atomically records and
rejects retries (`ErrDuplicate`), with pluggable stores. Built for CQRS
command handling.

## Should io-surface use it?

No. io-surface is a stateless one-shot gate: it walks a local tree, loads a
CUE manifest, prints findings. There is no message delivery, no
acknowledgement loss, no retrying client, nowhere to store keys. Every input
is a file read; every invocation is independent by design.

Adoption trigger (for honesty): a daemonized/service mode with submitted jobs
would need exactly this — no such mode exists on any roadmap.

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-idempotency go.mod go.sum          # no hits
git -C /home/lars/projects/go-idempotency tag --sort=-creatordate | head -1   # v0.3.2
```
