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

| Rule              | 08-08 (8 rules) | This run (10 rules) | Delta reading                                                                                     |
| ----------------- | --------------- | ------------------- | ------------------------------------------------------------------------------------------------- |
| io-in-loop        | 349             | 351                 | stable (2-repo drift in the fleet itself; not a rule change)                                        |
| stat-then-open    | 50              | 50                  | unchanged                                                                                           |
| client-no-timeout | 13              | 13                  | unchanged                                                                                           |
| err-dropped       | 11              | 11                  | unchanged (carve-outs already in that run)                                                          |
| ctx-dropped       | 3               | 3                   | unchanged                                                                                           |
| defer-in-loop     | 1               | 1                   | unchanged                                                                                           |
| write-non-atomic  | — (not shipped) | **340**             | the single largest new signal pool; every hit is an un-attempted atomic-write adoption candidate   |
| tiny-buffer       | — (not shipped) | **0**               | the loop-gating (consumer must sit in a loop) makes it precise-and-rare; zero fleet noise           |

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
