# go-filewatcher — adoption & utilization review

> **Status: NOT USED** (absent from the module graph)
> **Verdict: reject today — the one rejected library with a real, nameable future fit (`check --watch`); defer until pilot demand exists.**
> Latest version: **v2.5.0** (2026-11-06; master 4 commits ahead) · module `github.com/larsartmann/go-filewatcher/v2`
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What it offers

fsnotify-based watching without the boilerplate: automatic recursion, 20
built-in filters plus AND/OR/NOT combinators, 19 middleware (debounce,
coalesce, retry), ENOSPC resilience, observability hooks — production-grade
by design (`-race` tested).

## Should io-surface use it?

Not today. io-surface is a one-shot batch gate; CI invokes it and exits on
the result. A watch mode would be a new product surface (long-running
process, incremental re-extraction, output clearing) — genuine scope, not a
plug-in.

But it is the honest exception in this series: `io-surface check --watch`
(re-check on save, dev-loop style) is a plausible pilot-request feature, and
if it happens this library is the correct foundation rather than raw
fsnotify — the repo's own `--ignore` globs map directly onto its filters,
and debouncing is exactly what editor-save event storms need. The cost is
one new direct dependency, confined to `cmd/` (the stdlib-only doctrine of
`internal/extract` is untouched either way — watching happens above the
extractor).

## Recommendation

**Do not adopt now. Record the trigger:** build `--watch` on this library
if and only if a pilot repo asks for a live dev loop. Do not hand-roll
fsnotify when that day comes.

## How to verify

```bash
rg go-filewatcher go.mod go.sum            # no hits
git -C /home/lars/projects/go-filewatcher tag --sort=-creatordate | head -1   # v2.5.0
```
