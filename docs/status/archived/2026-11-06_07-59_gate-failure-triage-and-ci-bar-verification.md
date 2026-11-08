# Status Report — Gate-Failure Triage & CI-Bar Verification

**Date:** 2026-11-06 07:59 CET
**Session window:** ~07:45–08:00 CET (single conversation; one task)
**Author:** Crush session (glm-5.3-flash), triggered by a pasted `check` gate failure

## Scope line

**Verified here** (fresh commands, this session):

- `go run ./cmd/io-surface check --root .` → **exit 0**, twice (mid-session:
  `14 declared, 14 extracted (14 resolved, 0 token), 0 baselined — 0 errors,
  0 warnings, 0 infos`; final: same but `2 baselined`). The pasted failure
  (3 undeclared-site errors, 3 location-drift infos) does **not** reproduce.
- All 9 CI steps replicated locally **in ci.yml order**, all green:
  build → vet → test (10 packages ok) → `gofmt -l .` (empty) → self-check →
  facts-freshness (`init --force` + `git diff -I` filtered: "facts in sync") →
  self-render → INVENTORY freshness (`render --doc … --check`) → SARIF emit.
- `git log`: `410ce52` (auto-commit, 6 files — includes the facts.cue refresh)
  and `5a6c289` (`feat(check): shadow-capability rule names I/O reachability
  hides`) landed **during** this session.
- `git status` read at three points: start = only `M internal/check/check.go`;
  mid = `M ci.yml, CHANGELOG.md, facts.cue, internal/emit/emit_test.go`;
  end = `M AGENTS.md (mine), baseline.cue, check.go, check_test.go`. A second
  agent session was actively working the whole time.
- Read in full: `.github/workflows/ci.yml`, `facts.cue` (post-refresh),
  `internal/absorb/absorb.go` (through line 200), the `check.go` WIP diff,
  `internal/emit` package doc, `INVENTORY.md` head.

**NOT verified here** (and why):

- **Remote CI** — nothing was pushed during this session; GitHub CI state is
  unknown. Local replication ≠ a remote green run.
- **Whether `410ce52`/`5a6c289` are pushed** — I never ran `git fetch` /
  `ls-remote` (out of the triage path I chose).
- **SARIF content** — CI's bar is "emission path works"; I redirected to a
  temp file and did not parse rule/severity mapping for the new rules.
- **The other session's in-flight work** — `baseline.cue`, `check.go`,
  `check_test.go` were mid-edit at session end; their final state is unknown.
- **`claims.cue` contents** — grepped only; the 20-claims-vs-"Claimed 0"
  observation in INVENTORY is unexplained (see f-20).
- **My AGENTS.md edits' survival** — the other session was editing AGENTS.md
  too (it updated the check row to "14 rules"); I did not re-diff after.
- **gopls** — left in a stale state (see d-4); never restarted.

---

## Self-review answers (asked directly)

**What did you forget?**

1. **Routing discoveries into `TODO_LIST.md`.** The errcheck findings, the
   CONTRIBUTING/golangci drift, the dprint non-enforcement — I reported them
   in chat and now in this report, but never landed them in the repo's
   interactive backlog. Chat output is not a state (BuildFlow loop rule 7:
   fixed / filed / documented).
2. **`lsp_restart`.** gopls served stale compiler errors for the rest of the
   session; I worked around it instead of clearing it.
3. **Remote/push state.** AGENTS.md says every publish pushes BOTH remotes;
   I never even checked whether the tree was ahead of origin.
4. **Post-edit collision check.** I edited AGENTS.md while another session
   owned in-flight edits to it; I did not re-verify my three edits afterwards.

**What could you have done better?**

1. **Ground-truth first.** I grepped manifest internals and built a false
   "INVENTORY contradicts facts.cue" narrative around a single read of a file
   that was actively being rewritten. Running `check` (30 seconds) before
   investigating internals would have shown green immediately and saved two
   full investigation rounds.
2. **Verify before asserting.** I stated "the WIP doesn't compile" as fact
   from gopls diagnostics in one message, then corrected it the next after
   `go build` passed. The compiler is the fact source; the LSP is a hypothesis.
3. **First grep used a nonexistent path** (`cmd/io-surface/facts.cue` — there
   is no facts file under `cmd/`). Sloppy command construction that fed the
   confusion in (2).

**What could you still improve?**

1. **Concurrency awareness as default.** This repo runs multiple agent
   sessions + an auto-commit daemon. Any file read is potentially stale; any
   pasted output is potentially already fixed. This should be a checked habit,
   not a discovery (I added the sharp-edge bullet to AGENTS.md — that is
   documentation, not yet a protocol).
2. **Close the loop mechanically** — every finding ends in code, an issue, or
   a tracked TODO in the same session, not in prose.
3. **Distinguish "my verification churn" from "repo state"** — my
   `init --force` verification run bumped the audit line in facts.cue and left
   it for the daemon. Harmless by design (CI filters those lines), but a
   read-only verify mode would eliminate the class (see f-23).

---

## a) FULLY DONE

1. **Root-caused the pasted gate failure.** The 3 undeclared sites
   (`runAbsorb`, `absorb.ModulePath`, `absorb.gitHead`) + 3 location drifts
   were a stale `facts.cue` (11 sites) vs. the tree's absorb feature (14
   sites). Evidence: pre-refresh grep vs. post-refresh `facts.cue` read;
   resolution landed in `410ce52`; `check` now exits 0.
2. **Verified the full CI quality bar locally, in CI order** — 9 steps, all
   green (commands listed in the scope line). This is the strongest claim of
   the session.
3. **Prevented a false-positive "fix".** The uncommitted `shadowCapabilities`
   WIP in `check.go` looked broken (stale gopls: `undefined: capability` ×4);
   `go build` proved it compiles (the import was present at check.go:32).
   Hands stayed off another session's in-flight work.
4. **AGENTS.md memory updated** (3 edits, multiedit-confirmed, `M AGENTS.md`
   in final git status): corrected CI step list (was missing
   facts-freshness/inventory-fresh/SARIF emit), added architecture rows for
   `internal/absorb`, `internal/capability`, `internal/emit`, added the
   concurrent-sessions sharp-edge bullet.
5. **Final gate confirmation post-edits:** exit 0 (14 declared, 14 extracted,
   2 baselined — the other session's baseline additions absorbed cleanly).

## b) PARTIALLY DONE

1. **AGENTS.md architecture accuracy** — table rows added, but the data-flow
   intro (§Architecture, lines 85–88) still omits the absorb
   (Capslock→capabilities.cue) and emit (findings→SARIF) legs, and the header
   claim "three direct dependencies" was never re-verified against today's
   go.mod. Remaining: S effort.
2. **Discovery routing** — findings live in chat + this report; TODO_LIST.md
   not yet updated (HARVEST pending, see c-1). Blocker: none, just not done.
3. **Concurrent-session coordination** — detected, characterized (commits
   `410ce52`, `5a6c289` landed mid-session; in-flight work at end), documented
   in AGENTS.md — but no actual coordination protocol exists (who finishes,
   who verifies, who pushes). Blocker: needs the user's intent (g-1).
4. **Repo state at handoff** — gate green at my last run, but the tree was a
   moving target (`baseline.cue`, `check.go`, `check_test.go` mid-edit); the
   final CI bar after the other session finishes is unrun. Remaining: one
   full-bar run + push (f-45).

## c) NOT STARTED

1. **HARVEST** of this report's section (f) into `TODO_LIST.md`/`ROADMAP.md`
   (docs-health owns the mode; deliberately deferred — user said report then
   wait).
2. **Pushing both remotes** — `410ce52`, `5a6c289`, and the daemon's commits
   (incl. my AGENTS.md edit) are local-only (not started; push policy needs
   the user, g-2).
3. **Forgejo push mirror + visibility flip** — AGENTS.md documents both as
   blocked on a Forgejo API token not present on this machine.
4. **errcheck cleanup** — 5 unchecked `fmt.Fprintf`/`Fprint`/`Fprintln` in
   `cmd/io-surface/commands.go` (90, 114, 115, 117, 229). Not started; needs
   a policy decision first (fix vs nolint vs CI lint step, e-5).
5. **golangci-lint policy** — CONTRIBUTING mentions it; no `.golangci.yml`
   exists; CI doesn't run it. Decision pending.
6. **dprint enforcement** — `dprint.json` exists for non-Go files; CI ignores
   it. Decision pending.
7. **gopls hygiene** — stale diagnostics never cleared.

## d) TOTALLY FUCKED UP

Nothing I executed ended the session broken — but four things deserve the
strongest honest framing:

1. **Near-miss: almost edited another session's WIP on a false premise.**
   I declared the `check.go` WIP "broken" from stale gopls diagnostics in a
   session message before running the compiler. Had I proceeded, I would have
   been "fixing" (i.e., rewriting) a concurrent agent's finished work based on
   a lie from my own tooling. Caught within one step; the cost was one wrong
   assertion in-session. Severity: none (no damage); lesson: `go build` is the
   fact source.
2. **Two wasted investigation rounds on stale reads.** During active
   concurrency I treated a single grep result as stable ground truth and
   constructed a contradiction narrative (INVENTORY vs facts.cue) that never
   existed by the time I finished reasoning about it. Severity: wasted effort
   only. Root cause: ground-truth-last instead of ground-truth-first.
3. **Observed, not caused by me: the committed repo state at session start was
   internally inconsistent.** Committed `INVENTORY.md` showed 14 sites
   (incl. absorb) while committed `facts.cue` had 11; the pasted `check`
   output (3 errors) matches that state. CI would have been red on at least
   self-check and facts-freshness had it been pushed. Root cause: the
   auto-commit daemon commits feature code without running the gate its own
   CI demands. Mitigation: local full-bar verification (which the other
   session apparently did before `410ce52`); the structural gap remains
   (e-2, f-36).
4. **gopls served false compiler errors** (`undefined: capability` ×4 at
   check.go:412–424) while the package compiled cleanly. Severity: medium
   hazard — any agent (or human) trusting the LSP over the compiler here
   edits healthy code. Workaround: compiler-first verification; not restarted
   (c-7).

## e) WHAT WE SHOULD IMPROVE

1. **Ground-truth-first ordering**: run the gate (`check`, `go build`) before
   reading manifest internals or trusting any snapshot in a repo with
   concurrent writers. Impact: this session burned ~2 rounds on exactly this.
2. **Daemon gate integration**: the auto-commit daemon ships commit states the
   repo's own CI would reject (feature code without refreshed facts.cue).
   A pre-commit self-check (or a post-commit verify with revert-on-red) closes
   the gap mechanically. Impact: prevents red-CI surprises on push.
3. **LSP distrust policy**: treat gopls errors as hypotheses until `go build`
   rules; restart the LSP after external file changes. Impact: prevents
   false-positive "fixes" of healthy code.
4. **AGENTS.md single-writer discipline**: two sessions edited it this
   session; clobber risk on a load-bearing file. Suggested fix: announce
   intent (status note) before editing shared docs during concurrency, or
   serialize doc edits.
5. **Lint policy singularity**: one decision — either CI runs
   golangci-lint (then the 5 errcheck findings and any config belong in-repo)
   or CONTRIBUTING stops implying it. Impact: kills recurring errcheck noise
   in every tool result.
6. **SARIF emit step semantics**: `check --format sarif` exits 1 on findings,
   so a finding on a branch fails CI at the emit step before upload — likely
   surprising. Decide deliberate-vs-resilient and document.
7. **Read-only facts-freshness**: CI verifies freshness by mutating the tree
   (`init --force` + `git diff`). A verify mode (`init --check` /
   `check --facts-fresh`) would make verification side-effect-free and kill
   the audit-line churn class this session produced.
8. **Close-the-loop as mechanism**: findings get fixed/filed/tracked in the
   same session; chat prose doesn't count. This report + HARVEST is the
   documented path; use it every time.

## f) Next tasks (up to 50, ranked by impact)

Impact: Critical/High/Medium/Low · Effort: S <30min, M 30min–2h, L >2h ·
Category: Bug/Feature/Quality/Cleanup/Documentation/Process

| #  | Task                                                                                                                          | Impact   | Effort | Category      |
| -- | ----------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Run docs-health HARVEST: route section (f) below into TODO_LIST.md / ROADMAP.md                                               | Critical | S      | Process       |
| 2  | Push both remotes (`origin` + `github`) per AGENTS.md publish rule — 410ce52, 5a6c289, daemon commits are local-only          | Critical | S      | Process       |
| 3  | After the other session finishes: one final full CI-bar run on the settled tree, then push                                    | Critical | S      | Process       |
| 4  | Verify remote GitHub CI green after push (local replication ≠ remote run)                                                     | High     | S      | Process       |
| 5  | Investigate claims pairing: facts audit says `claims: 20` but INVENTORY rendered `Claimed 0` for all sites — reconcile        | High     | M      | Bug           |
| 6  | Review the 2 baseline entries added late-session: what they suppress, whether claim prose should own them instead             | High     | S      | Quality       |
| 7  | Verify README documents the `absorb` subcommand (flags, Capslock pinning, --check mode)                                       | High     | S      | Documentation |
| 8  | Verify CHANGELOG.md has entries for absorb + capabilities v1.1 + shadow-capability (file was mid-edit this session)           | High     | S      | Documentation |
| 9  | Confirm `cmd/io-surface/main_test.go` e2e-covers runAbsorb both modes (write + --check drift)                                 | High     | S      | Quality       |
| 10 | Confirm `check_test.go` covers shadowCapabilities (covered/uncovered paths, baseline suppression, unknown-name skip)          | High     | S      | Quality       |
| 11 | Verify TODO_LIST.md is current post-concurrent-session (it may already own several items below — de-duplicate at harvest)     | High     | S      | Documentation |
| 12 | Verify my 3 AGENTS.md edits survived the other session's concurrent AGENTS.md edits (re-diff now)                             | Medium   | S      | Documentation |
| 13 | Extend AGENTS.md data-flow line to the absorb (Capslock→capabilities.cue) and emit (findings→SARIF) legs                      | Medium   | S      | Documentation |
| 14 | Re-verify AGENTS.md header "three direct dependencies" against current go.mod                                                 | Medium   | S      | Documentation |
| 15 | Verify FEATURES.md inventories v1.1 capabilities, absorb command, shadow-capability rule                                      | Medium   | S      | Documentation |
| 16 | Decide Forgejo push mirror (Settings → Mirrors → GitHub) — blocked on Forgejo API token not on this machine                   | High     | S+tok  | Process       |
| 17 | Decide Forgejo repo visibility (public like gogenfilter, or keep private) — same token blocker                                | Low      | S+tok  | Process       |
| 18 | Fix or nolint the 5 errcheck findings in cmd/io-surface/commands.go:90,114,115,117,229 (after #19 decides the policy)         | Medium   | S      | Quality       |
| 19 | Decide golangci-lint policy: add `.golangci.yml` + CI step, or correct CONTRIBUTING.md to stop mentioning it                  | Medium   | M      | Process       |
| 20 | Decide dprint enforcement for non-Go files (config exists, CI ignores it)                                                     | Low      | S      | Process       |
| 21 | Verify `internal/emit` maps the new rules (shadow-capability) with correct severity/ruleId — add emit test if missing         | Medium   | S      | Quality       |
| 22 | Validate emitted SARIF content once (parse /tmp/io-surface-verify.sarif or regenerate; ruleIds, severities, positions)        | Medium   | S      | Quality       |
| 23 | Add a read-only facts-freshness mode (`init --check` or `check --facts-fresh`) so CI verifies without mutating the tree       | Medium   | M      | Feature       |
| 24 | Make `capabilities.cue` writes atomic (temp+rename) to avoid torn manifest dirs on crash mid-absorb                           | Medium   | S      | Quality       |
| 25 | De-duplicate the `absorb.Drift` double call in `runAbsorb` (compute once, reuse for lines + count)                            | Low      | S      | Cleanup       |
| 26 | Decide whether CI should run `absorb --check` against a pinned fixture (capability-drift advisory currently never runs in CI) | Medium   | M      | Process       |
| 27 | Add golden/round-trip test for `absorb.CUE` byte contract if absorb_test.go lacks one                                         | Medium   | M      | Quality       |
| 28 | Verify absorb tests are hermetic (no network/git needed in sandbox) — assert explicitly                                       | Medium   | S      | Quality       |
| 29 | Reconcile INVENTORY/claims/facts once all concurrent edits land; re-run full bar                                              | High     | S      | Quality       |
| 30 | Verify internal/manifest + internal/render direct-test gap is still tracked in TODO_LIST (AGENTS.md says it is)               | Medium   | S      | Quality       |
| 31 | Size and schedule the internal/manifest + internal/render direct tests (known gap; only scaffold-roundtrip covers them)       | Medium   | M      | Quality       |
| 32 | Confirm baseline canonical-key hiding + id-form dotted-ID sharp edges are owned in TODO_LIST (not just AGENTS.md prose)       | Low      | S      | Quality       |
| 33 | Decide git tagging/release flow for the module (version const exists in main.go; no release process documented here)          | Low      | M      | Process       |
| 34 | After push: verify `go install github.com/larsartmann/io-surface@latest` resolves via the proxy                               | Medium   | S      | Quality       |
| 35 | Document in CONTRIBUTING that Forgejo does not run CI (GitHub-only)                                                           | Low      | S      | Documentation |
| 36 | Evaluate a daemon-side gate hook: block (or auto-verify) commits that leave the self-check red                                | Medium   | M      | Process       |
| 37 | Establish a concurrency protocol for multi-agent sessions in this repo (announce, ground-truth-first, who closes out)         | Medium   | S      | Process       |
| 38 | Run a dependency sweep for cuelang/cmdguard/go-finding (buildflow update owns it)                                             | Low      | S      | Process       |
| 39 | Annotate stale status reports (docs-health ANNOTATE) once this one's items are resolved                                       | Low      | S      | Documentation |
| 40 | Verify SARIF artifact renders on GitHub (clean run = empty results list; still proves the path)                               | Low      | S      | Quality       |
| 41 | Check ROADMAP.md §g open questions (incl. deliberate no-flake.nix) still reflect reality                                      | Low      | S      | Documentation |
| 42 | Clear gopls stale state (restart) and note recurrence behavior if it re-stales mid-edit                                       | Low      | S      | Cleanup       |
| 43 | Consider `--out` flag for `check --format sarif` instead of stdout-redirect (only if it earns its keep — YAGNI check)         | Low      | S      | Feature       |
| 44 | Verify README quickstart still matches actual commands after v1.1 additions                                                   | Medium   | S      | Documentation |
| 45 | Post-concurrency sanity: confirm no session left uncommitted garbage outside their intended files (git status audit)          | Medium   | S      | Cleanup       |

Honesty note: items 1–14, 18, 21–25, 29, 31–32, 42, 45 come from direct
observation this session; 15–17, 19–20, 26–28, 30, 33–41, 43–44 are decisions
or verifications inferred from this session's reads (docs/CI/config state),
not new research — per the session constraint they were not re-verified today.

## g) Questions I cannot answer myself

1. **The concurrent session (coordination):** another agent session was
   mid-flight at 07:59 (editing `check.go`/`check_test.go`, `baseline.cue`,
   CI, CHANGELOG; it committed `410ce52`/`5a6c289` during my run). Is it
   still supposed to be running, and who owns the close-out (final full CI
   bar + push)? I stayed hands-off its files — is that the right division,
   and do you want my AGENTS.md/AGENTS-adjacent edits to wait for it?
2. **Push policy:** everything from this session window is local-only.
   AGENTS.md mandates `git push origin master && git push github master` per
   publish, but pushing mid-flight work from the _other_ session could
   publish its half-done state. Do you want me to push now, wait for that
   session to finish, or do you push manually?
3. **Forgejo token:** the push mirror and the public-visibility flip are
   documented in AGENTS.md as blocked on a Forgejo API token that is not on
   this machine. Can you provide it (or set up the mirror in the Forgejo UI
   yourself), or should per-push dual-push remain the standing procedure?

---

_Per status-report skill: section (f) is HARVEST input for TODO_LIST.md /
ROADMAP.md — item 1. Point-in-time snapshot; goes stale by design._
