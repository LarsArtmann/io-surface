# SUPERB Pareto execution: Waves 1-3 shipped (P01-P12), Wave-3 tail and Wave 4 pending

**Session:** 2026-11-15, report written 12:56 CET.
**Scope:** full execution of the owner-approved plan
`docs/planning/2026-11-15_08-57_SUPERB-io-surface-pareto-execution-plan.md`
(P01-P26 over ALL 32 then-open TODO rows). This report covers the run so
far; a concurrent session landed `41ecbd3` (row #139 done, #27 unblocked)
mid-run. **Final state at write time:** branch `master`, 18 commits ahead
of `origin/master` (NOT pushed — push only on your word), working tree
clean, all gates green (`go build/vet/test`, `gofmt -l .` empty,
`check --root .` 0 findings, docs-gates 5/5).
**Ledger:** 171 rows — 146 DONE, 19 open TODO, 6 BLOCKED
(`grep -c` over `TODO_LIST.md`). 13 rows closed by this session.

Every claim below names the command or commit that verifies it.

## a) FULLY DONE

| #  | What                                                                                     | Evidence (command / commit)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| -- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **P01 living-docs VERIFY** — 16 drift claims fixed across README/FEATURES/ROADMAP/AGENTS | commit `7d9e97d` (daemon-carried); README gained the 5 undocumented gate rules (severities read from emit sites in `internal/check/check.go:41-55`) + 3 shipped-but-undocumented flags; FEATURES re-swept at v0.5.1 (SARIF ghost-PLANNED closed, baseline-add split brain deduped, TODO 3/7/8/19/20/21/28 stale citations corrected); ROADMAP shipped-ideas moved out; CHANGELOG verified const==head (`rg version cmd/io-surface/main.go` → 0.5.1)                                                                                                                                                                                                                                            |
| 2  | **P02 TODO reconciliation**                                                              | commit `55a3f62`; row #162 closed with the 16-claim evidence list; row #163 re-scoped (golangci half already fixed by concurrent session — `rg golangci CONTRIBUTING.md` → 0 hits); duplicate-id audit re-run (only the labeled snapshot-table namespace 90-93 collides — intentional, left)                                                                                                                                                                                                                                                                                                                                                                                                   |
| 3  | **P03 `scripts/self-gate.sh`** — the whole ci.yml test job as one local command          | commit `579cc6a` + daemon `d47c395`; failure path probe-verified (planted location drift in `cmd/io-surface/commands.go` → exit 1 with WHAT/WHY/FIX + diff); green path exit 0 (`./scripts/self-gate.sh; echo $?` → 0); docs-gates pin read FROM ci.yml at runtime so the script cannot drift from CI                                                                                                                                                                                                                                                                                                                                                                                          |
| 4  | **P04 AGENTS.md slim** — 224 → 174 lines (budget 220)                                    | commit `a8889d8` (row edit) + daemon-carried rewrite; `wc -l AGENTS.md` → 174; `nix flake check` all passed (agents-md-valid); corrections made while re-reading: detection tables extract.go→tables.go, stale "no direct tests for manifest/render" gap dropped (tests shipped via TODO 10/12)                                                                                                                                                                                                                                                                                                                                                                                                |
| 5  | **P05 docs-gates pin automation**                                                        | commits `603d4a8` (daemon: ci.yml + renovate.json) + `f500fcb` (row #161); pin 2278406afdfd→**v0.3.1** (untagged rev → release tag); battery run with the EXACT CI invocation before bump: `go run github.com/Artmann-Technologies/docs-gates/cmd/docs-gates@v0.3.1 run` → 5/5 PASS; renovate.json validated by `npx --package=renovate renovate-config-validator` ("Config validated successfully"); regex extraction verified against ci.yml                                                                                                                                                                                                                                                 |
| 6  | **P06 detector trio** (#158/#159/#160)                                                   | commits `4815f3b`/`bb6f046` (daemon, source+goldens+tests) + `19923b8` (CHANGELOG/rows); #158: probe proved go-finding renders `uri:""` for empty paths (invalid SARIF) → location-less gate findings now anchor at the manifest file where the fix lives (facts.cue / baseline.cue per rule, `anchorFor` in `internal/emit/emit.go`), lint face keeps the tool pseudo-file (41/41 real anchors on the fixture); #159: `check.SortFindings` exported, lint applies it — text/JSON/SARIF order identical across both faces, goldens re-pinned deliberately and reviewed; #160: `extract.Severity` named type, 10 registry entries on constants, `TestDetectorRuleSeverities` walks the registry |
| 7  | **P07 `--strict-weak` cmd e2e**                                                          | commit `0b70495`; `cmd/io-surface/gate_flag_test.go` (new file, honors the 650-line gate): default advisory exit 0 + info severity, `--strict-weak` same-count escalation to error/exit 1, audit demotes the escalation (exit 0) with `--enforce` override (exit 1), unknown flag + malformed bool → exit 2; goldens byte-identical (`git diff --stat cmd/io-surface/testdata/` empty)                                                                                                                                                                                                                                                                                                         |
| 8  | **P08 `--json` vs `--format` decision** (owner: keep `--format`)                         | commit `37f21a0` + daemon-carried README/CHANGELOG; evidence: `scripts/fleetscan.sh` already passes `--format json` (zero fleet callers of the shorthand); README leads with `--format`, `--json` documented as kept shorthand; CHANGELOG Unreleased Changed entry                                                                                                                                                                                                                                                                                                                                                                                                                             |
| 9  | **P09 docs-pointer existence check**                                                     | commit `4094432`; `scripts/check-docs-pointers.sh` (prose pointers in AGENTS/README/CONTRIBUTING must exist; fenced EXAMPLE paths excluded after it caught 2 false positives — `docs/architecture/IO.md` is a consumer-repo example); wired into `self-gate.sh` + ci.yml; probe-verified both directions (green tree exit 0; planted fake reference exit 1 naming file+source); two archived retrospective items ("add lint to the Commands block") struck — the line has been there since the detector line shipped (`sed -n '28p' AGENTS.md`)                                                                                                                                                |
| 10 | **P10 result/ rebuild + policy**                                                         | daemon `0716ce6`; root cause: stale `result` pointed at an old treefmt-check derivation; `nix build .# -o result` exit 0 → now targets `io-surface-4094432` derivation; policy bullet in AGENTS.md Conventions                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| 11 | **P11 flake test check verdict**                                                         | commit `5475aff`; `nix build .#checks.x86_64-linux.test` → exit 0 (sandboxed go test over vendored modules); `nix flake check` all-passed same day                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| 12 | **P12 archived-report banner pass**                                                      | commit `2bcf516`; 4 banner-less candidates of 28 audited — 3 fully drained (07-59/08-21/14-25: every named remainder resolved — bufio mystery pinned by `TestPkgOf_CoversEveryWatchedPackage`, #26/#38 DONE, InboxClean leak fixed there) → no banner; 21-30 GOT the banner (its routed owner/blocked cluster #40-42/#63/#50/#49/#61 is genuinely open, banner names it + points at the SUPERB packets)                                                                                                                                                                                                                                                                                        |
| 13 | **Audit regressions harvested the honest way (twice)**                                   | my AGENTS slim + row edits changed living-line matches → audit-all red twice; both times resolved by docs-health (2 enforcement-policy items struck as SHIPPED with facts.cue/manifest.go:268 evidence; go-structure-linter → new row #166; portfolio html → new row #167; 2 lint-command items struck) — commits `f451d6f` + `4094432`; battery back to 5/5 each time (`nix run github:Artmann-Technologies/docs-gates -- run .`)                                                                                                                                                                                                                                                             |

## b) PARTIALLY DONE

1. **P13 BuildFlow provider pin v0.4.0 → v0.5.1 — bump applied, commit BLOCKED.**
   `cd ~/projects/BuildFlow && go get github.com/larsartmann/io-surface@v0.5.1`
   → go.mod line 189 now `v0.5.1 // indirect`; main module builds; tools
   module builds; full providers suite PASSES against the new pin
   (`cd tools && go test ./providers/...` → ok 92s). Direction confirmed:
   BuildFlow blank-imports io-surface's provider; no io-surface-side change.
   **Blocker:** the BuildFlow pre-commit hook reddens on
   `TestVendorHashTripwire` with `fatal: Unable to create
   '.git/next-index-<pid>.lock.lock': File exists` — the three live
   `buildflow --fix` sessions hold BuildFlow's git index locks
   (`pgrep -af buildflow` → 3 processes since session start). TWO commit
   attempts failed on the same environmental collision (retried once after
   an 8s window — locks returned). The bump sits UNCOMMITTED in BuildFlow's
   working tree. I did NOT use `--no-verify` (hook bypass needs your call).
2. **P17 stale-binary/root-cause class — evidence gathered, RCA not written.**
   Today's BuildFlow pre-commit lock collisions are fresh evidence for the
   same "environmental gate red" class as the recorded 2026-11-11
   `go-mod-parity` false-red (AGENTS sharp edge). The refresh recipe
   (`nix build .# && nix run .#reinstall`) is already in AGENTS; what P17
   still owes: reproduce with `buildflow doctor`, sweep other fleet repos,
   and decide the docs-gates-binary freshness check idea.

## c) NOT STARTED

1. **P14** render `--lint` appendix DESIGN (P15's prerequisite).
2. **P15** render `--lint` appendix IMPLEMENT + tests + golden.
3. **P16** W3 promotion re-scan (`./scripts/fleetscan.sh` → docs/evidence/,
   re-derive the FP table vs the D4 bar).
4. **P18** go-taskqueue fan-out count RCA (53 vs 52 via `-s io-surface`).
5. **P19** go-taskqueue claims judgment hour (~230 blocks).
6. **P20** dnsblockd triage packet — scope SHRANK: the concurrent session
   closed #139 (`41ecbd3`), so the packet consolidates #27 + #146 (+ the
   MIGRATION-v1.2 §5 warnings) instead of the original trio.
7. **P21** M20 capability promotion evidence summary → ROADMAP Q5 input.
8. **P22** schema v1.3 DESIGN SPIKE (ownership closer-missing + runtime-power
   capabilities; design doc only — bump stays owner-gated).
9. **P23** mobile 320px render audit of the docs site index.html.
10. **P24** upstream repro packages (#50 Capslock phantom NETWORK edges, #133
    go-finding run-level propertyBag) — verify-before-filing + owner gate.
11. **P25** fleet unblock prep packets (#40 token packet, #41 mirror runbook,
    #42 visibility note, #155 Forgejo-CI port plan) — execution owner-gated.
12. **P26** confirm trio (#142 InboxClean gate state, #147 old GitHub
    Releases retention, #148 billing blast-radius audit note).
13. **Push:** 18 local commits on `master` (both remotes pending your word).

## d) TOTALLY FUCKED UP (honest list, ranked by severity)

1. **I attempted cross-repo commits without checking that repo's lock state
   first.** The sharp edge ("a live buildflow loop can revert/mid-air
   collide; re-read before editing when one is running,
   `pgrep -af buildflow`") is in MY OWN AGENTS.md for THIS repo, and I
   applied it here — but walked into BuildFlow and committed blind. Two
   failed commits, ~30s of hook budget burned twice, and the P13 result
   (a fully verified pin bump) is stranded uncommitted. Forgot, not
   unknown.
2. **A multiedit orphaned a function body.** My check.go edit replaced the
   `sortFindings` doc+signature and swallowed the `func` line, leaving
   `sort.SliceStable` bodyless — broken build, caught immediately, fixed.
   Whole-function replacements should go through `lsp_replace_symbol`
   (exact-boundary by design); I used text multiedit for a symbol-shaped
   change.
3. **Two python heredoc replacements misfired on multiline import patterns**
   (emit_test.go: the `extract")`-style anchors never matched the actual
   file layout). Each burned a cycle before the exact-text edit tool
   landed it. Lesson applied: for import blocks, use the edit tool with
   the literal block, never pattern surgery.
4. **The CHANGELOG edit dance:** my restructure edit accidentally DELETED
   the Changed section I had just added (old_string contained it,
   new_string dropped it); restored two edits later. One multiedit with
   the full section list would have been one step.
5. **`rg -rn` hook trap, again** (the host hook blocked it: `-r` is
   replace-display, not recursion). Third strike of this exact trap on
   this host per the hook's own message. It's `rg -n`; internalize it.
6. **My own slims caused the two audit-all regressions.** The compressions
   were right, but I didn't re-run the battery BETWEEN the AGENTS rewrite
   and the row edits — the red surfaced only at the next battery run,
   and diagnosis cost a loop both times. Batteries belong after every
   living-line batch, not just at task end.
7. **P12's plan premise half-inverted on evidence** (3 of 4 "missing
   banner" candidates had no remainder at all). Not a fuckup — the row
   said "only banner the ones missing it" and deeper honesty won — but
   the plan over-scheduled a mechanical pass that turned into a
   judgment pass. Plan sizing vs. reality: noted for the next plan.

## e) WHAT WE SHOULD IMPROVE

1. **Cross-repo pre-flight ritual:** before ANY commit in a sibling repo,
   check `git status`, `pgrep -af buildflow`, and `.git/*.lock` there —
   codify it in the fleet AGENTS guidance, not just io-surface's.
2. **Symbol-shaped edits via LSP tools** (`lsp_replace_symbol`), text edits
   only for prose/config. Tonight's orphaned-function slip is the second
   data point for this rule.
3. **Battery after every living-docs batch** (not per task): docs-gates
   `run .` costs <1s and catches living-line regressions while the
   causing edit is still in context.
4. **Quiet-window commits for hook-heavy repos** (BuildFlow's pre-commit
   runs a 60s-budget audit suite; committing while 3 `--fix` loops run
   wastes the budget and reddens environmental tests).
5. **Renovate activation is the missing half of P05** — the config is in,
   validated, and regex-verified, but the bot needs the GitHub App
   installed by you; until then the pin relies on self-gate.sh reading
   ci.yml (which at least makes local runs follow bumps automatically).
6. **Plan errata habit works** — the P12 inversion and the #139 rescope
   were absorbed without re-dispatching shipped work; keep writing the
   errata into the plan file as things drift.

## f) NEXT — up to 50 things (ordered: finish-the-wave first, then value)

1. **Commit the P13 pin in BuildFlow** (quiet-window retry, or your
   explicit leave-uncommitted decision — see §g-Q1).
2. **P14** render `--lint` design (flag shape, appendix placement, golden
   strategy) → record in row #153.
3. **P15** implement render `--lint` appendix + tests + golden + docs.
4. **P16** W3 re-scan: `./scripts/fleetscan.sh` → `docs/evidence/`, re-derive
   FP table vs D4 (FP≈0), record verdict in #151.
5. **P17** `buildflow doctor` RCA + fleet sweep + refresh the AGENTS recipe.
6. **P18** taskqueue 53-vs-52 fan-out delta.
7. **P19** go-taskqueue claims judgment (~230 blocks, 3 passes).
8. **P20** dnsblockd packet (#27 + #146 + §5 warnings; #139 done by
   concurrent session — verify their diff first).
9. **P21** M20 capability promotion evidence table → Q5 decision input.
10. **P22** schema v1.3 design doc (ownership + runtime-power; bump
    owner-gated).
11. **P23** 320px mobile audit of the docs site + small CSS fixes.
12. **P24** #50 Capslock minimal repro + #133 go-finding propertyBag drafts
    (verify-before-filing; owner gate before filing).
13. **P25** #40 token packet, #41 mirror runbook, #42 visibility note, #155
    Forgejo-CI port plan.
14. **P26 confirm trio rows:** #142 InboxClean gate state confirm
    (post-scope-shift), #147 old GitHub Releases retention check, #148
    billing blast-radius audit note.
15. **Push both remotes** when you say so (18 commits waiting).
16. TODO reconciliation pass 2 after Waves 3-4 land (renumber audit +
    battery).
17. Renovate App activation (owner) → first automated pin PR.
18. #167: confirm intent on `portfolio-io-insights.html`, then remove
    (owner staged the deletion once; a commit re-added it).
19. #166: `.go-structure-linter.yaml` re-evaluation (dated 2027-02-11;
    row exists so it cannot rot).
20. #63 re-check when cmdguard ships the replace-free lint module.
21. After P15: re-run self-gate to prove the new appendix survives the
    inventory-doc `--check` gate.
22. After P16: if any rule meets FP≈0, route the promotion decision to
    you (never self-promote).
23. After P19: go-taskqueue push (their repo's gate first).
24. After P25: everything in the fleet-unblock chain becomes 5-minute
    decisions the moment you drop the Forgejo token.
25. Wire `scripts/check-docs-pointers.sh` into the other fleet repos'
    gates (it caught 2 real false positives on day one; the pattern is
    fleet-worthy).
26. Consider exporting the self-gate pattern fleet-wide (the 2026-11-08
    and 11-11 CI-break classes were not io-surface-specific).
27. vulnix/govulncheck: re-run when nixpkgs/Go upstream ship fixes
    (GO-2026-6859 class — external, parked).
28. #40-#42/#155 execution the moment the token lands (P25's packets
    make it turnkey).
29. dnsblockd v1→v1.2 judgment migration (owner-scheduled 2026-11-08;
    sits behind the P20 packet decision).
30. go-taskqueue onboarding completion after P19 (advisory-first per Q1
    decision).
31. Consider a docs-gates binary-freshness check idea as a candidate row
    (P17.7 buffer item).
32. SARIF: if go-finding ships run-level propertyBag (#133), migrate the
    enforcement key there per emit.go's own comment.
33. Post-plan: annotate THIS plan file with its own errata (P12 inversion,
    #139 rescope, P13 blocker) so the next session isn't misled.
34. FEATURES: new rows for today's ships (self-gate.sh, pointer check,
    typed Severity, anchors) — the inventory should name them.
35. README: mention `scripts/check-docs-pointers.sh` in Development (it's
    in CONTRIBUTING + self-gate; README's stage list omits it).
36. CHANGELOG: cut v0.6.0 when P15/P16 land (the Changed section is
    growing; release protocol: `docs/release-protocol.md`).
37. After the fleet-unblock: retire the "push BOTH remotes" dance from
    AGENTS (mirror replaces it).
38. Baseline burn-down section in render (ROADMAP Theme 2 candidate;
    unchanged, still raw).
39. `io-surface diff` / `explain` (Theme 2; unchanged).
40. systemd hardening generator (Theme 2's highest-value consumer output;
    unchanged).
41. `--precise` tier (Theme 3 boundary; unchanged, deliberately separate).
42. os.Std* family decision (FEATURES family-coverage note names it as
    the one unwatched class; owner call whether console reads matter).
43. Mutation-recall corpus extension when new detectors land (P15's
    appendix is renderer-side; no new rules — n/a today, keep on radar).
44. Cross-check other siblings for the TestVendorHashTripwire
    lock-collision class (P18.6 buffer item).
45. InboxClean pin bump to the provider-v0.5.1 BuildFlow once P13 lands
    (their CI consumes BuildFlow, not io-surface directly — verify
    direction before touching).
46. Re-verify `nix flake check` after any flake-relevant change (cheap,
    already scripted in muscle memory).
47. Keep the docs-gates pin on release tags (v0.3.1 convention now set);
    bump deliberately with a local battery run.
48. When renovate opens its first PR: verify the regex manager matched
    (the managerFilePatterns anchor on workflow files).
49. Archive this report when its items drain (cadence ritual: banner +
    `git mv docs/status/archived/`).
50. A grand `scripts/self-gate.sh` green run as the final plan-closure
    artifact (exit 0 = every ci.yml stage + pointers + battery).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (3)

1. **P13 commit in BuildFlow:** the pin bump is verified and sitting
   uncommitted; the pre-commit's `TestVendorHashTripwire` collides with
   the git index locks of the concurrent `buildflow --fix` sessions.
   Options: (a) I keep retrying in quiet windows until it lands;
   (b) you authorize `--no-verify` for THIS one commit (the suite
   genuinely passed, the red is environmental); (c) leave it uncommitted
   for you. Which?
2. **The three `buildflow --fix` processes** (`pgrep -af buildflow`,
   running since before this session): are they yours and should I leave
   them alone, or are they stale leftovers I should stop as part of P17?
   Committing and gate-running in BuildFlow is unreliable while they
   hold locks.
3. **Push cadence:** 18 commits sit local on `master` (GitHub CI runs on
   push only, so CI has seen nothing of today's work). Push both remotes
   now, or push once after P14-P26 close the plan?

---

_Every claim names its verifying command or commit; numbers without one
are unverified and this report says so._
