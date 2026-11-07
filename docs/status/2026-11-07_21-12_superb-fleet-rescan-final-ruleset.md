# io-surface — status: superb verification begins — fleet re-scan with the final 10-rule ruleset

**Date:** 2026-11-07 (superb-verification plan T2)
**Scope:** the fleet evidence regenerated under the shipped ruleset
(write-non-atomic + tiny-buffer included for the first time), the evidence
home fix, and the delta against the released v0.3.0 numbers.

## a) The one thing to remember

The released fleet numbers (08-08 report) described an 8-rule world. The
shipped world is 10 rules, and its numbers are now measured, committed, and
re-derivable: **160/160 repos, 91 with findings, write-non-atomic 340,
tiny-buffer 0** (`docs/evidence/fleetscan-2026-11-07-v031.json`, produced by
`./scripts/fleetscan.sh docs/evidence/fleetscan-2026-11-07-v031.json`).

## b) The delta vs the v0.3.0-era scan (P0 ruleset, 08-08 report)

| Rule              | 08-08 (8 rules) | This run (10 rules) | Delta reading                                                                                    |
| ----------------- | --------------- | ------------------- | ------------------------------------------------------------------------------------------------ |
| io-in-loop        | 349             | 351                 | stable (2-repo drift in the fleet itself; not a rule change)                                     |
| stat-then-open    | 50              | 50                  | unchanged                                                                                        |
| client-no-timeout | 13              | 13                  | unchanged                                                                                        |
| err-dropped       | 11              | 11                  | unchanged (carve-outs already in that run)                                                       |
| ctx-dropped       | 3               | 3                   | unchanged                                                                                        |
| defer-in-loop     | 1               | 1                   | unchanged                                                                                        |
| write-non-atomic  | — (not shipped) | **340**             | the single largest new signal pool; every hit is an un-attempted atomic-write adoption candidate |
| tiny-buffer       | — (not shipped) | **0**               | the loop-gating (consumer must sit in a loop) makes it precise-and-rare; zero fleet noise        |

Repos with findings: 72 → 91 (+19 repos entered signal purely through
write-non-atomic — the rule sees a class the other nine do not).

## c) First decisions this measurement forces (plan T13 inputs)

- write-non-atomic at 340 is too loud to hand-verify at protocol volume (30
  per rule is the T3 sample — fine), and its per-repo density suggests many
  hits are config/data dumps where tearing is acceptable → the T13
  durability-gating decision has real data behind it now.
- tiny-buffer at 0 means the rule card needs no FP section yet and the rule's
  value is preventive (it never fires in the wild today, but the mutation
  corpus proves it fires when the bug appears — recall pinned 100%).

## c2) FP table v2 (plan T3 protocol, adapted to measured counts)

Protocol: 30 findings per rule (or all, when fewer exist), every site's file

- context hand-read. tiny-buffer was sampled as 0 hits (nothing to verify);
  stat-then-open took its protocol slot. Sample: first 30 alphabetically per
  rule over the committed fleet JSON (`docs/evidence/fleetscan-2026-11-07-v031.json`),
  locations re-extracted per repo with the lint face. BuildFlow-pr44 duplicates
  BuildFlow's tree (a PR branch in the fleet) — its 6 hits mirror the original
  line-by-line and are counted TRUE where the original is TRUE.

| Rule              | N  | TRUE | FP | FP rate | Verdict under the promotion policy (W2 < 5%)                                 |
| ----------------- | -- | ---- | -- | ------- | ---------------------------------------------------------------------------- |
| write-non-atomic  | 30 | 25   | 5  | 16.7%   | stays info — carve-out classes must land first                               |
| stat-then-open    | 30 | 13   | 17 | 56.7%   | stays info — dominant FP class is redundant pre-checks; needs semantics work |
| client-no-timeout | 13 | 3    | 10 | 76.9%   | stays info — worse than the earlier 25% pilot estimate                       |

FP classes found (each is a candidate carve-out or rule refinement, none
acted on in this run):

- write-non-atomic (5): writability probes (`.doctor-probe`), test golden
  writes (`-update` path), test-fixture scratch writes, append-journal
  `O_APPEND` handles, lockfile creates.
- stat-then-open (17): redundant existence pre-checks before reads (largest),
  existence branches selecting benign behavior (optional file, fallback),
  mode/symlink safety checks where the stat IS the point.
- client-no-timeout (10): test-harness clients against loopback servers,
  documented per-request `context.WithTimeout` designs, documented streaming
  clients (ResponseHeaderTimeout only, long-lived body), and one
  **post-construction timeout assignment** (`c := &http.Client{}; c.Timeout = …`)
  — a literal-matcher miss worth fixing.

Per-rule go/no-go refresh: **no rule approaches promotion**; write-non-atomic
is closest (16.7% → needs the five mechanical FP classes carved out, then a
re-sample). The promotion-policy ledger (`docs/planning/2026-11-07_promotion-policy.md` §9)
is updated with these rows in the same commit.

## d) Evidence hygiene (plan T4)

- `docs/evidence/README.md` records the provenance contract: fleet-scanned
  numbers cited anywhere must live in `docs/evidence/` or be re-derivable by
  the named command. `/tmp` is not evidence.
- The 08-08 report's `/tmp/io-surface-portfolio-stats.json` citation style is
  superseded by this home (TODO_LIST header updated in the same session).

## e) Verified how (every claim names its command)

- Scan: `./scripts/fleetscan.sh docs/evidence/fleetscan-2026-11-07-v031.json`
  (exit 0; "160/160 repos scanned, 91 with findings" from the script summary).
- JSON committed: `docs/evidence/fleetscan-2026-11-07-v031.json`
  (`findingsByRule` keys carry the io-surface/ prefix).
- Recall corpus: `go test ./internal/extract -run TestMutationRecall -v`
  (10 mutators, 8 rule classes, all at the pinned 100% floor).
- Suppression: `go test ./internal/extract -run TestSuppress -v` (3 tests) and
  the regenerated lint goldens (`go test ./cmd/io-surface -run TestGolden`).

## f) Up to 50 things to get done next

Routed into the superb plan (TODO_LIST #90–#114) already; nothing new opened
by this report beyond T3's verification sample (in flight).

## g) Questions I can NOT figure out myself

None new. Open question 2 (promotion-threshold ratification) now has its
document: `docs/planning/2026-11-07_promotion-policy.md` §8 — the owner box
remains unchecked, and under decision D1 nothing promotes until it is.
