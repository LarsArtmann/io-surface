# io-surface Status — Capslock Comparison Session + the 70 MB Binary the Daemon Buried in History

**Report generated:** 2026-11-06 05:36 CET
**Session window:** ~05:15–05:36 CET, 2026-11-06
**HEAD at writing:** `4a11e21` (master)

---

## Scope: what this report VERIFIED vs did NOT verify

**Verified this session (command / artifact):**

- Capslock facts: fetched `google/capslock` README, `docs/caveats.md`, and repo internals
  (`go.mod`, `analyzer/analyzer.go`, `interesting/interesting.cm`, `proto/capability.proto`,
  `cmd/capslock/capslock.go`) via web fetch — capability enum, SSA/callgraph engine, CLI
  modes all read from source, not from memory.
- Working tree state: `git status --porcelain` (**clean**), `git log --oneline -8`,
  `git diff 6705ab1..HEAD --stat` (18 files, +900/−416), per-commit `git show --stat`
  for all five auto-commits, `git ls-tree -r HEAD` (binary IS tracked),
  `git check-ignore -v cmd/io-surface/io-surface` (**NOT ignored**).
- Dependency drift: `git diff 6705ab1..HEAD -- go.mod` (cmdguard v4.2.0, go-finding
  v1.15.1, fang/lipgloss/templ indirects) vs `grep 'external dependency' AGENTS.md README.md`
  (AGENTS.md:6 still claims "one external dependency").
- Local CI bar at HEAD: `go build ./...` + `go vet ./...` (**OK**), `go test ./...`
  (**all packages pass**; `internal/manifest` still has no test files), `gofmt -l .`
  (**empty**), `go run ./cmd/io-surface check --root .` (**0 errors, 0 warnings —
  10 declared, 10 extracted, all resolved**), `render --root .` (**OK**).

**NOT verified (and why):**

- GitHub CI green at `4a11e21` — not attempted; local bar passed but CI also runs the
  facts-freshness gate and SARIF upload, which I did not re-run.
- The exact content of the `facts.cue` ±6-line change in commit `05874f3` — read at
  stat level only; plausibly location drift from the `main.go` refactor, not confirmed.
- Every TODO_LIST evidence `file:line` — inherited from the 04:27 harvest re-verification;
  line numbers were correct then, but `main.go` shrank by ~350 lines since, so several
  `cmd/io-surface/main.go:NN` evidence pointers in TODO_LIST are likely already rotten
  (commands.go now holds the handlers). Not re-verified this session.
- Prior sessions' claims (InboxClean/dnsblockd pilot states, SARIF artifact contents,
  docs-gates PASS) — out of scope per the user's instruction; taken from the reports,
  not re-verified against those repos.
- **This session produced NO code changes.** The deliverable was one verified
  comparison answer (prose) plus this report.

---

## a) FULLY DONE

_(this session only — it was a question-and-verify session by design)_

1. **Capslock vs io-surface comparison, delivered and source-verified.** The answer,
   in one line each: Capslock asks "what capabilities can this package reach
   _transitively_, including through dependencies?" (package/function granularity,
   `go/packages`+SSA+callgraph, ~13 capabilities incl. `CGO`/`UNSAFE_POINTER`/`REFLECT`,
   informational with a `compare`-vs-baseline exit-1 mode). io-surface asks "is every
   I/O call site in this first-party tree _declared, justified, and unchanged_?"
   (exact `file:line` sites, hermetic stdlib AST walk, CUE manifest with
   facts/claims/baseline, bidirectional CI gate with exit codes 0/1/2, two-tier
   honesty where unresolved receivers can never fail a build). Verdict delivered:
   **complement, not substitute** — Capslock wins depth (interfaces, wrappers,
   vendored deps); io-surface wins intent and drift (claims, `location-drift`,
   vanished-declaration detection, per-site reviewable artifact).
2. **Session-state verification pass.** Confirmed the working tree the conversation
   snapshot showed as dirty had in the meantime been swept into five heuristic
   auto-commits (`05874f3`, `e15bf62`, `28d7ebc`, `b1671ce`, `4a11e21`); identified
   each commit's content; ran the full local CI bar and confirmed HEAD is green.

## b) PARTIALLY DONE

_(observed state at HEAD; not authored by this session)_

1. **cmdguard v4 CLI migration.** `commands.go` (+249) + `main.go` (−350) landed via
   auto-commit `05874f3`, with real e2e tests (`main_test.go`, +178) and AGENTS.md
   table updated. Builds, vets, tests pass, self-check green. **Partial because:**
   AGENTS.md's headline claim (`AGENTS.md:6` "one external dependency
   (`cuelang.org/go`)") was not updated — go.mod now carries cmdguard + go-finding
   (+ charm/fang/templ indirects). The README was updated; AGENTS.md was not.
   Split brain, one-line fix.
2. **Facts freshness after the refactor.** `facts.cue` changed ±6 in the same commit
   and self-check reports 10/10 green at HEAD, so the manifest matches the tree —
   but the individual changed lines were not diffed this session (declared-vs-drift
   reconciliation unconfirmed).
3. **The overnight roadmap** (04:27 report + TODO_LIST): SARIF emit, render golden
   tests, CLI tests, `render --check`, facts-freshness CI are DONE; the P0 tail
   (`violations` surfacing #3, manifest tests #12, render tests #13, enforcement
   policy #14, audit.commit #4) is untouched.

## c) NOT STARTED

1. **Binary de-tracking and history purge** — discovered this session, zero work done
   (see d1): no `.gitignore` pattern, binary still tracked at HEAD, history untouched.
2. **AGENTS.md dependency-claim sync** — not started (b1).
3. **TODO_LIST open items** — #3, #4, #7, #8, #9, #12, #13, #14 (High/Med), plus the
   Medium/Low tail (#15, #16, #18–#26, #28–#48), per the harvested list. Not started,
   by design of the harvest.
4. **Capslock as complementary tooling** — cross-check idea from this session's
   comparison; not started, not even recorded in ROADMAP yet (this report is the
   first record).

## d) TOTALLY FUCKED UP

1. **The auto-commit daemon committed a 35 MB build binary — twice.**
   `cmd/io-surface/io-surface` was `git add`-ed by the daemon's heuristic in
   `05874f3` (35,180,278 bytes) and then, after an un-noticed rebuild, committed
   AGAIN in `e15bf62` (35,181,139 bytes). That is ~70 MB of binary blobs permanently
   in git history on both remotes. `git check-ignore -v` confirms **no ignore
   pattern covers it**, so the NEXT rebuild gets committed a third time. The bitter
   symmetry: the 04:27 report's §d1 was "a gitignore pattern ate files I meant to
   commit" — hours later the opposite gitignore failure hit: no pattern exists, so
   a file nobody should ever commit keeps getting committed. Both failures are the
   same root cause: **no artifact-class discipline in staging, only file-by-file
   luck.**
2. **This session shipped zero code.** One comparison answer, this report, nothing
   else. Defensible (the user asked a question and then ordered a report-then-wait),
   but if the bar is "every session leaves the repo better", this session moved
   nothing except knowledge. It ALSO left the binary landmine in place rather than
   defusing it (`git rm --cached` + ignore pattern would have been a 2-minute fix;
   withheld because the instruction was report-then-wait and history-rewrite needs
   an owner decision).
3. **Stale state trusted once.** The conversation's opening git snapshot showed a
   dirty tree; I initially reasoned from it before re-running `git status` — which
   revealed everything had already been committed (including the binary). Caught
   before it corrupted the report, but the instinct to trust a snapshot over a fresh
   command was wrong once this session.
4. **Inherited, still un-fixed:** TODO_LIST #45 ("verified how" discipline). The
   04:27 report hit the same failure class AGAIN (§d5, "5 errors" was 6) within
   hours of codifying the rule. Nothing in this session changes that trajectory.

## e) WHAT WE SHOULD IMPROVE

1. **Artifact-class ignore discipline, daemon-aware.** One change kills d1 forever:
   ignore patterns for build outputs (`/cmd/io-surface/io-surface`, `bin/`, `dist/`,
   or `io-surface` binary name globally) PLUS a daemon-side guard — the daemon's
   heuristic commits by size/count with no binary detection. Repo-side ignore is
   necessary but not sufficient: the daemon demonstrably commits whatever is
   untracked, so `git status --porcelain` output should be filtered through
   `git check-ignore --no-index` before any heuristic `git add`.
2. **Claims that rot get replaced by commands that don't.** `AGENTS.md:6` says "one
   external dependency" — a count that decays on every legit dependency add.
   Replace with the generating command (`go list -f '{{.Path}}' -m all | grep -v /`
   minus indirect, or "see go.mod") so the doc cannot lie.
3. **Use Capslock where io-surface is deliberately blind.** The comparison exposed
   the honest limitation: no transitive/type-resolved reachability, token-tier is
   advisory. Capslock covers exactly that blind spot for dependencies. A periodic
   capslock run over this repo + pilots (GitHub CI only — it needs module
   resolution and cannot run hermetically in the Nix sandbox) would cross-check
   the `facts.extractable: false` escape hatch. Record in ROADMAP, don't implement
   blindly: two capability vocabularies (10 families × 7 disciplines vs ~13 caps)
   must not merge into a split brain — if adopted, document the mapping once.
4. **TODO_LIST evidence lines rot on refactor.** The cmdguard split moved ~350 lines
   out of `main.go` hours after harvest; every `main.go:NN` pointer in TODO_LIST is
   suspect. Improve the format: cite symbol names (`runCheck`, `exitCode`) as
   anchor plus line, so a refactor moves the line but not the anchor. Delegates to
   existing item #47; make it the rule, not a reminder.
5. **Rebuild hygiene that produced d1.** Something rebuilt the binary inside the
   source dir between the two daemon sweeps. Convention should be `go build -o
   bin/ …` (or buildflow-managed output dir) — then even a missed ignore pattern
   has only one place to cover.
6. **Prior reports' §e list still stands** (verify-after-staging not after-push,
   gate-validate hand-ported manifests on arrival, re-verify inherited counts,
   `--json`/`--format json` split-brain watch, CI log capture at triage time).

## f) NEXT TASKS (50, ranked by leverage — brainstorm, not commitment; new items first, then carry-over from TODO_LIST #3–#48)

| #  | Task                                                                                                                                                                                                                                                                      | Impact | Effort         | Source                |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------------- | --------------------- |
| 1  | Un-track the binary: `git rm --cached cmd/io-surface/io-surface` + ignore pattern (repo-wide artifact class)                                                                                                                                                              | High   | 5min           | This session d1       |
| 2  | Decide binary history fate: filter-repo purge (force-push both remotes, break clones) vs accept ~70 MB forever                                                                                                                                                            | High   | owner decision | This session d1       |
| 3  | Daemon-side guard: auto-commit daemon must skip binaries/artifacts (check-ignore-aware heuristic)                                                                                                                                                                         | High   | cross-repo     | This session d1       |
| 4  | Fix `AGENTS.md:6` "one external dependency" → generating-command phrasing (cmdguard + go-finding are real deps now)                                                                                                                                                       | Med    | 5min           | This session b1       |
| 5  | Record Capslock cross-check idea in ROADMAP (capability-vocabulary mapping, CI-only, hermeticity caveat)                                                                                                                                                                  | Med    | 15min          | This session c4       |
| 6  | Diff the `facts.cue` ±6 change from `05874f3` and confirm what the refactor did to declared sites                                                                                                                                                                         | Low    | 10min          | This session b2       |
| 7  | Adopt `go build -o bin/` convention (buildflow-managed output dir) so artifacts have one coverable home                                                                                                                                                                   | Med    | 15min          | This session e5       |
| 8  | Re-anchor TODO_LIST evidence to symbols (`runCheck`, `exitCode`) after the `main.go` → `commands.go` split                                                                                                                                                                | Med    | 30min          | This session e4 / #47 |
| 9  | Print actual `violations` entries on CUE unification failure (opaque "incompatible list lengths" today)                                                                                                                                                                   | High   | 1-2h           | TODO #3               |
| 10 | `audit.commit` = real `git rev-parse --short HEAD` (±`-dirty`), fallback `working-tree`                                                                                                                                                                                   | Med    | 30min          | TODO #4               |
| 11 | Baseline "only shrinks" enforcement (fail on growth without an explicit allow flag)                                                                                                                                                                                       | Med    | 1-2h           | TODO #7               |
| 12 | Baseline entries gain `owner` + `expires`; expired entries fail the gate                                                                                                                                                                                                  | Med    | 1-2h           | TODO #8               |
| 13 | Hint when declared sites fall outside `--dirs` (the 18 silent findings failure mode)                                                                                                                                                                                      | Med    | 30min          | TODO #9               |
| 14 | `internal/manifest` tests: decode edge cases + planted constraint violation (0% today)                                                                                                                                                                                    | Med    | 1h             | TODO #12              |
| 15 | `internal/render` tests: claims columns, judgments section, scope OUT                                                                                                                                                                                                     | Med    | 1h             | TODO #13              |
| 16 | Record enforcement policy (D4) in the manifest schema so repos cannot silently opt out                                                                                                                                                                                    | Med    | 2h             | TODO #14              |
| 17 | Local pre-commit gate: `io-surface check` + `gofmt`/`vet` outside CI                                                                                                                                                                                                      | Med    | 30min          | TODO #15/#16          |
| 18 | BuildFlow provider via `go-finding/toolsdk` so `./bin/dev` runs the check                                                                                                                                                                                                 | Med    | 2-3h           | TODO #18              |
| 19 | Missing extraction families: `os.Stdin/Stdout/Stderr`, `syscall.Mmap`, `net.ListenConfig`, `bufio`, `http.ServeContent`                                                                                                                                                   | Med    | 1h             | TODO #19              |
| 20 | Derive seams from manifest `via` values instead of hardcoded `seamIdent`                                                                                                                                                                                                  | Med    | 2h             | TODO #20              |
| 21 | Cross-package seam support (helper pkg owning the raw call)                                                                                                                                                                                                               | Med    | 2-3h           | TODO #21              |
| 22 | `io-surface baseline add <id> --note --date` subcommand (stop hand-editing CUE)                                                                                                                                                                                           | Med    | 1h             | TODO #22              |
| 23 | `version: "io-surface/v1"` field in manifest; bump on schema change                                                                                                                                                                                                       | Med    | 30min          | TODO #23              |
| 24 | `--ignore` globs for generated code (`*_templ.go`, `zz_generated*`)                                                                                                                                                                                                       | Med    | 1h             | TODO #24              |
| 25 | Gate InboxClean: track `docs/architecture/` + CI check (cross-repo)                                                                                                                                                                                                       | Med    | 30min          | TODO #25              |
| 26 | Declare or baseline dnsblockd's 6 errors; get it to 0 (cross-repo)                                                                                                                                                                                                        | Med    | 1h             | TODO #26              |
| 27 | dnsblockd reverse-mode check + retire `tools/io-extract` (BLOCKED: fleet lock)                                                                                                                                                                                            | Med    | 2h             | TODO #27              |
| 28 | Reclassify bare `.Do`/`.Close` tokens as ambiguous, not a family                                                                                                                                                                                                          | Low    | 30min          | TODO #28              |
| 29 | `extract --stats` per-package histogram                                                                                                                                                                                                                                   | Low    | 1h             | TODO #29              |
| 30 | ~~`version --json` with commit + build date~~ RESOLVED 2026-11-06: shipped (`runVersion` + `TestVersionJSON`); TODO_LIST #30 DONE                                                                                                                                         | Low    | 30min          | DONE 2026-11-06       |
| 31 | Document taxonomy freeze rule (new family = schema version bump)                                                                                                                                                                                                          | Low    | 15min          | TODO #31              |
| 32 | Document ID migration for package moves                                                                                                                                                                                                                                   | Low    | 15min          | TODO #32              |
| 33 | Register repo in fleet inventory (`meta` / stack-selection)                                                                                                                                                                                                               | Low    | 30min          | TODO #33              |
| 34 | `-race` + very-large-repo benchmark; record results                                                                                                                                                                                                                       | Low    | 1h             | TODO #34              |
| 35 | Family-distribution run over crush-daily + ~10 daemon repos                                                                                                                                                                                                               | Low    | 2h             | TODO #35              |
| 36 | ~~Tag `v0.1.0` (const/README/CHANGELOG all say 0.1.0; no git tag)~~ RESOLVED 2026-11-06: `v0.1.0` annotated tag verified pushed to both remotes (`git ls-remote --tags` → `596be743`); superseded same day by the v0.2.0 release (CHANGELOG cut, CI green, proxy serving) | Low    | 15min          | TODO #36              |
| 37 | Smoke-test public `go install …@latest` through proxy.golang.org                                                                                                                                                                                                          | High   | 10min          | TODO #37              |
| 38 | Verify pkg.go.dev indexing after proxy cache                                                                                                                                                                                                                              | Low    | 10min          | TODO #38              |
| 39 | Review/merge the open Dependabot actions-group PR                                                                                                                                                                                                                         | Med    | 15min          | TODO #39              |
| 40 | Provision Forgejo API token on this machine (BLOCKED on external)                                                                                                                                                                                                         | Med    | 15min          | TODO #40              |
| 41 | Server-side push mirror Forgejo → GitHub (BLOCKED by #40)                                                                                                                                                                                                                 | Med    | 15min          | TODO #41              |
| 42 | Flip Forgejo repo visibility per owner decision (BLOCKED by #40)                                                                                                                                                                                                          | Low    | 5min           | TODO #42              |
| 43 | Wire docs-gates into CI as a standing pinned gate                                                                                                                                                                                                                         | Med    | 1h             | TODO #43              |
| 44 | CI job installing the module from the public path (module-path tripwire)                                                                                                                                                                                                  | Med    | 30min          | TODO #44              |
| 45 | Seed `.docs-gates.json` thresholds for this repo                                                                                                                                                                                                                          | Low    | 15min          | TODO #46              |
| 46 | Run Capslock over this repo once, manually, as a blind-spot audit (before any CI adoption)                                                                                                                                                                                | Med    | 30min          | This session e3       |
| 47 | Deprecation plan for `--json` vs `--format json` if a third format lands                                                                                                                                                                                                  | Low    | 15min          | 04:27 §e              |
| 48 | Re-derive GitHub CI status for `4a11e21` (first push after cmdguard landing)                                                                                                                                                                                              | Med    | 10min          | Scope gap             |
| 49 | Consider GoReleaser (or buildflow release path) now that binaries exist as a topic                                                                                                                                                                                        | Low    | 30min          | This session e5       |
| 50 | CHANGELOG entry for the cmdguard migration + dep additions if not already covered                                                                                                                                                                                         | Low    | 10min          | This session b1       |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (3)

1. **Binary history: purge or keep?** `git filter-repo` on `cmd/io-surface/io-surface`
   shrinks the repo by ~70 MB but rewrites history and forces a force-push to BOTH
   remotes (GitHub mirror included), breaking every existing clone. Keep = permanent
   bloat, but zero disruption. This is an irreversible-vs-ugly tradeoff only you can
   make — and if purge: now, before more history piles up?
2. **What is the artifact policy for this repo?** "Binaries never enter git; releases
   come from tags/GoReleaser" — or do you deliberately want a committed binary for
   some workflow I can't see? The answer determines whether #1–#3, #7, #49 are one
   policy line or five separate fights.
3. **Auto-commit daemon: repo-side ignore enough, or does the daemon itself need a
   guard?** The daemon's heuristic committed a 35 MB binary twice within minutes —
   `.gitignore` fixes THIS repo, but the same daemon runs on every repo you have.
   Should the daemon (buildflow territory) learn "never `git add` binaries / check
   ignore before staging", or is per-repo ignore discipline the accepted limit?

---

_Point-in-time snapshot. Section (f) items #1–#8 are this session's fresh discoveries;
#9–#50 are carry-over from the harvested TODO_LIST and prior reports — route through
`docs-health` HARVEST before treating any as committed work. Then wait for
instructions._
