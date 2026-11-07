# 2026-11-07 20-01 — io-bug-radar session retrospective v2: delta after the superb-bar strategy round

**Session:** Crush, 2026-11-07 ~05:10→20:00 CET. The detector plan
(`docs/planning/2026-11-07_05-02_io-bug-radar-pareto-plan.md`, T1–T25) was
executed, released (v0.3.0), and retrospectived earlier today — the full repo
snapshot lives in `docs/status/2026-11-07_09-09_io-bug-radar-retrospective-full-execution.md`.
**This report is the CURRENT snapshot:** everything from that retro still
stands, updated by (1) a concurrent session's dogfooding verification at
19:52 (`docs/status/2026-11-07_19-52_dogfooding-status.md`), (2) this
session's two strategy rounds after the retro (review guidance + the
"superb" framework — no code written in them), (3) daemon activity.

**HEAD:** `0d78ec5` (master; v0.3.0 tag at `336c279`, feature commit
`5b4432b`). **Live gates at 20:00:** self-check 0/0/0, `go test ./...` all
packages ok, gofmt clean, tree clean. **CI verified GREEN on the pushed HEAD
by the concurrent session** (`gh run list` → run 42353726369, success,
08:02Z — all 9 gate steps incl. facts-freshness and public-install tripwire).

## Delta since the 09:09 retrospective

| Change                                                                                       | Source                                    | Impact                                                                                    |
| -------------------------------------------------------------------------------------------- | ----------------------------------------- | ----------------------------------------------------------------------------------------- |
| Retrospective report committed                                                               | daemon (`1425f39`)                        | —                                                                                         |
| `facts.cue` refreshed (audit-line drift drained)                                             | daemon/`init --force` (`228159d`)         | CI facts-freshness stays green                                                            |
| Concurrent session's dogfooding report landed                                                | `0d78ec5`                                 | CI-green + pilot states now VERIFIED, not assumed                                         |
| gogenfilter pilot landed on master                                                           | daemon (`a6e78aa`) + dep bump (`ebdac85`) | pilot #2 fully shipped                                                                    |
| InboxClean: still 4 uncommitted pilot files (ci.yml pin, claims, schema, pdf.go fd-leak fix) | nobody — awaiting owner review            | the one real bug fix is still disk-only                                                   |
| This session: review-guidance + superb-framework answers                                     | strategy only                             | two NEW plan items identified: mutation corpus (recall %), formal promotion policy number |

## a) FULLY DONE (unchanged facts, re-verified)

Plan T1–T25 all done — full table in the 09:09 retro §a. Short form: 10
detector rules + lint face shipped behind `extract.ExtractWithObservations`
(Site JSON byte-identical, D1 test-pinned); v0.3.0 released, tagged both
remotes, GitHub Release live, `go list -m @v0.3.0` serves; fleet harness
(`scripts/fleetscan.sh`) ran 160/160 repos (72 with findings); claims v1.2
(`criticality`/`durability`) + severity modulation + `claim-conflict`;
pilots InboxClean (migration + criticality + **real fd-leak fix in
pdf.go**) and gogenfilter (now landed); ADR in ROADMAP Theme 5; TODO #61
decided; FEATURES/TODO_LIST/AGENTS synced; docs-gates 5/5 PASS; self-check
0 errors / 0 warnings / 0 infos; CI green on the pushed commit (now
verified, previously assumed).

**Two real bugs found and fixed by the suite:** io-surface's own
`symbolVisible` N+1 (fixed, committed) and InboxClean's `papersync/pdf.go`
`outFile` fd leak (fixed in that repo's tree — UNCOMMITTED, see b).

## b) PARTIALLY DONE

1. **InboxClean pilot is disk-only.** All four pilot files uncommitted
   (confirmed independently by the 19:52 dogfooding report and by
   `git -C InboxClean status` at 20:00). "Done" is true on disk, not in
   history. Owner review pending (question 3).
2. **Fleet evidence stale for the P3 rules.** `write-non-atomic` and
   `tiny-buffer` shipped after the v0.3.0 fleet scan; their fleet counts
   have never been measured. The superb-bar item "re-scan after every rule
   change" exists precisely because of this.
3. **FP sample depth.** ~13 findings verified in depth + all 27 err-dropped
   classified fleet-wide; the plan's 3×30 protocol was not run to volume.
4. **Promotion policy has no number.** T17 said "FP≈0" — never formalized
   (threshold, sample size, per-rule vs whole-suite). The T17 verdict table
   exists; the policy document does not.
5. **gogenfilter claims port.** Scaffold + CI landed; zero
   claims/criticality written (its 5 sites are unclaimed judgment).
6. **`tiny-buffer` persistent-wrapper semantics** (`bufio.NewReaderSize`
   outside a loop is silently exempt) — defensible, undocumented beyond a
   test comment.
7. **dnsblockd diagnosis shallow.** v1-era manifest, 2 opaque constraint
   violations, family conflicts on refresh; the tool's unusable validation
   message (TODO #3) blocked enumeration.
8. **T18.8 BuildFlow pickup note thin.** Provider half (BuildFlow
   `sdk_imports.go` blank import) needs its go.mod bumped to `@v0.3.0`;
   recorded in prose, not as an actionable item.
9. **InboxClean post-refresh noise:** ~21 new token-tier advisory sites +
   drifts undeclared-by-design; untriaged per-site.
10. **Recall never measured.** The suite's two real-bug finds are an
    existence proof, not a rate (see the superb framework: mutation corpus).

## c) NOT STARTED

1. **CHANGELOG Unreleased is still empty while HEAD carries P3 features**
   (claims v1.2, modulation, claim-conflict, write-non-atomic, tiny-buffer).
   The next release cut would start from a wrong baseline. (Daemon did not
   touch CHANGELOG either.)
2. Lint findings suppression mechanism (per-site ignore / lint-baseline with
   only-shrinks — the gate has baseline, the lint face has nothing).
3. Self-lint in CI (`io-surface lint --root .` as a CI step; T10 dogfood
   was manual).
4. **Mutation corpus + recall measurement** (NEW, from the superb
   discussion): inject the 8 bug classes into real fleet code, require the
   suite to catch them, publish recall % per rule.
5. dnsblockd v1→v1.2 judgment migration (owner call — question 1).
6. io-in-loop same-path refinement (0/3 sampled actionable — fix shape known,
   not built).
7. err-dropped os.Remove-cleanup class decision (accept or suppress-story).
8. ROADMAP Theme 2 cross-link for write-non-atomic.
9. README: `--enforce` × `criticality: critical` interaction doc (escalation
   documented; its build-failing consequence not).
10. Formal promotion-policy document (see b4).

## d) TOTALLY FUCKED UP (all caught pre-commit; carried from 09:09 retro §d)

1. `capture.node` never set → SIGSEGV on first err-dropped run.
2. **The flagship case was silently dead:** `x, _ := f()` classified
   `expression` because the assign check demanded `len(Lhs)==len(Rhs)` —
   multi-result calls always have one RHS expression. The fixture test
   caught it, not design review.
3. First observations.go draft had a `currentFset` global hack + phantom
   helpers; rewritten in full.
4. `detectCtxDropped` duplicated by a silent double-append; build caught it.
5. `runDetectors` re-processed ALL prior captures per function (duplicate
   observations); fixed with a per-fn boundary.
6. Shadowing bug in my own `symbolVisible` fix (inner `src, err :=` left the
   outer `src` nil) — the fix for a found bug was itself buggy until a test
   flagged it.
7. `openFileCreates(evidenceOf(nil, …))` — would panic; fixed to take the
   extractor.
8. fleetscan.sh needed three tries (typo + unfinished python; pma key casing
   `name`/`path` vs the portfolio runner's normalized keys; Go nil-slice
   `findings: null` breaking the python loop).
9. Pre-commit 650-line gate tripped twice (main_test.go 711, detectors.go
   676) — the `.buildflow.yml` pin was never checked before growing files.
10. Count assertions hand-maintained via sed across three files
    (3→5→7→11→28→34); one round missed a file.

## e) WHAT WE SHOULD IMPROVE

1. Fixture-first for every detector: positive AND negative rows before the
   rule code (bug #2 dies at birth).
2. Check the 650-line pin before writing new files; rule files ≤300 lines.
3. Derive finding-count assertions from goldens, not hand-maintained seds.
4. Commit early and logically — daemon mod-time flapping cost many edit-tool
   retries.
5. Fix the opaque manifest-validation message (TODO #3) — "2 violations: - ."
   made the dnsblockd diagnosis a dead end.
6. Re-run the fleet scan as the LAST step of any rule change (evidence must
   match the shipped ruleset).
7. Write policies as committed documents with numbers, not phrases.
8. Cleaner AST evidence printing (raw `format.Node` keeps ugly spacing).
9. Verify pilot changes by watching the real CI run, not local parity only
   (the 19:52 session did exactly this — make it the standard).
10. Adopt the superb loop as the standing definition: MEASURE (precision +
    recall) → FIX (evidence-driven carve-outs) → PROMOTE (numbered policy) →
    PROTECT (goldens, self-gate, suppression).

## f) NEXT (up to 50, merged + re-ranked)

**Evidence hygiene (the superb bar, first)**

1. Promotion policy document with a real number (threshold, sample size,
   per-rule vs suite) — unblocks every severity decision forever.
2. Re-run `scripts/fleetscan.sh` with the final ruleset; publish
   write-non-atomic/tiny-buffer fleet counts.
3. Mutation corpus: inject the 8 bug classes into real fleet code; recall %
   per rule, in CI (the superb missing half).
4. Full FP protocol: 30 verified findings × every rule; refresh the FP table.
5. Copy fleet JSON out of /tmp into `docs/evidence/` (TODO_LIST header cites
   /tmp paths too).
6. CHANGELOG: fill Unreleased with the P3 features.
7. BuildFlow pickup checklist: bump `sdk_imports.go` to `@v0.3.0` (T18.8
   made actionable).
8. Decide v0.3.1 (small cut: P3 delta + lint polish) vs folding into v0.4.
9. Verify InboxClean CI green after its pilot files land (the gogenfilter
   equivalent was observed; InboxClean's wasn't — its CI never ran the pin).
10. Archive `portfolio-io-insights.html` from repo root (stale artifact).

**Detector quality**
11. Lint suppression mechanism: per-site `//io-surface:ignore <rule> reason`
or `lint-baseline.cue` (only-shrinks like the gate's).
12. io-in-loop same-path refinement (printer-keyed target repetition).
13. tiny-buffer: decide persistent-wrapper semantics; write the rule card.
14. err-dropped os.Remove-cleanup class: accept-or-suppress decision.
15. ctx-dropped force-kill class: same decision.
16. write-non-atomic: measure fleet noise; consider durability-claims-only
gating if huge.
17. stat-then-open cache-probe FP class → into the rule card metadata.
18. Test-infra carve-out for `http.NewRequest` blanks in `_test.go` (measure
first).
19. Rule cards in README: one table row per rule with FP classes + examples.
20. Emit rule descriptions into SARIF `properties` from `DetectorRules()`.

**Adoption**
21. dnsblockd v1→v1.2 migration (needs owner judgment — question 1).
22. Fix the opaque validation message (TODO #3) — unblocks 21.
23. Land InboxClean's four pilot files (question 3), then triage its ~21
token-tier advisory sites (declare or baseline).
24. Port claims/criticality for gogenfilter's 5 sites (finish pilot #2).
25. `lint` SARIF artifact step in InboxClean/gogenfilter CI (Theme 5 policy:
artifact-only until promotion) — builds the upload evidence base.
26. Self-lint CI step for io-surface.
27. go-taskqueue pilot (37 findings — biggest untriaged signal pool).
28. Wire `lint` into the fleet BuildFlow DAG as an opt-in tool.
29. Pilot retro #2: did criticality claims change triage behavior?

**Schema/contract**
30. Claims v1.3 candidate: per-site `ownership` (self/caller/helper) — kills
the closer-missing hand-off FP class properly.
31. `render --lint` appendix option: one doc, both faces.
32. `lint --baseline` reusing the gate's baseline.cue canonically.
33. `#aspect` IDs for detector observations (sites with two bug-classes
double-count in per-rule aggregates).
34. `--format github` PR-annotation output (cheap, high adoption).
35. Benchmark `lint` on the biggest fleet repo (display-only per non-goals).

**Tooling/tests**
36. Derive count assertions from goldens (kill the sed churn).
37. Parse-only CI guard for `testdata/detectors` + gofmt it explicitly.
38. Dead-table guard: errResults/closableCtors/etc. entries must exist in
callFamily (mirror of `TestPkgOf_CoversEveryWatchedPackage`).
39. Fuzz the capture tracker on the fixture corpus (FuncLits inside loop
bodies).
40. TODO #64 remainder: split `internal/check/check.go` pairing.
41. Unit tests for fleetscan aggregation (extract the embedded python into a
committed, tested file).
42. `describe --output json`: assert lint's flags are in the contract (only
the path is asserted today).

**Docs**
43. README: `--enforce` × `criticality` interaction.
44. AGENTS.md Commands block: add `lint`.
45. ROADMAP Theme 2 cross-link for write-non-atomic.
46. CHANGELOG v0.3.0 section: state `--enforce` escalation semantics
precisely (it currently says rules never fail; modulation can escalate).
47. Superb bar one-pager (from the strategy answer) into docs/planning.
48. Link the fleet JSON location decision in both status reports (see 5).
49. Per-rule doc anchor IDs for SARIF consumers to deep-link.
50. Errors-that-help pass on lint's exit-0 policy note (users should not
have to read the flag help to learn findings never fail by default).

## g) QUESTIONS I CANNOT ANSWER MYSELF (carried — still open)

1. **dnsblockd migration mode:** with-you (per-claim judgment context) or
   mechanical (constraints dropped, claims preserved, drift documented for
   review)?
2. **Promotion bar:** what FP rate + sample size qualifies a rule for
   `warning` — and per-rule or whole-suite?
3. **Pilot commit policy:** commit+push InboxClean's four files (incl. the
   real pdf.go fd-leak fix) and gogenfilter (already landed via daemon)
   now, or after your review of the diffs?

---

**Where to look:** full session snapshot `…09-09_io-bug-radar-retrospective-full-execution.md`;
dogfooding verification `…19-52_dogfooding-status.md` (concurrent session);
evidence `…08-08_io-bug-radar-p0-p2-detectors-fleet-proven.md`; plan with
final design §5.1 `docs/planning/2026-11-07_05-02_io-bug-radar-pareto-plan.md`;
ADR ROADMAP Theme 5.
