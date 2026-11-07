# io-surface Promotion Policy v1 (PROPOSED — awaits owner ratification)

**Created:** 2026-11-07 (superb-verification plan T1)
**Status:** 🔶 PROPOSED — no promotion has happened under this policy yet, and
none may until the owner ratifies the numbers (plan decision D1; open question 2
of the detector plan). Ratification = the owner edits the **Ratification**
section below with a name+date, or amends the numbers in place.
**Applies to:** detector rules emitted by `io-surface lint` (all currently
`info`), not to manifest gate rules (`internal/check`) — those are already
error-severity by manifest contract.

---

## 1. Why a policy

Severity is currently a judgment call ("FP≈0"), which is a vibe, not a bar.
This policy turns promotion into an evidence transaction: a rule earns
`warning` or `error` by **published numbers**, and loses severity the same way.
The numbers below are proposed by the superb-verification plan; the owner owns
their final values.

## 2. Vocabulary

- **Verified finding** — a lint hit where a human read the file + context and
  confirmed the flagged code exhibits the rule's bug class (the rule is right
  that something worth looking at is there). Documentation and code examples
  (`testdata/`, docs snippets) do not count.
- **FP (false positive)** — a lint hit where the flagged code is correct as
  written; firing is noise for that site.
- **Evidence run** — one fleet scan (`scripts/fleetscan.sh`) plus the manual
  verification sample taken from it, recorded in `docs/evidence/` and the
  current evidence status report.
- **Sample size N** — verified findings + FPs hand-checked for that rule in the
  evidence run. The rule's **FP rate** = FPs / N.

## 3. Promotion to `warning`

A rule promotes `info` → `warning` when **all** hold, measured on its most
recent evidence run:

| #  | Gate         | Threshold                                               | Rationale                                                                  |
| -- | ------------ | ------------------------------------------------------- | -------------------------------------------------------------------------- |
| W1 | Volume       | ≥ 30 verified findings fleet-wide                       | below this, the rate estimate is too noisy to trust                        |
| W2 | Precision    | FP rate < 5% over the sample                            | warning is the "fix this soon" tier; noise budget is small but nonzero     |
| W3 | Self-gate    | `io-surface lint --root . --enforce` green on this repo | the rule must not fail the tool's own tree before it may nag anyone else's |
| W4 | Sample floor | N ≥ 30 for the rule (verified + FP combined)            | 30 is the minimum for a 5% rate to mean anything                           |

## 4. Promotion to `error`

A rule promotes `warning` → `error` (or `info` → `error` for an exceptional
rule backed by claims) when **all** hold:

| #  | Gate              | Threshold                                                                                                                                             | Rationale                                                                                       |
| -- | ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| E1 | Precision         | FP rate < 1% over the sample (0 FPs in N ≥ 100 preferred; N ≥ 30 with 0 FPs acceptable at first promotion)                                            | error fails builds; the bar is near-perfection                                                  |
| E2 | Claims backing    | the rule's bug class is declared in at least one pilot repo's `claims.cue` with `criticality: "critical"` (or the owner grants an explicit exception) | error severity means "this class is unacceptable here"; the claims system is how a repo says so |
| E3 | Self-gate         | W3 holds (self-gate green under `--enforce`)                                                                                                          |                                                                                                 |
| E4 | Real-bug evidence | ≥ 1 confirmed real production bug found by this rule (existence proof that the class bites)                                                           | e.g. closer-missing: InboxClean pdf.go fd leak                                                  |

## 5. De-promotion (the path back down)

Severity is rented, not owned. A rule demotes one tier when any of:

- **FP regression:** in any later evidence run with N ≥ 30, its FP rate exceeds
  its tier's ceiling (warning: 5%, error: 1%).
- **Trust failure:** the owner or a pilot user reports an FP class the rule
  cards do not document, and the class is confirmed by a sample.
- **Mechanism:** demotion is a one-line severity change in the rule metadata +
  CHANGELOG entry + rule card note; it does **not** require removing the rule.
- **Who decides:** demotion is automatic-on-evidence (the next evidence run
  that violates the ceiling triggers it in the same session); the owner can
  demote any rule at any time without evidence (judgment outranks numbers).

## 6. Per-rule vs whole-suite

**Decision: per-rule.** Each rule promotes on its own evidence. Rationale: the
suite's rules have wildly different FP profiles (fleet scan: err-dropped 11
hits vs io-in-loop 349); a whole-suite gate would let the worst rule hold the
best hostage and let a strong rule ride on weak neighbors. The suite as a
whole has exactly one shared gate: W3/E3 (self-gate green under `--enforce`).

## 7. Cadence

- Every rule-changing task re-runs the fleet scan (plan D4) — that evidence
  run is the input for any promotion decision made that session.
- A standing re-verification: before any promotion, the rule's FP classes are
  re-sampled (N ≥ 30) from the **current** ruleset, not inherited from older
  runs.

## 8. Ratification

**Owner decision required (open question 2, carried since 2026-11-07):** do the
thresholds above stand as written?

- [ ] Ratified as-is — `Lars Artmann, 2026-__-__`
- [ ] Amended — owner edits the tables in §3/§4 directly and initials here.

Until one of those boxes is checked: **no rule promotes**, regardless of what
the evidence says. Promotion tasks may _prepare_ (gather evidence, fill the
table in §9), but the severity flip is owner-gated.

## 9. Standing promotion ledger

| Rule              | Current | Evidence run            | N   | FPs                  | FP rate          | Verdict under this policy    |
| ----------------- | ------- | ----------------------- | --- | -------------------- | ---------------- | ---------------------------- |
| err-dropped       | info    | 2026-11-07 fleet        | 11  | ~2 (post-carve)      | ~18%*            | stays info (W1 volume unmet) |
| closer-missing    | info    | 2026-11-07 pilots       | 2   | 0                    | 0% (N too small) | stays info (W1/W4 unmet)     |
| ctx-dropped       | info    | 2026-11-07 fleet        | 3   | 3 (force-kill class) | 100%*            | stays info (W1/W2 unmet)     |
| defer-in-loop     | info    | 2026-11-07 fleet        | 1   | 0                    | 0% (N too small) | stays info (W1/W4 unmet)     |
| client-no-timeout | info    | 2026-11-07 fleet+triage | 13+ | TBD T3/T12           | TBD              | decided by T3/T12 evidence   |
| stat-then-open    | info    | 2026-11-07 fleet        | 50  | TBD T3/T12           | TBD              | decided by T3/T12 evidence   |
| io-in-loop        | info    | 2026-11-07 fleet        | 349 | 3/3 sampled FP       | TBD              | refinement first (T16)       |
| write-non-atomic  | info    | none (postdates scan)   | —   | —                    | —                | T2/T3 evidence pending       |
| tiny-buffer       | info    | none (postdates scan)   | —   | —                    | —                | T2/T3 evidence pending       |

\* small-N rates are estimates; the ledger keeps them only for trend
visibility, never for a decision on their own (W4 blocks that).

The ledger is updated by every evidence run. It is the single table the owner
reads to see whether anything _could_ promote.
