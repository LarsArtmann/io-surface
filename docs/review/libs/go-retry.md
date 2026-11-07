# go-retry — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject — nothing in io-surface retries, and the gate must stay deterministic.**
> Latest version: **v0.9.0** (2026-10-27; master 4 commits ahead) · module `github.com/larsartmann/go-retry`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

A dependency-light retry loop with exponential backoff and additive jitter,
pluggable retryable predicate built on go-error-family's `IsRetryable` —
explicitly scoped for CLIs and batch jobs (no CQRS types, no OTel SDK).

## Should io-surface use it?

No — despite the "for CLIs" framing being the closest fit of any rejected
library. io-surface's operations are local filesystem reads, an in-process
CUE load, and stdout writes. None has a transient-failure mode worth
retrying: a missing manifest must fail immediately (that is the gate working),
and a parse error is deterministic by definition. Retrying anything would
make the gate's output timing-dependent — the opposite of its contract.

Adoption trigger: a networked mode (e.g. fetching remote manifests or a
capability engine) — explicitly out of scope (hermetic-by-design, ROADMAP).

## Recommendation

**Do not adopt.**

## How to verify

```bash
rg go-retry go.mod go.sum                # no hits
git -C /home/lars/projects/go-retry tag --sort=-creatordate | head -1        # v0.9.0
```
