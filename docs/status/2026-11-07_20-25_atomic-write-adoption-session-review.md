# Session Review — go-atomic-write adoption + dogfooding status (2026-11-07, ~19:45–20:25)

**Scope.** This review covers ONLY this session's two threads: (1) the
dogfooding status question (report at `2026-11-07_19-52_dogfooding-status.md`,
push executed on order), and (2) the go-atomic-write adoption ("do it!").
Verified with: full gate battery (`gofmt -l .` clean, `go build/vet/test
-count=1` all ok), `check --root .` → 16/16, 0 findings, `lint --root .` →
4 infos (was 11), `render --doc … --check` → fresh, CI-filtered facts diff →
quiet, `gh run list` → CI green on the pushed commit (42403657881, 1m39s).
NOT verified: the new concurrent-modification error branch has no dedicated
test (§b3); the other session's in-flight suppression work
(`internal/extract/suppress_test.go` etc.) — visible in the tree, untouched,
not assessed; Forgejo-side CI (runs nothing).

## Context: the storm (read §d first if confused)

While I edited, THREE writers were active on this repo: the auto-commit
daemon, a `buildflow --fix` loop (pid 1481793, later a second one), and a
second agent session (SUPERB plan, v0.4.0 release, portfolio HTML). Buildflow
reverted my committed edits in the working tree mid-flight; the daemon then
committed the revert (`36fd6e7`), producing a torn HEAD for ~30 minutes. I
waited out both buildflow pids, re-applied, and repaired the fallout
(TODO_LIST conflict markers). Net data loss: zero.

## a) FULLY DONE

| Item | Verified how |
| --- | --- |
| Dogfooding status report, every claim named its command | `docs/status/2026-11-07_19-52_dogfooding-status.md` |
| Push to both remotes on explicit order; sync proven | `5b4432b..0d78ec5`, `rev-list --count` = 0 both sides; CI green (run 42403657881) |
| go-atomic-write adopted as direct dep (v0.6.0) | `go.mod` require without `// indirect`; already in the binary via cmdguard→samber-do-auditlog before |
| All 5 write sites swapped: `WriteWithPerm` (scaffold, capabilities.cue, render --out), `WriteVerified`+fingerprint (baseline append, doc embed) | `commands.go:243,293,301,337`; `baseline_write.go:62`; `scaffold.go:78` |
| Extractor taught the library (resolved tier): pkgOf + callFamily (`Write*`→file-write, `FingerprintFile`→file-read) + errResults (slot 0; WriteIfChanged slot 1) | `internal/extract/tables.go`; fixture `testdata/extract/internal/store/atomicwrite.go`; cases in extract_test + check fixture |
| Self-dogfood improved: lint 11 → 4 infos (all 5 write-non-atomic + stat-then-open gone) | `lint --root .` output before/after this session |
| Concurrent-modification UX: baseline append / doc embed now fail with a re-run hint instead of clobbering | `errors.Is(err, ErrConcurrentModification)` branch, `baseline_write.go` |
| Storm recovery: diagnosed daemon/buildflow/second-session interplay, waited out pids, re-applied, zero data loss | reflog, `git show 36fd6e7 --stat`, pgrep output |
| TODO_LIST conflict-marker repair (buildflow stash fallout): kept upstream side incl. the other session's new SUPERB section, dropped the duplicated stale table | `rg '^(<<<<<<<\|>>>>>>>)' TODO_LIST.md` → no matches |
| Docs: AGENTS dep posture 3→4 deps; AGENTS sharp edge "buildflow --fix can revert uncommitted edits"; CHANGELOG [Unreleased] Added ×2; TODO #92 → PARTIAL with 6-of-11 accounted | file diffs, absorbed by daemon (`7adb441`, `1fbcd66`) |
| Facts + INVENTORY fresh; CI-facts-fresh filter quiet | `init --force`; `git diff -I… ` quiet; `render --doc --check` exit 0 |

## b) PARTIALLY DONE

1. **Concurrent-write error branch is untested.** The `ErrConcurrentModification`
   handling in AppendBaseline and applyDoc has no test (the fingerprint is
   captured inside the function, so a race isn't injectable without a small
   refactor). Happy paths fully covered by existing suites.
2. **TODO #92 triage is PARTIAL by design:** 6 of 11 infos answered by the
   adoption; the 4 io-in-loop residuals are accepted as report-only
   (T15/T17 verdicts) but NOT formally baselined or suppressed — the
   other session's suppression mechanism (#94/#105) may own that.
3. **CI verification of the push closed late:** I started the background
   watch, moved on, and only read its result at session end (green). The
   thread was open for ~30 min.
4. **INVENTORY freshness was caught by my final battery, not by my first
   one:** I ran check+lint+init after the swaps but forgot CI's
   `render --doc --check` step until the end. CI would have gone red.
5. **go.mod promotion ordering:** after the revert I ran `go get` BEFORE the
   imports existed, so the dep stayed `// indirect`; caught by my own final
   grep, fixed by `go mod tidy`. A mechanical error I should not make twice.
6. **Noticed but not escalated:** the SUPERB section reuses row numbers
   #90–114 while the detector section above uses #65–93 — a numbering
   collision that will confuse future HARVEST passes. Left for the owning
   session.

## c) NOT STARTED

- Upstream BuildFlow report: fix-loop reverting committed work + the
  stash it leaves (conflict markers in TODO_LIST). Needs the
  verify-before-filing ritual and a reproducible minimal case; today's
  incident had three concurrent writers, so attribution is muddy.
- Pushing the post-adoption commits (HEAD moved again via daemon; last
  human push was 0d78ec5).
- Re-running `lint --format sarif` + `--enforce` on the final state (row 91's
  CI step cites "11 findings"; the count is now 4 — worth confirming no
  golden/count assertion breaks).
- Propagating the pattern: go-atomic-write as the named fix hint inside the
  `write-non-atomic` finding message (today the advice is generic
  temp+rename prose).
- lessons.md candidate (cross-project): "multi-writer repo: `pgrep buildflow`
  before edit batches" — belongs in crush-config's references, by commit.

## d) TOTALLY FUCKED UP

1. **I edited a live multi-writer repo without checking who else was
   writing.** `pgrep -af buildflow` would have shown the loop 4:40 into its
   15-minute budget. Cost: one full revert/debug/re-apply cycle, a torn
   commit in history (`36fd6e7` commits a revert; the follow-up commits
   re-apply — confusing archaeology), a stray stash, and conflict markers
   in TODO_LIST. The AGENTS concurrency sharp edge existed; I had even
   cited it for dnsblockd earlier THE SAME SESSION. This is the session's
   real failure.
2. **"Transient" misdiagnosis.** When the extract test failed with 14 sites,
   I re-ran, saw green, and told the user it was transient — without a
   mechanism. The truth was the storm mutating files mid-battery; the same
   signature (site counts differing between runs) came back in check_test.
   A pass without an explanation is not a pass.
3. **`rg -rn` hook block — third strike of that exact trap on this host**
   (the hook's own words). Sloppy repetition of a known foot-gun.
4. **Forgotten step in my own battery.** I had read ci.yml line-by-line that
   morning (doc-freshness step included) and still shipped a post-change
   battery without `render --doc --check`. My final grep caught the indirect
   dep the same way — the pattern is: I assemble batteries ad hoc instead of
   running CI's exact step list.

## e) WHAT WE SHOULD IMPROVE

1. **Mirror CI, don't improvise:** after any code change run the exact
   ci.yml `test` job sequence (build → vet → test → gofmt → check →
   facts-fresh → self-render → doc-fresh → SARIF). One script, no memory.
   (Candidate: a `scripts/self-gate.sh`.)
2. **Pre-flight for write-storms:** `pgrep -af buildflow` + `git status
   --porcelain` BEFORE any multi-file batch in a daemon repo; wait out live
   fix loops; verify edits on disk immediately after applying.
3. **Ordering discipline for dep ops:** imports first, then `go get`, then
   `go mod tidy`, then verify the require line lost its `// indirect`.
4. **No verdict without mechanism:** if a test result changes between runs,
   stop and explain the delta before re-running to green.
5. **Close every background job or report it dead** — the CI watch sat
   unanswered for half the session.
6. **Content-anchored edits on daemon-active files;** sed line-ranges are a
   race against concurrent insertions (mine happened to land clean).
7. **Fix-should-name-the-fix:** detectors that flag a pattern whose cure is a
   first-party library should say so in the finding text (write-non-atomic →
   go-atomic-write). Turns findings into one-line fixes for the fleet.
8. **Flag collisions when you see them** (row numbering) in the final
   message, not in private notes.
9. **New error branches get tests or get named untested** in the docs.

## f) Up to 50 things to get done next (session-derived; most are ROADMAP fuel)

**Direct follow-ups (this week):**
1. Push the current HEAD (daemon commits since 0d78ec5 incl. adoption) — user order or standing rule (§g).
2. Watch one CI run on a HEAD that contains the adoption + row 91's self-lint step together; record the link.
3. Confirm row 91's CI lint step on disk matches the current 4-info state (no stale count assertions).
4. Test the `ErrConcurrentModification` branch: refactor fingerprint capture behind a small seam (read-func injectable) and add a deterministic race test (AppendBaseline first — file is small).
5. Same seam test for applyDoc.
6. Detector hint upgrade: `write-non-atomic` message names go-atomic-write as the sanctioned fix; regen lint goldens (`go test ./cmd/io-surface -run TestGolden -update`).
7. SUPERB T5 suppression mechanism (other session, in flight) — when it lands, re-evaluate #92's 4 io-in-loop residuals for formal suppression with reasons.
8. Renumber SUPERB section rows to #94+ to kill the collision with detector rows 65–93.
9. `scripts/self-gate.sh` mirroring ci.yml's test job (improvement #1).
10. docs/status/README.md index: add today's three reports (19-52 dogfooding, this review, the other session's).
11. HARVEST this report's §f into TODO_LIST/ROADMAP per the status-report loop (do not entomb).
12. Drop or keep the redundant `buildflow-pre-commit` stash — owner call; I verified it holds nothing HEAD lacks (v0.4.0 bump already in).

**Adoption ripple (fleet):**
13. README: short "atomic writes" guidance pointing adopters at go-atomic-write alongside the write-non-atomic rule.
14. InboxClean + gogenfilter: sweep their write-non-atomic hits with the same swap (pilot ripple, owner-gated per D6).
15. Provider description already names write-non-atomic; confirm no pin bump needed (provider unchanged — verify, don't assume).
16. FEATURES.md family-coverage row: add "third-party seams are teachable via tables (go-atomic-write landed)".
17. AGENTS testing-patterns note: third-party package teaching ritual (tables + fixture + both test layers) now has a worked example — reference it.
18. err-dropped fixture case for `atomicwrite.WriteIfChanged` (slot-1 errResults entry currently has no dedicated fixture).
19. Consider `WriteIfChanged` for `facts.cue` refresh — rejected for now (audit commit line changes every run, so it would never skip); document the why in scaffold.go.
20. errcheck warnings in commands.go (8 × fmt.Fprint unchecked) — pre-existing, in a file I touched: either fix with explicit ignores or record the handler convention as intentional in AGENTS.

**Detector depth (SUPERB-adjacent, from today's residuals):**
21. `symbolVisible` residual: teach the loop detector loop-carried memoization, or suppress with reason once T5 lands (`internal/check/location.go:28`).
22. `internal/extract/tables.go:96,106` stat pairs: batch the fixture-existence checks (one ReadDir instead of two Stats).
23. `scaffold.Write:76` existence guard: derive from a single ReadDir of the manifest dir instead of per-file Stat.
24. `io-in-loop` same-path refinement (SUPERB #105) — cross-link today's three concrete self-hits as test cases.
25. Recall harness (SUPERB #99/#100): add today's real fixed bugs (symbolVisible N+1, the 5 write sites) as mutation-corpus entries — bugs we actually had are the best recall probes.

**Repo hygiene:**
26. lesson → crush-config `references/lessons.md` (by commit): "pgrep buildflow before edit batches in daemon repos".
27. TODO_LIST numbering convention decision: per-section namespaces vs global monotonic (owner call).
28. Verify the second buildflow loop's owner (other session?) is done before any next batch edit.
29. `git stash list` audit across fleet repos — buildflow leaves stashes; sweep quietly.
30. facts.cue audit lines: consider whether `init --force` during a multi-writer storm should refuse to write when HEAD moved mid-write (tiny race today, tolerated by CI filters).
31. INVENTORY.md regen belongs in my personal battery script (item 9 covers).
32. CI: consider caching `go run` builds for the self-gate steps (runs 1m27–1m39; the four `go run` invocations recompile).
33. Post-adoption CHANGELOG: move entries into the next version heading at release time (v0.4.1 owner call).
34. Tag/release ritual if owner wants the adoption shipped as v0.4.1 (version const, CHANGELOG cut, both remotes, proxy check).
35. Confirm pkg.go.dev/proxy sees v0.4.0 (other session's release) — one `go list -m` check.

**Carried from the dogfooding report (still open, unchanged):**
36. InboxClean: commit the 4 pilot files (owner-policy question, still unanswered).
37. gogenfilter: push + watch CI; claims pass for 5 sites.
38. dnsblockd v1→v1.2 migration (owner-gated).
39. Pilot CI observation loop (retro §f8/§f10).
40. Next pilot cohort (go-taskqueue 37 findings).
41. Forgejo push mirror (TODO #41, blocked on token).
42. Forgejo visibility flip (TODO #42, blocked on token).
43. Capslock phantom-edge upstream report (blocked per verify-before-filing).
44. cmdguard-lint CI wiring (TODO #63, blocked upstream).
45. TODO #93 SARIF enforcement property.
46. TODO #64 file-size split continuation (check.go pairing).
47. Baseline canonical-key hiding (documented by-design; revisit if a real collision appears).
48. `--precise` tier decision (ROADMAP Theme 3) — T24 ADR chose standalone+SARIF; token-tier stays advisory (§g-Q3 stance).
49. ROADMAP Q5 capability vocabulary growth (schema bump process) — untouched, as designed.
50. Status-report cadence: next dogfooding re-check once the suppression mechanism lands (it changes the triage math for #92).

## g) Questions I cannot figure out myself

1. **The push ritual, as a standing rule:** today I pushed once on your
   explicit order; HEAD has already moved again (daemon commits with the
   adoption + docs). Do you want "push origin+github at end of session when
   the full battery is green" as a default I follow, or is pushing always
   your call?
2. **Baseline vs. acceptance for the 4 io-in-loop residuals:** accept them as
   report-only noise until the suppression mechanism lands (my current
   choice), or write formal `baseline.cue` entries / claims now so the
   triage is machine-visible? (The 4 are: `location.go:28` post-fix
   residual, `tables.go:96,106` stat pairs, `scaffold.go:76` existence guard.)
3. **BuildFlow upstream:** do you want the fix-loop behavior (reverting
   committed work under concurrent writers + leftover stash with conflict
   markers) verified and filed upstream, or do you already know it (local
   config / known issue)? I will not re-attempt without a clean repro per
   the verify-before-filing gate.

_Arte in Aeternum_
