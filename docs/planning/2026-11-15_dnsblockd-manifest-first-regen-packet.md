# dnsblockd manifest-first regen — owner decision packet

**Date:** 2026-11-15 · **Status:** DECISION REQUESTED (one call) · **Supersedes:** the
regen section of `2026-11-08_dnsblockd-v12-session-packet.md` · **Tracks:** io-surface
TODO #27 (BLOCKED upstream), #146 (closed: conflicts dissolved by the #139 overlay)

## What

dnsblockd's manifest carries **151 declared sites, 122 extractable** — 29 sites exist
only as declarations (seam functions, `extractable: false`, token-tier indirections).
io-surface's `init --force` regenerates `facts.cue` from pure extraction and
**drops every declared-but-blind site** (empirically proven 2026-11-15 at both
v0.5.1 and master `v0.5.1-88-gc4c10e1`: 151 declared → 113/122 written, 0
extractable survive). It also stamps the scaffold-default `scope:` unconditionally,
which conflicts under CUE unification with any package that customizes scope.

The needed upstream feature: a **manifest-first `--force`** that

1. refreshes extracted sites (locations, calls) from the tree,
2. keeps declared-but-not-extracted sites with their claims/baseline intact
   (facts.cue already carries `extractable: false` — the data survives unification,
   only the WRITER drops it),
3. skips `scope:` when the existing manifest declares one.

That is exactly the emit semantics of dnsblockd's own `tools/io-extract` — which is
why #27 cannot retire the tool yet.

## Why it matters

- dnsblockd's v1.2 migration (inline claims → claims.cue) is **prepared and proven
  clean** (0 site mismatches in `cue export`) and the v1.2 diff review LANDED
  2026-11-15 (MIGRATION-v1.2.md §5.2; fleet CLI 0 errors/0 warnings; all three
  dnsblockd io gates green; retirement experiment + revert recorded in dnsblockd
  commits `83f35759` + `b16bc0cc`).
- The ONLY remaining blocker for the Nix-check flip + `tools/io-extract` retirement
  is this one upstream writer behavior. Without it, every facts refresh on dnsblockd
  silently amputates 29 declared sites — unacceptable for a gated repo.

## Options

| # | Option | Cost | Tradeoff |
| - | ------ | ---- | -------- |
| A | Implement manifest-first `--force` in io-surface (seed regen from the existing manifest: union of extracted + declared-blind sites; skip scope when declared) | ~1 session (scaffold writer + fixture tests; schema unchanged — v1.2 suffices) | The clean fix; benefits every fleet repo with seam sites, not just dnsblockd |
| B | dnsblockd keeps `tools/io-extract` forever, never runs io-surface `init --force` | 0 upstream cost | Permanent fork of the regen path; the tool the pilot was meant to retire stays; every facts refresh is a bespoke script run |
| C | Manual merge ritual: run `--force`, then re-apply the 29 blind sites from a saved copy (`apply-extractable-sites.py` pattern, already in the manifest dir) | 0 upstream cost, ~15 min per refresh | Works today; relies on ritual discipline; a missed merge fails the gate LOUDLY (undeclared-site errors turn into silent drift only if baseline hides them) |

## Recommendation

**Option A.** The writer semantics are already half-proven (dnsblockd's
`tools/io-extract` implements the target behavior; `apply-extractable-sites.py`
demonstrates the merge direction). Schema impact: none expected — facts.cue already
represents every needed state; if the spike in ROADMAP Q-schedule (schema v1.3,
TODO #135/#136) lands first, the writer should ride it, but does not require it.
Suggested acceptance test: round-trip on the dnsblockd manifest itself —
`init --force` → `check` green, 151 declared preserved, byte-stable claims/baseline.

## Verification trail (every claim names its command)

- "29 blind sites": `go run ./cmd/io-surface check --root ~/projects/dnsblockd` →
  "151 declared, 122 extracted" (2026-11-15).
- "drops blind sites": concurrent-session empirical proof 2026-11-15 at v0.5.1 and
  master `v0.5.1-88-gc4c10e1` (regen output counts in the TODO #27 evidence).
- "conflicts dissolved": same check command → 0 errors, 0 warnings, 8 infos,
  0 claim-conflict findings (2026-11-15) — TODO #146 closed on this.
- "migration clean": dnsblockd MIGRATION-v1.2.md §5.2 (review landed, commits
  `83f35759`, `b16bc0cc`).
