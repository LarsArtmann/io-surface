# Status Report — BuildFlow Red→Green: Fixture/Formatter Contract, Stale Binary, Result Cache

**Date:** 2026-11-11 14:39 CET
**Session scope:** one `buildflow --fix --build-mode=full` run that entered red
(exit 69, 3 failed steps at 13:05) and the work to make it — and everything it
touches — green (exit 0 at 14:29). No CLI behavior changed; no release cut.

## Scope line — what this report verified vs. did not

**Verified this session (command named per claim):**

- BuildFlow full pass green: `buildflow --fix --build-mode=full …` → exit 0
  (`/tmp/bf-final2.log`).
- Flake green: `nix flake check` → "all checks passed" (x86_64-linux only —
  aarch64-linux omitted as incompatible in this environment, NOT verified).
- Repo battery: `gofmt -l .` (empty) → `go build` → `go vet` → `go test ./...`
  (all ok) → `io-surface check --root .` (17/17, 0 err/warn/info) →
  `render --doc docs/architecture/io-surface/INVENTORY.md --check` (fresh) →
  `docs-gates run` (all 5 gates PASS).
- go-structure-linter clean: `go-structure-linter .` → 1 advisory warning
  (assets/), 0 errors.
- vendorHash valid: `nix build .#` succeeded.

**NOT verified / not attempted:**

- Push state vs. the real remotes: local tracking ref says `master` is in
  sync with `origin/master`, but that reflects the last fetch, not live
  remote truth — I did not fetch or push (no instruction to do so).
- aarch64-linux build (proxy-vendored FOD hashes differently per platform —
  documented; never built here).
- Whether the SAME go-fix-mangles-testdata hazard exists in other fleet repos
  (gogenfilter, InboxClean, …) — not investigated.
- Whether `go-mod-tidy`/`go-mod-update` "rewrote: go.sum" produced any real
  content change (git showed none committed; assumption: byte-identical
  rewrite).
- The 24 go-auto-upgrade / 13 art-dupl / 22 branching-flow / 26 vulnix / 59
  govulncheck findings were triaged only at severity level, not itemized.

---

## a) FULLY DONE

1. **BuildFlow gate green end-to-end** — `buildflow --fix --build-mode=full`
   exit 0; zero failed steps; only detect-only advisories remain.
2. **Fixture/formatter contract restored and frozen** — testdata fixtures
   restored from git blobs (`c85b837` main.go, `94cdfa3` store.go) after
   go-fix mangled them; `flake.nix` now excludes `testdata/**` from treefmt
   (`perSystem.treefmt.settings.excludes`), so goimports/gofumpt can never
   again prune the seam import or rewrite the embed import. CI's
   `gofmt -l .` stays whole-tree and is green.
3. **treefmt exclusion made to actually work** — first attempt
   (`**/testdata/**`) did not match; empirically established in a sandbox
   that treefmt v2.6 global excludes match root-relative globs only.
   Recorded in AGENTS.md.
4. **go-mod-parity step fixed** — root cause: stale BuildFlow binary ran
   probes against a literal `go-mod-parity` binary (fixed upstream in
   BuildFlow `8936ca0a2`); rebuilt + reinstalled from HEAD
   (`nix build .# && nix run .#reinstall`), `buildflow doctor` freshness ✓.
5. **`TestExtract_CrossPackageSeam`, `TestExtract_IDsAreRepoRelative`,
   `TestRun_LocationDriftIsInfoOnly`** — all red from the fixture mangling,
   all green again (full `go test ./...` ok).
6. **docs-gates green (all 5 gates)** — audit-all red fixed by harvesting
   report item 29 (README now documents `absorb`'s dropped-unmapped-
   capabilities report, struck with a verified verdict); cadence red fixed
   by the archive ritual: 16 reports (2026-11-07/11-08 series) moved to
   `docs/status/archived/` with `> **Live remainder:**` banners.
7. **AGENTS.md brought inside the 220-line gate** — 356 → 219 lines
   (linter-counts-wc+1 semantics established empirically); release protocol,
   check semantics, and errcheck policy relocated to
   `docs/release-protocol.md`, `docs/check-semantics.md`,
   `docs/errcheck-policy.md` without fact loss.
8. **golangci-config rule conflict resolved** — `.go-structure-linter.yaml`
   suppresses the rule with a dated (2027-02-11), reasoned pointer to the
   recorded no-golangci decision (TODO #123, AGENTS.md conventions).
9. **vendorHash proven valid** — `nix build .#` green; the nix-checker
   "vendorHash may be stale" finding documented as mtime-heuristic noise.
10. **facts.cue + INVENTORY.md freshness restored** — `init --force` after
    go-fix shifted `commands.go` lines; `render --doc …INVENTORY.md`
    regenerated (the CI inventory gate would otherwise have been red).
11. **SECURITY.md advisory link fixed** — `github.com/lars/…` →
    `github.com/LarsArtmann/…` (was lychee 404).
12. **`cmd/io-surface/main_test.go` import alias** —
    `atomicwrite "github.com/larsartmann/go-atomic-write"` (goimports
    demand; path base is not a valid identifier).
13. **AGENTS.md memory updated in-session** — treefmt/testdata contract,
    stale-binary failure signature, result-cache gotcha, treefmt glob
    semantics, docs-gates verb naming.
14. **Formatter fingerprint refreshed** — `buildflow format` (20 tools ok)
    so cached verdicts re-verify under the new toolchain identity.

## b) PARTIALLY DONE

1. **Mid-run snapshot race understood but not upstream-fixed** — gate builds
   at 13:06 evaluated pre-fix fixture content that the daemon only committed
   at 13:16. Diagnosed and documented; the BuildFlow-side fix (go-fix must
   skip testdata) is only an issue draft, not filed.
2. **Result-cache staleness worked around, not explained** — fixed state
   kept reporting old error findings until `BUILDFLOW_NO_RESULT_CACHE=1`;
   the exact cache-key gap (untracked→tracked `.go-structure-linter.yaml`?
   content-hash timing?) is unresolved.
3. **AGENTS.md trim** — inside the cap, but the trim was driven by the
   linter's off-by-one counting discovered late; two earlier compression
   passes were line-neutral rewraps (wasted cycles). The uniform
   `Live remainder:` banners on 16 archived reports are honest but not
   per-report tailored.
4. **Advisory findings triage** — remaining findings (go-auto-upgrade 24,
   art-dupl 13, branching-flow 22, io-surface self-infos 5, vulnix 26,
   govulncheck 59) are classified by severity/policy but not itemized or
   routed into TODO_LIST yet.
5. **dprint remote-plugin warning** — accepted (plugins cached), not
   localized to nix store paths.

## c) NOT STARTED

- samber/lo suggestion suppression for go-auto-upgrade (policy: declined —
  stdlib-first/hermetic; currently re-reports 24 warnings every run).
- The 5 N+1-shape self-findings (`internal/check/location.go:29`,
  `internal/extract/tables.go:104,115`, `internal/scaffold/scaffold.go:77,81`)
  — info severity, deliberately untouched this session.
- devShell completeness (lychee, prettier, vulnix, go-mod-parity all resolve
  via `nix run nixpkgs#…` WITHOUT project deps).
- `nix flake check --all-systems` (aarch64-linux).
- Forgejo push mirror configuration (server UI; needs an API token not on
  this machine).
- dnsblockd schema v1 → v1.2 migration (owner decision, pending since the
  pilot retro).
- Itemized re-triage of vulnix/govulncheck CVE backlogs.

## d) TOTALLY FUCKED UP (what the session exposed as broken)

Nothing is red right now — but three things were genuinely broken by/toxic
to this session:

1. **go-fix mangled load-bearing fixture data** (2026-11-11, 14d0a61): it
   reformatted `testdata/` as if it were source, pruning the fixture's
   unresolvable seam import and rewriting the embed import — breaking 3
   tests and the seam-extraction contract. BuildFlow's repair layer has no
   concept of "formatter-frozen fixture data". This WILL bite every fleet
   repo with testdata trees until fixed upstream.
2. **Gate-vs-daemon snapshot race** — the 13:05 run's nix evals captured
   working-tree content that mid-run repairs then changed; the resulting
   reds described files that no longer existed by the time a human read
   them. Wasteful, confusing, and it nearly sent me down the wrong path.
3. **Result cache served stale ERROR findings** — after the fix, both
   `buildflow -s go-structure-linter` and a full run kept reporting the two
   dead errors; only `BUILDFLOW_NO_RESULT_CACHE=1` revealed the truth. A
   verification gate that lies after you fix the thing is worse than no
   cache.

## e) WHAT WE SHOULD IMPROVE

1. **Check tool semantics BEFORE iterating**: I wrote `settings.exclude`
   (singular) before reading the pinned treefmt-nix `module-options.nix`
   (`settings.excludes`), and discovered the linter's wc+1 line counting
   only after two line-neutral AGENTS.md passes. Both cost 2–3 wasted
   cycles each.
2. **Empirical sandbox testing FIRST**: the winning move of the session was
   reproducing treefmt in `/tmp/tf-test` with candidate patterns — should
   have been the first step for the excludes question, not the fourth.
3. **Repair layers must respect contract data**: go-fix needs excludes
   (testdata, goldens, `*_pb.go`, generated code) — a fleet-level fix.
4. **Verification gates must be cache-honest**: detector results that gate
   on `--fail-on=error` should either key on the exact fix inputs or be
   re-validatable in one flag.
5. **Green-is-not-green until the cache is bypassed once**: after fixing a
   BuildFlow-reported finding, re-verify with `BUILDFLOW_NO_RESULT_CACHE=1`
   before believing either state. (Now in AGENTS.md.)
6. **Fixture byte-stability should fail at test time**: nothing in the test
   suite pins fixture bytes, so formatter damage surfaced as three
   confusing test failures instead of one loud "fixture changed" assertion.
7. **Uniform archive banners are a compromise**: fast, honest, verified —
   but per-report remainders are the docs-health ideal.
8. **Background jobs were under-parallelized early on** (serial
   `job_output` waits where two builds could overlap).

## f) Up to 50 things we should get done next

_Brainstorm, not commitment — most items beyond the first ~15 are ROADMAP
fuel; docs-health HARVEST should apply routing rigor before any land in
TODO_LIST._

**Upstream / fleet (BuildFlow, go-nix-helpers):**

1. File upstream BuildFlow issue: go-fix must skip `testdata/**` (and
   generated/golden files) — craft a faithful repro first
   (verify-before-filing).
2. File upstream BuildFlow issue: result cache served stale error findings
   after input files changed (determine the exact key gap from cache.db
   first).
3. Upstream: nix-checker vendorHash staleness heuristic should
   content-compare go.sum, not mtime-compare.
4. Upstream: go-mod-parity step should hint "stale binary?" in its failure
   message when the literal-binary signature is detected.
5. Audit other fleet repos (gogenfilter, InboxClean, …) for the same
   testdata-in-treefmt exposure; port the `testdata/**` excludes where
   needed.
6. Consider a fleet-wide `.go-structure-linter.yaml` convention for recorded
   decision suppressions (gogenfilter likely wants the same golangci-config
   suppression).
7. go-standard module: expose a treefmt-excludes option (or document the
   `perSystem.treefmt.settings.excludes` consumer pattern) so the fix isn't
   rediscovered per repo.
8. go-standard module: add lychee/prettier/vulnix/go-mod-parity to
   devShells.default fleet-wide (kills 4 "WITHOUT project deps" advisories).

**io-surface repo (near-term, actionable):**
9. Suppress/record the samber/lo decline for go-auto-upgrade so 24 warnings
stop re-reporting every run (owner decision on mechanism).
10. Fix the 5 N+1-shape self-findings (hoist `os.Stat`/`os.ReadFile` out of
loops) — then `init --force` + INVENTORY regen.
11. Add a fixture byte-stability test (hash-pin `testdata/extract/**` +
`testdata/detectors/**`) so formatter damage fails loudly in
`go test`.
12. Itemize + route the art-dupl 13 findings (likely intentional
table-driven similarity — document carve-outs or dedupe).
13. Itemize + route the branching-flow 22 findings (BaselineEntry naming,
18-field Observation struct, 7-bool checkFlags — real data-model
candidates).
14. Localize dprint plugins (`nix build .#dprint-plugins` +
`BUILDFLOW_DPRINT_PLUGINS=<store>/share/dprint-plugins`) to kill the
remote-plugin warning and cold-run hang risk.
15. Run `nix flake check --all-systems` and record the aarch64-linux
verdict (or pin the exclusion with rationale).
16. Rebuild `result` (`nix build .# -o result`) to clear the vulnix
"stale target" warning.
17. Resolve the push question (see §g-Q1) — 14 local daemon commits, CI runs
on GitHub only after push.
18. docs-health HARVEST: route this report's §f items into
TODO_LIST/ROADMAP (the skill mandates it once the session continues).
19. Give the 16 archived reports per-report tailored Live-remainder banners
(docs-health ANNOTATE rigor).
20. Update CONTRIBUTING.md — it advertises golangci-lint, which is
deliberately absent (stale doc, contradicts the decision record).
21. Verify archived-report item 31 (schema.cue deployed-copy drift guard in
CI) is actually still open and file it if so.
22. Verify item 26 (SARIF enforcement property) shipped state — the strike
text says shipped; confirm the SARIF payload carries it.
23. Item 27 (provider DAG e2e through BuildFlow itself) — still open.
24. Item 28 (`version --json` commit/date stamping in CI release flow) —
still open; v0.2.0-era released binaries affected.
25. Item 32 (CI concurrency group to cancel superseded runs) — still open.
26. Item 33 (cache the capslock install in the capabilities job) — still
open.
27. Item 34 (renovate/dependabot for `go run pkg@pin` lines) — still open.
28. Item 35 (whole-repo dprint debt) — `buildflow format` passed this
session; verify the debt count is actually zero now and strike the row.
29. TODO_LIST #123 (errcheck decision) — the decision is now documented in
`docs/errcheck-policy.md`; consider striking the TODO row.
30. TODO_LIST #64 (max_file_size 650 split) — still parked; the pin now has
its rationale in `.buildflow.yml`.
31. Add `.go-structure-linter.yaml` re-evaluation (2027-02-11 expiry) as a
dated TODO row so the suppression cannot rot silently.
32. Document the go-fix fixture-restore procedure (`git cat-file blob`) next
to the testing patterns (AGENTS.md has it; check FEATURES/TODO_LIST
don't need it).
33. README: document `absorb --check` drift output format in the same depth
as the dropped-report paragraph just added.
34. Consider hoisting the N+1 infos into the detector's own test fixtures
so the tool's self-scan stays stable across refactors.
35. Add a CI or doctor check that AGENTS.md/docs pointers
(`docs/release-protocol.md`, `docs/check-semantics.md`,
`docs/errcheck-policy.md`) actually exist — the relocation created a
pointer graph that can rot.
36. Record the "linter counts wc+1" gotcha in the go-structure-linter repo
docs (fleet-shared knowledge, not just this repo).

**Pilots / adoption:**
37. dnsblockd schema v1 → v1.2 migration — blocked on owner decision (§g-Q2
territory).
38. Verify InboxClean/gogenfilter still pass their own io-surface gates
after this session's fixture contract change (no tool behavior change,
but the fixtures are shared vocabulary).
39. Fleet rescan (`./scripts/fleetscan.sh`) after the N+1 fixes to refresh
`docs/evidence/` numbers.

**Environment / housekeeping:**
40. vulnix backlog: bump nixpkgs pin when binutils 2.46/2.47, bison 3.8.2,
brotli 1.2.x fixes land (CVE-2026-111497 brotli is 9.8 critical).
41. govulncheck GO-2026-6859 (net/http MaxBytesReader race): bump Go
toolchain when nixpkgs ships the fix; re-run and record.
42. `system/io-history-ratchet` warning (29–64 files vs median 3): run one
`--profile-io` pass to confirm it's the vulnix/govulncheck whole-tree
scans, not a real memo-layer regression.
43. Confirm `go.sum` "rewrites" by go-mod-tidy/update are byte-identical
no-ops; if not, feed the diff back upstream.
44. Decide whether `result/` should stay gitignored (it is) or become a
checked artifact pattern for vulnix freshness.
45. Look at the `formatter-fingerprint-drift` cadence — it warned twice
today; a post-`nix flake update` hook running `buildflow format` would
keep it quiet.
46. Assess whether the preflight `docs/agents-md-size` gate should count
like the linter (wc+1) — today the preflight said 221 when wc said 220,
i.e. the two gates disagree by one line at the boundary.
47. Re-check `docs/status/README.md` inventory rules against the 17→1
unarchived swing this session caused (cadence thresholds now trivially
green — fine, but the README may state the old corpus size).
48. Sweep ROADMAP Q5 (capability vocabulary) for the unmapped-capability
names this session's README work surfaced — nothing added, but the
pointer in README should stay in sync if Q5 lands.
49. Keep `buildflow`/`docs-gates` shims fresh as standing hygiene: the
shim-refresh dance worked; consider a weekly `buildflow doctor` cron
note.
50. Close the loop on this report: docs-health HARVEST (item 18) is the
mandatory next step — this list must not stay entombed in a timestamped
file.

## g) Questions I cannot figure out myself

**Q1 — Push or hold?** 14 daemon commits carrying today's green state sit
local-only; local tracking _says_ `master` == `origin/master` but that is
the last-fetched ref, not live truth. Push BOTH remotes now (GitHub CI will
then run the whole battery on the pushed commit), or hold until the Forgejo
push mirror exists so `origin` becomes the only push target?

**Q2 — samber/lo policy surface?** The 24 go-auto-upgrade findings are
stdlib-first policy violations the tool will re-report on every run. Do you
want them formally suppressed (BuildFlow config, like the golangci-config
suppression), or deliberately left visible as standing policy pressure?

**Q3 — go-fix upstream posture?** The fixture-mangling bug will recur in
every fleet repo with formatter-frozen testdata. Do you want me to verify +
file the BuildFlow issue now (I'd build the faithful repro in a scratch
repo per verify-before-filing), or batch it into the next BuildFlow working
session you plan anyway?

---

_Verified how: every claim in §a carries its command in context; the run
logs live at `/tmp/bf-final2.log` and `/tmp/bf-verify.log` (session-local,
not committed). Unverified items are named in the scope line, not hidden._
