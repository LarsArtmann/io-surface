# io-surface — status v3: the superb-verification line executed (plan 20-06, T1–T25)

**Date:** 2026-11-07 (wrap of
`docs/planning/2026-11-07_20-06_io-surface-superb-verification-pareto-plan.md`)
**Scope:** the whole superb plan — measured precision, measured recall,
suppression, published promotion policy, protection machinery — plus two
real bugs the work itself surfaced.

## a) The one-paragraph summary

All 25 L1 tasks of the superb plan executed and verified. Precision is now a
published table (FP protocol over 73 hand-read fleet sites), recall is a CI
test (`TestMutationRecall`, 10 mutators, 100% pinned per rule), severity has
a numbered owner-ratified-pending policy, suppression is a reason-mandatory
comment directive, the fleet evidence matches the shipped ruleset twice over
(pre- and post-refinement), and the io-in-loop refinement cut fleet noise by
103 hits (−29%) without moving any other rule. Two bonus finds: dnsblockd's
week-old "2 opaque constraint violations" were a phantom-decode bug in
io-surface itself (now fixed with tests), and one new gate-debt harvest (the
manifest version refusal).

## b) Shipped (verified how named per claim)

| Deliverable                                                     | Where                                                         | Verified how                                                                           |
| --------------------------------------------------------------- | ------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Promotion policy v1 (numbers + de-promotion + ratification box) | `docs/planning/2026-11-07_promotion-policy.md`                | docs-gates; ledger §9 carries measured rows                                            |
| Fleet evidence ×2 (10-rule + refined)                           | `docs/evidence/fleetscan-2026-11-07-{v031,t16-refined}.json`  | `./scripts/fleetscan.sh <out>` — 160/160 both runs                                     |
| FP table v2 (30+30+13 hand-verified)                            | 21-12 evidence report §c2                                     | per-site file+context reads; classes listed                                            |
| Suppression `//io-surface:ignore <rule> <reason>`               | `internal/extract/suppress.go`                                | `go test ./internal/extract -run TestSuppress -v` (3 tests), goldens                   |
| Mutation recall corpus                                          | `internal/extract/mutation_test.go`                           | `go test ./internal/extract -run TestMutationRecall -v` — 10/10 classes at pinned 100% |
| Per-site validation errors + phantom-violation fix              | `internal/manifest/manifest.go` + first direct manifest tests | dnsblockd `check` now loads clean; planted bad-discipline names the CUE path           |
| io-in-loop same-path refinement                                 | `capture.loopRef/loopSeen/perItem` + rule                     | fleet 351→248; repeat/distinct fixture pair; recall corpus green                       |
| `lint --format github`                                          | `printGitHubAnnotations`                                      | e2e `TestExecute_LintGitHubFormat`; exit contract untouched                            |
| Contract polish                                                 | describe lint flags; SARIF `metadata.ruleDescription`         | contract test; regenerated SARIF golden                                                |
| Rule cards (11 rules, FP classes, decided semantics)            | README "Detector rule cards"                                  | docs-gates; help text links them                                                       |
| CHANGELOG Unreleased fill                                       | CHANGELOG.md                                                  | claims v1.2 + modulation + claim-conflict + wna + tiny-buffer + suppression listed     |
| Docs sync (T24, all four homes)                                 | AGENTS/ROADMAP/README/detectors                               | docs-gates 5/5                                                                         |

## c) The two bonus finds

1. **dnsblockd phantom violations (real bug, fixed):** `violationEntries`
   decoded `list.Concat` operand sub-lists into contentless violation
   structs — dnsblockd's gate has been red since 2026-11-06 on artifacts,
   not violations. Fix: content-guard on recovered entries + CUE paths on
   every reported violation. dnsblockd's real inventory is 2 undeclared
   sites + 10 extraction-blind infos (runbook prepared, T19).
2. **Gate-debt harvest:** the 04-27 report's rows 29–36 re-verified against
   HEAD — six struck as shipped-with-evidence, one genuinely open row
   harvested to TODO_LIST #116 (manifest unknown-major version refusal).

## d) Measured numbers (all commands in the evidence report)

- Fleet: 160/160 repos; findings-bearing 72 → 91 (10-rule) → 88 (refined).
- Per rule (refined): wna 340, io-in-loop 248, sto 50, cnt 13, err 11, ctx 3,
  dil 1, tiny-buffer 0.
- FP rates: wna 16.7%, sto 56.7%, cnt 76.9% — **nothing promotes**; ledger
  updated (policy W2 < 5% unmet everywhere).
- Recall: 10/10 mutation classes caught, 100% pinned per rule in CI.
- Benchmark: 239 ms median on the fleet's biggest tree (209k LOC).

## e) Owner decisions still open (unchanged + two new)

1. Promotion-policy numbers ratification (policy §8 checkbox) — nothing
   promotes until checked (D1).
2. dnsblockd v1→v1.2 migration mode (carried).
3. InboxClean CI pin bump + SARIF step (carried; bundle prepared at
   `InboxClean/docs/architecture/io-surface/PILOT-REVIEW-2026-11-07.md`).
4. NEW: ownership claims v1.3 scope (ROADMAP Q6 ADR).
5. NEW: the pilot batch on InboxClean (`a8be23dc`) is daemon-committed but
   unreviewed — the review bundle is the 5-minute pass.

## f) Up to 50 things to get done next

Routed: #116 (version refusal), #117 (ratifications), #115 (applyDoc seam
test, harvested from the other session's report). Nothing else new — the
next natural line is the FP-class carve-outs for write-non-atomic
(5 mechanical classes listed in the rule card), then a fresh FP sample under
the policy's cadence rule.

## g) Questions I can NOT figure out myself

None new beyond §e (they are all owner judgment calls by design — D1/D6).

## h) Verified how (whole-session battery)

After every L1 task: `gofmt -l .` (empty) → `go build ./...` → `go vet ./...`
→ `go test ./...` (0 FAIL) → `go run ./cmd/io-surface check --root .`
(0/0/0) → `render --doc docs/architecture/io-surface/INVENTORY.md --check`
→ `docs-gates run` (5/5). Facts refreshed via
`go run ./cmd/io-surface init --root . --force` after every line-moving task.
