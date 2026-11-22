# Pareto-plan execution: v0.6.0 + v0.7.0 releases, go-finding v1.16.0 train, schema v1.3 code-complete

**Date:** 2026-11-22 20:31 CET · **Session:** pareto-plan execution
(`docs/planning/2026-11-22_15-06_SUPERB-io-surface-pareto-unblock-and-release-plan.md`,
owner directive "GET SHIT DONE, the WHOLE TODO LIST") · **Branch:** master
`9a6c3b8`, both remotes pushed · **Battery:** PASS at last run (re-run owed
after the v1.3 edits, see §b5) · **Self-gate:** GREEN at v0.7.0 cut; full
gate re-run owed on the v1.3 tree (see §b5)

## a) Fully done

1. **v0.6.0 released end-to-end** (#173, the plan's 1%→51% tier). Grouping
   decision: the 159 purge bullets shipped as-is (they ARE the done-record;
   the GitHub Release carries curated notes instead — rationale recorded in
   the struck row). const 0.5.1→0.6.0, README status, CHANGELOG cut,
   lint-sarif golden re-pinned (version string only), self-gate GREEN,
   pushed both remotes (`86e2ebe`), CI green (run `45044551330`, 1m44s),
   annotated tag to both remotes, proxy serves `v0.6.0`, GitHub Release live
   (`docs/drafts/v0.6.0-release-notes.md`). The #131-class ritual ran FROM
   CI: the absorb step showed zero drift on `cmd/io-surface` +
   `internal/extract` — both baselined Capslock phantoms alive under pinned
   capslock v0.3.4.
2. **v0.7.0 released end-to-end** (the run-metadata line). With go-finding
   v1.16.0: enforcement posture + rule descriptions ride `runs[0].properties`
   once (`go-finding/meta/enforcement` string,
   `go-finding/meta/ruleDescriptions` map); per-result copies GONE. Commit
   `4df8711`, CI green (run `45058522306`, 27m — GitHub runner backlog),
   tag on the validated SHA, both remotes, proxy serves it, GitHub Release
   live.
3. **go-finding v1.16.0 released** (#133's feature, implemented upstream the
   same session): `sarifRun.Properties`, `Report.SetRunProperty`/
   `RunProperties` (lock-disciplined, copy-isolated),
   `RunPropertiesFromSARIF`, 5 new tests, full go-finding suite green,
   commit `5f568e0` pushed, tag pushed, proxy serves it (verified). The
   mangled `### Changed-` CHANGELOG heading repaired while adding the
   feature entry.
4. **go-finding#53 filed** (LarsArtmann/go-finding, own repo): run-level
   propertyBag request — voice-checked 0/0
   (`scripts/check-draft/check-draft.go --kind body-issue --ai-drafted
   --unsolicited`), citations re-verified at source as the LAST step before
   posting (`sarif_types.go:63/:96`, master `bfd32c7`), landed and verified
   via `gh issue view` (body 2474 chars).
5. **docs-gates#9 + #10 filed** (#178): the §f near-miss warn
   (`SectionFRe = (?i)^#{2,}\s*f\)`, reports.go:18, audit.go:119's blind
   "has no §f section" message; the 6-day incident cited) and the purge
   semantic guard (`splitStrike` never reads the status cell,
   docspurge_collect.go:33/:53; the 6-row purge-eat incident cited). Both
   voice-checked, both landed, both verified. **#178 struck.**
6. **fp-sample.sh + the first §7-compliant FP sample** (#175): the sampler
   re-runs lint per repo from a fleetscan aggregate and writes an
   adjudication worksheet into docs/evidence/. First run: write-non-atomic
   N=30 from 10 repos, ALL 30 hand-adjudicated by reading the flagged code:
   **3 verified / 27 FP = 90% FP rate → stays info, W2 grossly unmet**. The
   verified three are persistent stores without temp+rename (Cyberdom memory
   store Write+Append; DiscordSync blobstore — no read-path checksum, so a
   crash mid-copy persists a truncated blob undetected). §9 ledger row
   refreshed, provenance in docs/evidence/README.md. **#175 struck.**
7. **Phantom class A minimized** (#183): the json-encoder shape was the
   wrong trigger (variants A1–A3 all zero). Bisecting the real tree's
   universe found the pair: a ~10-line **templ repro** (one component
   rendered to a bytes.Buffer) yields 3 phantom NETWORK edges — with
   **net/http ABSENT from the analyzed universe** (module deps =
   github.com/a-h/templ only), the strongest possible misattribution proof:
   capslock names `(*net/http.response).WriteString/Flush` for a package it
   never analyzed. GOEXPERIMENT-independent. Packet §CLASS A MINIMIZED +
   full-tree census refresh (63 entries stable, 7 class-A rooted at
   runCheck/runExtract/runLint) appended. **#183 struck. #50 NOT filed**
   (owner publish order — the packet now carries BOTH standalone repros).
8. **Provider DAG e2e proven** (#170): scratch module (one undeclared
   `os.WriteFile`) through `buildflow run` (be700ae) →
   `providers: 130 total` → `✔ io-surface` → `1 finding reported`, and
   `--format finding` carries
   `"io-surface:io-surface/write-non-atomic:cmd/e2e/main.go:8"` in
   BuildFlow's own finding model — the exact chain the unit tests stub.
   Recipe + proof in docs/evidence/README.md. **#170 struck.**
9. **Fleetscan hygiene pair** (#176 + #177). #176: the discordsync
   double-listing was two REAL paths on disk on 11-15 (identical 294-site
   duplicate checkout), NOT a pma bug; the lowercase checkout was removed
   from disk since; pma today lists exactly one DiscordSync (247 entries);
   verdict recorded in the evidence README. **#176 struck.** #177:
   template-readme's 17 err-dropped hits reproduced and all triaged: **a
   carve-out class, not real bugs** — 15 best-effort cleanup/removal (temp
   cleanup, cache auto-repair eviction, test-dir RemoveAll), 2 explicit
   deliberate ignores; one borderline site (defaults.go:496 preferences
   save) recommended for a site-level ignore with mandatory reason. Full
   triage in the evidence README. **#177 struck.**
10. **320px probe committed** (#181): `scripts/320px-probe.sh` (the
    measurement half; the shot script's soname provisioning mirrored
    verbatim, kept standalone). Verified against the recorded evidence:
    `document scrollWidth 305 <= 320` on docs/review/libs/index.html. The
    three real fixes the harness needed: `--allow-file-access-from-files`
    (file:// iframes are cross-origin by default), `--virtual-time-budget`
    (dump-dom fires before the setTimeout), and extraction scoped to the
    `<pre id="out">` payload (the dump includes the script source).
    **#181 struck.**
11. **docs-health matcher facts** (#179): the audit-all matcher mechanics
    recorded in the docs-health SKILL.md — MinTokenLen=4 (tokens.go:38),
    matchQuorum=2 (check.go:28), DefaultGenericDF=8 (config.go:33),
    SkipMarker prefix form (check.go:18) — all source-verified at docs-gates
    master `c81685b`. Edited the checkout
    (`/home/lars/projects/SKILLS/docs-health`), fan-out guard run clean
    (54 symlinks, no store pins). **#179 struck.**
12. **BuildFlow courtesy** (#180, one half): the vendorHash REAL-vs-noise
    discriminator added to BuildFlow AGENTS.md §Stale vendorHash (go.sum
    content change = real → hash from the nix error; unchanged content =
    mtime noise → plain `nix build .#`; persistent step ERROR = result cache
    → `BUILDFLOW_NO_RESULT_CACHE=1`). The daemon-commit intent confirm
    stays owner-gated (row kept open with evidence staged).
13. **cmdguard-lint re-check** (#63): v0.4.0 is out (two releases past the
    row's v0.2.1 evidence) and STILL ships replace directives — the same
    proxy failure, re-verified today; row evidence refreshed.
14. **Schema v1.3 train: code-complete** (#135 + #136, T8–T10). Full suite
    12/12 green, gofmt clean. Details in §c1's done-list; the train is
    UNRELEASED and its user-docs sync is open (§b).
15. **TODO bookkeeping:** nine rows struck with per-cell DONE verdicts
    (#170, #173, #175, #176, #177, #178, #179, #181, #183); battery
    verified green after the strikes.

## b) Partially done

1. **Schema v1.3 train — docs sync open.** Code + tests complete (§c1);
   NOT yet in: README v1.3 paragraph, DOMAIN_LANGUAGE.md (gate-rule count
   15→16, ownership vocabulary, capability row = closed ten), AGENTS.md
   invariant numbers, FEATURES rows, CHANGELOG [Unreleased] entries.
2. **The scaffold stamp question is UNVERIFIED:** I found no hardcoded
   version string in scaffold Go code; the spike requires new scaffolds to
   stamp `io-surface/v1.3`. Where the stamp actually comes from is open —
   if it resolves to a table still saying v1.2, that's a one-line follow-up.
3. **#63 nudge:** evidence refreshed (v0.4.0 still ships replace) but no
   actual nudge comment was posted anywhere — the row's history treats
   "nudge recorded" as evidence text only; if you want a real ping in the
   cmdguard repo, that's a follow-up.
4. **#180:** the AGENTS.md courtesy half landed; the daemon-commit intent
   confirm (BuildFlow `4e5def7b9` pin bump + vendorHash commits vs owner
   intent) is owner-gated and pending.
5. **Gates on the v1.3 tree:** `go test ./...` 12/12 + gofmt clean is
   verified; the FULL self-gate and the docs-gates battery have NOT been
   re-run since the v1.3 edits (schema.cue + check.go changed — expect
   green, but per repo convention a number without its command is
   unverified, and this one hasn't run at all yet).
6. **#133 strike decision:** the filing AND the upstream train both landed;
   what remains is BuildFlow's require bump (its pickup of io-surface
   v0.7.0 provider). Strike now with the residual routed, or keep open —
   owner's call, evidence updated either way.

## c) Not started

1. **Schema v1.3 train, the code that IS done** (for the record, all
   verified by the suite): version enum v1.3 (both schema copies
   byte-identical, `SupportedVersions` + lookup map, version tests updated —
   typo case moved to v1.4, refusal error names the newest); closed-ten
   `#CapabilityName` + `capability.RuntimePower` classifier + sync-safe
   tests; absorb stores runtime-power names (truly unknown names still drop
   loudly, pinned by fixture + e2e); `claims.ownership` + `helper` (schema
   defaults `*"self"`, Claims struct, `ValidateClaims` wired into `Load` —
   helper required under `helper`, forbidden otherwise); closer-missing
   ownership modulation (caller drops unconditionally — the "covers shapes
   the AST cannot see" policy reading; helper drops ONLY when the named
   helper is really called — a lying claim keeps the finding;
   receiver-insensitive base-name match via the new `FuncCalls` field on
   Observation + `funcCallNames` collector); `runtime-power-direct` (the
   16th check rule, advisory info, baseline-suppressible) with the
   shadow-capability v1.3 split (runtime-power caps never shadow).
   New tests: ownership modulation on the real detector tree, 5 ownership
   validation cases, the rule+shadow split test.
2. **T24 #27 manifest-first regen** — the oldest BLOCKED pilot row
   (`init --force` seeds from the EXISTING manifest, keeps declared-but-blind
   sites + claims/baseline, skips scope when the package declares one;
   dnsblockd round-trip acceptance test; then their Nix-check flip +
   io-extract retirement).
3. **T2–T6 (Forgejo token → mirror → visibility → CI port → InboxClean
   re-confirm)** — no token exists on this machine; the entire chain is
   owner-console-blocked exactly as the plan predicted.
4. **T7 #174 Renovate App** — owner console.
5. **T20 #184/#182** — artifact consumer-or-drop scan (the agent half of
   #184) not started; both severity decisions owner calls.
6. **T21 #166 calendar entry, #167 html archive** — not started (owner
   intent gates #167).
7. **T23 #172 promotion routing decision** — owner verdict; the inputs are
   now one rule fresher (write-non-atomic 90% FP).
8. **T26 cadence ritual** — date-gated 2026-12-02.
9. **T27 post-fix health re-score** vs the 7.25/5.35 baseline.
10. **The purge of today's nine struck rows** — they accumulate cleanly for
    the next `docs-gates purge --apply` (battery stays green with them;
    9 unarchived reports, threshold 15).
11. **BuildFlow require bump** to io-surface v0.7.0 (provider pickup,
    go-ecosystem-upgrade class).
12. **go-finding v1.16.0 release-workflow verification** — the tag is pushed
    but the goreleaser workflow run on it was not watched (v1.15.1's
    nixpkgs-404 fix ships IN this tag; the run should finally go green).

## d) Totally fucked up (owned)

1. **The capslock JSON parse bug that invented an engine change.** I parsed
   the CI artifact looking for capability key `NETWORK` when the real value
   is `CAPABILITY_NETWORK` — concluded "the phantom is gone, the engine
   changed" and burned four experiment cycles (including "proofs" that the
   GOEXPERIMENT/toolchain differed) before downloading the CI artifact and
   reading its actual schema. The control repro "regression" was my parser,
   not the world. The verify-before-filing discipline caught it only because
   I pulled ground truth from CI. Lesson recorded in §e1.
2. **Three malformed row strikes.** #183, #179, #170 all got missing-opening-
   tilde strikes because my string replaces targeted anchors that included a
   leading `~~` which did not exist in the file (the replace silently no-
   oped while the verdict text still landed) — each needed a repair pass,
   and #177's verdict replace also silently no-oped once (anchor with a
   leading space that had already been consumed by the strike replace).
   Root cause: ad-hoc string surgery on structured rows instead of one
   `strike_row(n, verdict)` helper that constructs the full row shape.
3. **The v0.7.0 train dangled mid-session:** pushed master, then let the
   tag wait ~25 minutes on a queued CI run while starting the v1.3 train.
   Protocol-correct (CI green then tag) but the tail should have been
   either completed at the first green poll or explicitly flagged as
   paused — it read as "done-ish" when it wasn't.
4. **Double golden re-pin:** the first `TestGolden -update` for the v0.6.0
   bump ran before the const was bumped to 0.7.0, so the whole pin cycle
   ran twice. Wasted cycle, zero damage.
5. **The docs-health skill edit split a paragraph mid-sentence** (old_string
   ended at a line that continued into the next sentence) — caught and
   repaired immediately, but the mechanical check would not have caught it;
   only the re-read did.
6. **The runtime-power/shadow test churned three rounds** (wrong index
   order, stale assertion after the expectation change, unused variable) —
   the test should have been written against the sorted-output contract
   first.
7. **Started a multi-package schema train while a concurrent session is
   actively landing erraudit work in the same repo.** Zero collisions
   materialized (their active set was cmd/errors + absorb + render; mine was
   schema/manifest/capability/check/lint/extract) and every edit re-read
   first — but the risk was accepted by judgment, not eliminated. The
   un-run full gate (§b5) is the honest cost of that choice.

## e) What we should improve

1. **Verify the parser before interpreting "zero results"** from external
   tool output: dump the actual schema (top-level keys, value enums) before
   any experiment matrix rides on the parse. The CI artifact as ground
   truth caught what four local experiments mis-signed.
2. **One strike helper for TODO rows** — `strike_row(num, verdict, status)`
   constructs `| N | ~~task~~ DONE date — verdict | 🟢 DONE |...` from the
   parsed row, never from anchor-replace gymnastics. The purge-eat trap
   makes malformed strikes a REAL hazard class, not a cosmetic one.
3. **Release trains checkpoint at CI-green:** the tag step is its own
   commit point; don't start the next train until the current one's tag,
   proxy check, and release page are verified (three of the four v0.7.0
   tail steps happened 25 minutes late for exactly this reason).
4. **Train docs sync rides with the code** — the v1.3 CHANGELOG/README/
   DOMAIN_LANGUAGE/AGENTS updates should have been part of the same edit
   series, not deferred to a §b list; the taxonomy freeze is a
   documentation event as much as a code event.
5. **Pre-flight collision check before multi-package work:**
   `git log --since=2h --name-only` against the concurrent session's file
   set beats per-edit vigilance.
6. **Write rule tests against the sorted-output contract first** (the
   check rules sort by package): fixture indices derived from the sort,
   not guessed.

## f) Up to 50 things to get done next

1. Re-run the FULL self-gate on the v1.3 tree (§b5 — build/vet/test/gofmt/
   self-check/facts-fresh/self-render/inventory-doc/dprint/SARIF).
2. Re-run the docs-gates battery post-strikes-and-v1.3 (freshness/audit-all/
   annotate/routing/cadence).
3. Verify the scaffold's new-manifest version stamp emits `io-surface/v1.3`
   (§b2 — locate where the stamp is produced; bump if hardcoded).
4. CHANGELOG [Unreleased]: the whole v1.3 train (Added: runtime-power
   capabilities stored at package level, claims.ownership + helper,
   `runtime-power-direct` advisory rule; Changed: shadow-capability never
   fires on runtime-power caps; the 16-rule count).
5. README: v1.3 paragraph (ownership semantics, closed-ten capabilities,
   the new rule) + the closer-missing rule card's FP-mapping note
   (cache store→caller, pool/session→caller, pool hand-off→helper).
6. DOMAIN_LANGUAGE.md: 15→16 gate rules + the runtime-power-direct row +
   ownership/helper vocabulary in the Claims row + capability row (closed
   ten, package-level).
7. AGENTS.md: invariant numbers (check-rule count, "closed ten" capability
   vocabulary sentence) + the v1.3 line in the schema-version sentence.
8. FEATURES.md: v1.3 rows (DONE: capability storage + ownership modulation;
   the advisory rule).
9. Strike #135 + #136 with implementation evidence, flagging the one
   interpretive call: `caller` drops UNCONDITIONALLY (spike's "covers
   shapes the AST cannot see" read as policy-not-suppression).
10. Decide the v0.8.0 cut scope with the owner: v1.3 rides the next release
    train promptly (taxonomy freeze lands publicly) or holds for line
    review (§g2).
11. Mint the fixture-e2e TODO: a manifest fixture carrying a DIRECT
    runtime-power cap through a full `check.Run` (the rule is tested at
    function level; the spike's §5.3 asked for an end-to-end fire).
12. Decide/document the SARIF-visibility question: ownership drops have no
    SARIF trace (no finding exists to annotate); confirm "absent by design"
    in the rule card or add a drop-reason metadata line later.
13. Document the helper-match known-FP class (receiver-insensitive base
    name can false-drop, e.g. a claim naming "Read" on a function that
    calls `f.Read`) in the rule card; consider requiring the full dotted
    path when the AST can carry it.
14. #50: file on google/capslock when the publish order lands (packet
    carries both standalone repros; re-verify line numbers + templ version
    at posting time per verify-before-filing §5).
15. #172: write the promotion-routing decision doc with the fresh ledger
    (write-non-atomic 90% FP → stays info).
16. Complete the §7 re-sample for io-in-loop (320 hits) and stat-then-open
    (71 hits) — the only other rules above the N≥30 floor.
17. T24 #27: implement manifest-first regen (`init --force` seeds from the
    existing manifest; keep declared-but-blind sites + claims/baseline;
    skip scope when the package declares one).
18. dnsblockd round-trip acceptance test for the regen (the packet's
    criterion), then their Nix-check flip + `tools/io-extract` retirement.
19. T2: owner provisions the Forgejo token (packet §1; lands at
    ~/.config/forgejo/token chmod 600) — unblocks 16–20 below.
20. T3: push mirror via runbook §2 (POST push_mirrors, gogenfilter deploy
    key, marker-commit sync ≤10 min, AGENTS remotes flip).
21. T4: Forgejo visibility decision record (stay private through burn-in).
22. T5: Forgejo CI port (evo-x2 runner `native` label, port ci.yml without
    windows, first green run, required-status flip).
23. T6: InboxClean gate re-confirm + the billing blast-radius inventory
    (#142 + #148).
24. T7: Renovate App install + first pin PR verify (#174).
25. T20: artifact consumer-or-drop scan for the lint SARIF + capslock.json
    uploads (#184 agent half).
26. #182: edit self-gate.sh tripwire severity after the owner WARN/FAIL
    call; probe the stale-entry-point path.
27. T21 #166: calendar entry 2027-02-11 for the suppression re-evaluation.
28. #167: confirm intent → trash portfolio-io-insights.html + evidence
    bullet update.
29. #143: home-manager input bump + system switch (owner, system level).
30. #180: owner intent confirm for BuildFlow's daemon commits
    (`4e5def7b9` pin bump + vendorHash).
31. Verify go-finding v1.16.0's release workflow run (goreleaser; the
    nixpkgs-404 fix is in this tag — it should finally go green end-to-end).
32. Close go-finding#53 with "shipped in v1.16.0" (or hold for owner review
    of the implementation, §g3).
33. BuildFlow: bump require to io-surface v0.7.0 (+ toolsdk v1.16.0
    already), run their gate loop, then their release — provider pickup
    (go-ecosystem-upgrade class).
34. Mint the err-dropped carve-out refinement TODO (cleanup-removal on
    cache/temp paths — the #177 verdict's candidate; detector work rides a
    future session).
35. Mint the template-readme defaults.go:496 site-level ignore TODO (the
    borderline preferences-save site; mandatory reason per policy).
36. Next fleetscan run: verify single DiscordSync entry; fresh evidence run
    for the ledger (runtime-power-direct will add info findings fleet-wide —
    expected, artifact-only).
37. Promotion-policy §9: refresh the remaining 8 ledger rows' N/FP columns
    at that evidence run (2026-11-07 columns are stale).
38. docs-gates#9/#10 follow-through: upstream PR once triaged (micro-043).
39. Purge today's nine struck rows (`docs-gates purge . --apply`) — verify
    each lands as a clean bullet, no fragments.
40. T27: post-fix health re-score vs the 7.25/5.35 as-found baseline.
41. Coordinate with the concurrent erraudit session: ID-space (next mints
    are #185+), confirm their errorfamily train is complete before any
    v0.8.0 cut.
42. Cadence ritual 2026-12-02: verify remainders struck, prepend
    live-remainder banners, archive the 11-11/11-12 reports (T26).
43. go-finding: the CHANGELOG "Sub-module changelogs" links still list
    pipeline/analysis/cmd/toolsdk — confirm none grew a v1.16.0-relevant
    entry that belongs in the root cut notes.
44. io-surface CI: the capabilities job's 3 advisory drifts (internal/lint,
    internal/scaffold, provider reach deeper than committed) — refresh
    capabilities.cue via a local absorb with the CI toolchain, or record
    the drift as accepted.
45. Mint the absorb-refresh TODO properly if 44 is accepted drift.
46. Verify pkg.go.dev indexes v0.7.0 (crawler lag is normal; the proxy is
    the guarantee — one `go list -m` already passed).
47. Consider minting a TODO for `runtime-power-direct`'s D4 posture note:
    it ships at info "forever absent FP≈0 evidence" — the promotion-policy
    §9 ledger gains a row at the next evidence run.
48. The lint face could surface ownership drops as a COUNT in its summary
    ("N closer-missing dropped by claims.ownership") — visibility for the
    declared policy without SARIF noise; small feature, needs a golden.
49. Strike #63's evidence convention: decide whether "nudge recorded" rows
    need an actual upstream ping artifact (an issue link) to count.
50. Update the pareto plan file's execution marks (T1/T11-half/T12/T13–T19/
    T22/T25 status) so the next session reads outcomes, not intentions.

## g) Owner questions

1. **#50 publish order:** the packet now carries BOTH standalone repros
   (class B `strings/iter.go:41` + class A templ with net/http absent from
   the analyzed universe — the stronger proof). File as-is on
   google/capslock, or do you want to line-review the draft first? (The
   provenance banner stays unchecked until your review either way.)
2. **The v1.3 train's one interpretive call:** `claims.ownership: "caller"`
   drops closer-missing UNCONDITIONALLY — I read the spike's "covers shapes
   the AST cannot see" as policy-not-suppression (the human asserts the
   handle leaves the function; no AST corroboration required). Cut v0.8.0
   with this reading, or do you want the stricter variant (caller drops
   only when the AST sees a return/store, with the declared policy covering
   the rest via an explicit message) before the taxonomy freeze lands
   publicly?
3. **go-finding#53 + the v0.7.0 breaking shape:** I filed the propertyBag
   request AND implemented/released it (v1.16.0) the same session, and
   v0.7.0 REMOVED the per-result `go-finding/meta/enforcement` /
   `ruleDescription` keys (consumers must read the run bag). Close #53 as
   shipped, and confirm BuildFlow's require bump to io-surface v0.7.0 is
   mine to execute (it is the only consumer of the dropped keys I know of).
