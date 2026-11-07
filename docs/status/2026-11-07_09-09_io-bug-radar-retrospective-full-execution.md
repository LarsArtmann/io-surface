# 2026-11-07 09-09 — io-bug-radar full execution: honest retrospective (all 25 tasks)

**Session:** Crush, 2026-11-07 ~05:10→09:05 CET. Plan executed end-to-end:
`docs/planning/2026-11-07_05-02_io-bug-radar-pareto-plan.md` (T1–T25).
**HEAD:** `5b4432b` (master, pushed to Forgejo + GitHub). **Release:** v0.3.0
(tagged both remotes, GitHub Release published, `go list -m @v0.3.0` serves).
**One unstaged diff pending:** `internal/manifest/manifest_test.go` — a
formatter rewrote the map-copy loop into `maps.Copy` AFTER my commit
(behavior-identical; folds into the next commit).

---

## a) FULLY DONE

| Work                                                                                                                                                             | Evidence (command that proves it)                                                       |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| T1 design pass — signatures fixed into plan §5.1                                                                                                                 | plan file; every later task built exactly on it                                         |
| T2 observation plumbing — `Observation`, `ExtractWithObservations`, capture tracker (shapes/blanks/loops/ctx), D1 site-identity pinned                           | `go test ./internal/extract/` (`TestExtractWithObservations_SitesIdentical`)            |
| T3 `err-dropped` + D8 deliberate-ignore                                                                                                                          | `TestDetectErrDropped_ShapeTable`                                                       |
| T4 detector fixture tree + text/SARIF goldens + scaffold round-trip lint assert                                                                                  | `go test ./cmd/io-surface -run TestGolden`; `internal/scaffold/scaffold_test.go`        |
| T5 `lint` face — cmdguard command, text/json/sarif, exit-0 policy                                                                                                | `TestExecute_LintFace`; `io-surface lint --root testdata/detectors`                     |
| T6/T7 `closer-missing` (+resp.Body variant) with ownership classifier                                                                                            | `TestDetectCloserMissing_OwnershipTable`, `TestDetectCloserMissing_HttpBody`            |
| T8/T9 `ctx-dropped` + fixtures/goldens                                                                                                                           | `TestDetectCtxDropped_ScopeGate`                                                        |
| T10 dogfood on self — 6 hits verified, real N+1 in `symbolVisible` FIXED                                                                                         | `io-surface lint --root .` (post-fix) + `go test ./internal/check/`                     |
| T11 pilot FP sampling on 5 repos                                                                                                                                 | FP table in `docs/status/2026-11-07_08-08_io-bug-radar-p0-p2-detectors-fleet-proven.md` |
| T12–T15 `defer-in-loop`, `client-no-timeout`, `stat-then-open`, `io-in-loop`                                                                                     | `TestDetectLoopAndTimeoutRules`                                                         |
| T16 fleet harness committed (`scripts/fleetscan.sh`) + 160/160-repo run                                                                                          | `./scripts/fleetscan.sh /tmp/fleetscan-final.json`                                      |
| T17 severity verdicts — nothing promoted (D4 bar unmet), verdict table committed                                                                                 | status report §T17                                                                      |
| T18 **v0.3.0 release**: CHANGELOG, README, version const, full gates, annotated tag BOTH remotes, proxy check, GitHub Release                                    | `go list -m github.com/larsartmann/io-surface@v0.3.0` → serves; release URL live        |
| T19 claims v1.2 (`criticality`/`durability`), schema copies byte-identical, defaults materialize                                                                 | `TestLoad_V12ClosedSchema`; `cmp` of the two schema.cue                                 |
| T20 severity modulation (critical escalates; ephemeral drops write findings) + `claim-conflict` gate rule                                                        | `TestRun_ClaimSeverityModulation`, `TestClaimConflict`                                  |
| T21 `write-non-atomic` (seam exemption) + T22 `tiny-buffer` (inLoop gate)                                                                                        | `TestDetectWriteNonAtomic`, `TestDetectTinyBuffer`                                      |
| T23 pilots: InboxClean (v1.2 migration, criticality planted, CI pin `@v0.3.0`, gate green) + gogenfilter (fresh scaffold + CI wired, gate green via live module) | `cd InboxClean && io-surface check` → 0 errors; same for gogenfilter                    |
| T24 composition/SARIF ADR → ROADMAP Theme 5; TODO #61 decided (artifact-only until promotion)                                                                    | ROADMAP §Theme 5                                                                        |
| T25 docs sync — TODO_LIST #65–#89 + #61 DONE with evidence, FEATURES synced, AGENTS.md architecture updated                                                      | `docs-gates run` → 5/5 PASS                                                             |
| **2 real bugs found & fixed:** io-surface `symbolVisible` N+1; InboxClean `papersync/pdf.go` `outFile` fd leak                                                   | both diffs committed; InboxClean papersync tests pass                                   |

**Fleet evidence (final rule set minus the two P3 rules):** 160/160 repos, 72
with findings — io-in-loop 349, stat-then-open 50, client-no-timeout 13,
err-dropped 11, ctx-dropped 3, defer-in-loop 1.

## b) PARTIALLY DONE

1. **Fleet numbers are stale for the P3 rules.** `write-non-atomic` and
   `tiny-buffer` were added AFTER the v0.3.0 release; the fleet scan never re-ran
   with them. The released evidence report does not cite their fleet counts.
2. **FP sample depth.** Plan asked 3×30 hand-verified findings; I verified
   ~13 in depth + classified all 27 err-dropped fleet-wide. Every rule class
   was sampled; the volume was not.
3. **T17.1 promotion THRESHOLD never formalized.** "FP≈0" was never given a
   number (<1%? <5%?) or a per-rule vs whole-suite policy. The verdict table
   exists; the policy does not.
4. **gogenfilter pilot = scaffold only.** Green-by-construction, but zero
   claims/criticality ported (plan T23 asked for hand-ported claims).
5. **InboxClean CI bump not observed in real CI.** The pin `@v0.1.0→@v0.3.0`
   is committed-to-working-tree only (daemon); no GitHub Actions run watched.
6. **dnsblockd diagnosis shallow.** Confirmed v1-era manifest + 2 constraint
   violations + family conflicts on facts refresh; did NOT enumerate which
   sites fail (the tool's opaque validation message — TODO #3 — blocked this).
7. **T18.8 BuildFlow pickup note is thin.** The provider half (BuildFlow's
   `sdk_imports.go` blank import) needs its own go.mod bump to `@v0.3.0`;
   recorded in prose but not as an actionable checklist item.
8. **InboxClean post-refresh advisory noise:** 24 infos (21 new token-tier
   sites + drifts) left undeclared-by-design; not triaged per-site.
9. **`tiny-buffer` semantics for persistent wrappers:** `bufio.NewReaderSize`
   outside a loop is silently exempt (uniform inLoop gate) — a defensible but
   debatable design; only documented in a test comment, not a rule card.

## c) NOT STARTED

1. `lint` findings suppression mechanism (per-site nolint / lint-baseline —
   the gate has baseline; the lint face has nothing).
2. Self-lint in CI (`io-surface lint --root .` as a CI step — T10 dogfood was
   manual only; nothing prevents a future promoted rule from surprising CI).
3. CHANGELOG entry for the P3 features (claims v1.2, modulation,
   claim-conflict, write-non-atomic, tiny-buffer) — Unreleased still says
   "Nothing yet" while HEAD carries them. The next release cut would start
   from a wrong baseline.
4. dnsblockd v1→v1.2 migration (documented as owner-call; untouched).
5. io-in-loop same-path refinement (loop-body × printer-keyed expression) —
   the 0/3-actionable FP class has a fix shape, not built.
6. err-dropped os.Remove-cleanup class decision (suppress-story vs accept).
7. ROADMAP Theme 2 cross-link for `write-non-atomic` (planned T25 cross-link
   covered Themes 3/5/Q3, not Theme 2 explicitly).
8. README interaction note: `--enforce` + `criticality: critical` turns info
   findings into build failures — escalation is documented, its enforce
   interaction is not.

## d) TOTALLY FUCKED UP (all caught before commit, but the process churned)

1. **`capture.node` never set** in the first tracker draft → SIGSEGV in the
   first err-dropped test run. Root cause: writing the struct field list and
   the initializer separately.
2. **The flagship shape initially didn't fire:** `x, _ := os.ReadFile(p)` was
   classified `expression` because the assign check demanded
   `len(Lhs)==len(Rhs)` — multi-result calls always have `len(Rhs)==1`. The
   one case the rule exists for was silently dead until the fixture test
   caught it.
3. **First observations.go draft shipped a `currentFset` package-global hack**
   and a phantom `strconvItoa`/`relPath` — caught by build, rewritten in full.
4. **Duplicate `detectCtxDropped`** — a `cat >>` append silently succeeded on
   retry, doubling the function and registry entry; caught by build error.
5. **`runDetectors` re-processed ALL prior captures per function** →
   duplicate observations; fixed with a per-fn slice boundary.
6. **Shadowing bug in my own fix:** `symbolVisible` memoization used
   `src, err :=` inside, leaving the outer `src` nil — broke
   `TestRun_ClassifiesMissingDeclaredSites` until fixed.
7. **`openFileCreates` called `evidenceOf(nil, …)`** — would panic on
   position lookup; caught by reading, fixed to take the extractor.
8. **fleetscan.sh needed three tries:** a `IO SURFACE BIN` typo + unfinished
   python block in the first draft; then `pma` key casing (`name`/`path` vs
   the portfolio runner's normalized keys); then Go nil-slice `findings: null`
   breaking the python loop.
9. **Pre-commit file-size gate tripped twice** (main_test.go 711/650;
   detectors.go 676/650) — I never checked `.buildflow.yml`'s 650 pin before
   growing files; both were fixed by splits AFTER the hook failed.
10. **Count-assertion churn:** finding totals were hand-maintained via sed
    (3→5→7→11→28→34) across three test files; one sed round missed a file and
    had to be re-fixed. Fragile; golden-derived counts would have been safer.

## e) WHAT WE SHOULD IMPROVE

1. **Stop hand-maintaining finding totals in tests** — derive counts from the
   golden files or a fixture manifest of expected findings.
2. **Check `.buildflow.yml` file-size pin (650) BEFORE writing new files**;
   write rule files ≤300 lines from the start.
3. **Commit (or stash) before long daemon-exposed stretches** — the
   auto-commit daemon's touch/mod-time flapping cost many edit-tool retries.
4. **Tool error messages for validation failures** (TODO #3): "2 constraint
   violations: - ." is unusable; per-site CUE paths would have made the
   dnsblockd diagnosis a five-minute task.
5. **A suppression story for lint findings** (see c1) — without it, every FP
   class is a permanent nag.
6. **Print AST evidence through a cleaner formatter** — `matches, _ :=
   filepath.Glob( filepath.Join(root, pattern), )` keeps raw spacing; collapse
   before the paren list.
7. **Re-run the fleet scan as the LAST step of any rule addition** — evidence
   must match the shipped rule set, not an earlier one.
8. **Write the promotion-threshold policy as a committed document** with a
   number, not a phrase.
9. **Fixture-first development for detectors**: write the fixture row and the
   negative row BEFORE the detector; the multi-assign bug would have died at
   birth.
10. **Verify pilot CI by watching the actual workflow run**, not just local
    gate parity.

## f) NEXT (up to 50, ranked by impact/effort)

**Evidence hygiene**

1. Re-run `scripts/fleetscan.sh` with the final rule set; publish
   write-non-atomic/tiny-buffer fleet counts into the status report.
2. Add write-non-atomic/tiny-buffer/claim-conflict rows to the fleet
   verification sample; hand-verify 10 of each.
3. Copy fleet JSON evidence out of /tmp into `docs/evidence/` (or note
   ephemeral everywhere it is cited — TODO_LIST header cites /tmp too).
4. Formalize the promotion threshold: commit `docs/planning/…promotion-policy.md`
   with a number (e.g. "warning requires ≥30 verified findings AND FP<5% AND
   self-gate green under --enforce").
5. CHANGELOG: move the P3 features from nothing into Unreleased.
6. GitHub Release for v0.3.0: append the P3 delta or plan a v0.3.1 cut.
7. BuildFlow pickup checklist item: bump BuildFlow `sdk_imports.go` to
   `@v0.3.0` + provider parity note (T18.8 made real).
8. Watch InboxClean CI run on the pin bump; record the green link.
9. Push gogenfilter + InboxClean branches explicitly (daemon may have
   committed; verify remotes).

**Detector quality**
10. lint suppression: per-site ignore comments (`//io-surface:ignore <rule> reason`)
or a `lint-baseline.cue` mirroring the gate baseline (only-shrinks).
11. io-in-loop same-path refinement: flag only when the SAME printer-keyed
path/target expression repeats in the loop body.
12. tiny-buffer: detect the `f.Read(buf)` method shape on
known-closable receivers (os.File) as a resolved-adjacent carve-out, or
document the coverage limit in the rule card.
13. tiny-buffer: decide the persistent-wrapper semantics (ReaderSize without
loop) and write the rule card either way.
14. err-dropped os.Remove-cleanup class: add the suppression story or accept
explicitly in the rule card.
15. ctx-dropped force-kill class: document or suppress via the same mechanism.
16. err-dropped: un-flag blank-slot `http.NewRequest` in `_test.go` paths
(test-infra carve-out) — measure first.
17. stat-then-open: add the cache-probe FP class to the rule card (documented
in the status report, not in the rule metadata).
18. write-non-atomic: measure fleet noise; if huge, gate on durability claims
only (durable → flag, others silent) — the claims vocabulary exists now.
19. Observation.Calls: consider emitting the resolved consumer set for
symbol-level messages (better SARIF UX).
20. Rule cards in README: one table row per rule with FP classes and examples
(currently prose-only for err-dropped).

**Adoption**
21. dnsblockd v1→v1.2 migration: per-site discipline→exception mapping WITH
the owner (needs judgment context per write).
22. Fix the opaque manifest-validation message (TODO #3) — unblocks 21.
23. Declare or baseline InboxClean's 21 new token-tier advisory sites.
24. Port claims/criticality for gogenfilter's 5 sites (finish the pilot).
25. Wire `lint` into InboxClean/gogenfilter CI as artifact-only SARIF step
(per Theme 5 policy) to build upload evidence.
26. Self-lint CI step for io-surface (`lint --root .` — report-only until a
rule promotes).
27. Wire `lint` into the fleet's BuildFlow DAG as an opt-in tool.
28. Onboard 2 more pilots from the hotspot list (go-taskqueue has 37 findings
to triage — biggest signal source left).
29. Pilot retro #2: measure whether criticality claims change triage behavior.

**Schema/contract**
30. `claims v1.3` candidate: per-site `ownership` (who closes: self/caller/
helper) — would kill the closer-missing hand-off FP class properly.
31. Render: lint-findings appendix option (`render --lint`) — one doc, both
faces.
32. `lint --baseline` flag reading the gate's baseline.cue to suppress
canonically-baselined sites in lint output.
33. Decide `#aspect` IDs for detector observations (a site spanning two
bug-classes currently double-counts in per-rule aggregates).
34. SARIF: emit `properties.ruleDescription` from `DetectorRules()`
descriptions (currently absent from results).

**Tooling/tests**
35. Derive test count assertions from goldens (kill the sed churn).
36. Add a `testdata/detectors` parse-only CI guard (gofmt it too — currently
gofmt -l . covers it, but no test asserts parseability).
37. Parity test: errResults/closableCtors/etc. entries must exist in
callFamily (dead-table guard like `TestPkgOf_CoversEveryWatchedPackage`).
38. Fuzz the capture tracker on the fixture corpus (parent-map stack
discipline under FuncLits in loop bodies).
39. Benchmark `lint` on the biggest fleet repo; publish a number (display-only
per non-goals).
40. `--format github` output (PR annotation syntax) — cheap, high adoption.
41. Complete TODO #64: `internal/check/check.go` pairing split (still >650?).
42. Delete or archive `portfolio-io-insights.html` from repo root (stale
session artifact).
43. Unit tests for `scripts/fleetscan.sh` aggregation logic (python-in-bash is
untested; extract to a committed python file with a test).

**Docs**
44. README: document the `--enforce` × `criticality: critical` interaction.
45. README: rule table with FP classes (see 20).
46. AGENTS.md: add `lint` to the Commands block.
47. ROADMAP Theme 2 cross-link for write-non-atomic (planned-but-skipped).
48. Status report: link the fleet JSON location decision (see 3).
49. Describe contract: lint already registered; verify `describe --output
    json` includes lint flags (e2e asserts path only).
50. CHANGELOG: retro-fix the v0.3.0 section to mention `--enforce` semantics
precisely (it says rules never fail; modulation can escalate under
--enforce — say so).

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **dnsblockd migration:** do you want the v1→v1.2 judgment migration done
   WITH you (each of the ~131 claims' discipline→exception mapping needs your
   "why this write is safe" context), or should I auto-migrate mechanically
   (claims preserved verbatim, constraints dropped, drift documented) and let
   you review the diff?
2. **Promotion bar:** what FP rate and sample size actually qualify a rule for
   `warning`? (My T17 said "FP≈0" with zero verified-true samples for several
   rules — I need your number and whether promotion is per-rule or whole-suite.)
3. **Pilot repos commit policy:** the InboxClean (4 files incl. a real bug
   fix + CI pin) and gogenfilter (manifest + CI) changes are in their working
   trees — should I explicitly commit AND push those repos now, or do you
   review first? (Their daemons may have already committed some.)

---

**Where to look:** evidence report
`docs/status/2026-11-07_08-08_io-bug-radar-p0-p2-detectors-fleet-proven.md`;
plan (with §5.1 final design)
`docs/planning/2026-11-07_05-02_io-bug-radar-pareto-plan.md`; ADR ROADMAP
Theme 5; pilot diffs in the two pilot repos' working trees.
