# Status: SUPERB resume — P13–P22 shipped, P23 mid-flight (screenshots landed, visual review pending), P24–P26 + push not started

**Session:** 2026-11-15, resumed after the 12:56 report and owner's "GET SHIT DONE — the whole TODO list".
**Ends 15:01 CET on owner order: full status, THEN WAIT.**
**Battery:** 5/5 PASS at write time. **Tree:** clean (the auto-commit daemon carried
stragglers all session; every commit below was verified landed via `git show --stat`).
**Unpushed:** io-surface is 37 commits ahead of `origin/master` (Forgejo primary;
GitHub mirror and CI have seen nothing since the morning push).

## §a — FULLY DONE

1. **P13 closed end-to-end.** BuildFlow `go.mod` pin v0.5.1 landed (daemon commit
   `4e5def7b9`, both files); the in-session commit retries were unnecessary. TODO
   #150 closed with the verified chain (`go get` + providers suite ok 92s +
   daemon-carry note + ~~wrap~~).
2. **self-gate.sh tag-pinned docs-gates bug fixed.** The sed only matched hex
   revs; the new pin is `v0.3.1` — pattern now `[0-9a-zA-Z][0-9a-zA-Z.-]*`,
   verified it extracts `v0.3.1` and the full self-gate (all 12 CI stages) runs
   GREEN.
3. **Cadence hygiene:** the 2 "DONE without wrap" half-strikes were my own
   #156/#161 — both wrapped; the cadence informational warning is gone (0 bare
   DONE rows, classifier replicated exactly before touching anything).
4. **P14 design + P15 implementation: `render --lint` SHIPPED** (commit
   `c4c10e1` + daemon `aaf0f6e`/`291071f` carry the sources):
   - new `render.LintSection(rep)` — advisory GFM table (Severity/Rule/Location/
     Message), pre-sorted contract trusted, empty report renders an explicit
     no-findings line, pipes escaped, location-less rows show `—`;
   - doc-splice API became content-based: `ApplyToDoc(doc, block)` /
     `CheckDoc(doc, block)` take the fresh block string; the CLI composes
     `Block(m) + LintSection(rep)` — renderer owns format, CLI owns which faces
     a doc carries;
   - appendix lives INSIDE the begin/end markers, so `--doc --check` freshness
     covers both faces; the consistency rule is enforced by test
     (`TestRenderLint_DocConsistency`: a `--check` without `--lint` on a
     lint-composed doc exits 1 — drift, by design);
   - this repo's INVENTORY.md self-hosts the appendix; ci.yml freshness step and
     self-gate.sh both carry `--lint`; facts.cue refreshed after the code move
     (gate 0 errors / 0 warnings / 0 infos);
   - tests: `TestLintSection_Golden` (+ empty + purity), composed-block
     round-trip in blocks_test, 3 cmd e2e via `runApp`; goldens re-pinned
     deliberately; README (command table + appendix section), FEATURES row,
     CHANGELOG Added entry; TODO #153 closed with the full evidence chain.
5. **Two audit-all regressions resolved the honest way (harvest, not line-gaming):**
   row #168 minted (report-level golden for `check` text output — genuinely open,
   only lint has byte goldens) and row #170 minted (provider DAG e2e through
   BuildFlow). Both cite their verifying commands.
6. **P16: W3 fleet re-scan RAN and verdict landed.** `docs/evidence/fleetscan-
   2026-11-15.json`: 170 repos (fleet grew 160→170), 115 with findings (was 91;
   the #157 silent-zero fix unlocked root-package repos). Trend table computed
   vs v0.3.1: wna 340→412 (76→99 repos), io-in-loop 351→320 across MORE repos
   (62→67 — per-repo density down, the T16 refinement held), stat-then-open
   50→71, err-dropped 11→28 (17/28 cluster in `template-readme` — a cluster,
   not a trend), cnt 13→12, ctx 3→5, defer 1→1. Promotion-policy §9 got the
   re-scan RUN paragraph; **verdict: nothing promotes** (D4 FP≈0 unmet; volumes
   move away from the bar); FP columns honestly remain the 2026-11-07 samples
   (N≥30 §7 re-sample owed before ANY severity decision). TODO #151 closed.
   Scan-hygiene note recorded: `discordsync` + `DiscordSync` double-listed
   (294 sites each).
7. **P17: stale-tooling RCA + real-red fix.** `buildflow doctor` captured
   (69 pass / 5 warn / 29 "failed" — ALL 29 are not-installed foreign-language
   tools: bandit/eslint/cargo-\*, availability noise for a Go-only repo, not
   gate reds). The REAL red of the day: BuildFlow's flake vendorHash broke on
   my own P13 pin bump (go.sum content changed — the hash was genuinely stale,
   NOT mtime noise). Fixed: `vendorHash.nix` ← hash from the nix error,
   `nix build .#` exit 0, `nix run .#reinstall` → REINSTALL-OK (buildflow-0bfa648
   installed, version 1d474a0). doctor's env/binary-freshness advisory +
   PATH-shadow trap (BuildFlow#81) recorded. AGENTS.md sharp-edge recipe
   refreshed with today's verified facts (real-vendorHash vs mtime-noise
   distinction, PATH-shadow check, doctor availability-noise note). TODO #143
   refreshed (shim current; the home-manager resurrection trap remains,
   owner-gated system switch). TODO #169 minted (docs-gates binary freshness
   tripwire — CI is pin-fresh by construction, local `nix run` entry points
   can lag the pin).
8. **P18: go-taskqueue fan-out count RCA — not reproducible, closed honestly.**
   Per-module fan-out sum == root CLI count EXACTLY at v0.5.1 (54 == 54; all
   54 from the root go.mod scan, zero findings from any of the 26 leaf
   modules; per-module `lint --format json` matrix). The historical +1
   (53-vs-52) pre-dated the same-day #157 silent-zero fix + T16 refinement.
   TODO #145 closed; wording re-checked against the /tmp-citation convention
   (verdict names the command class, not a tmp path).
9. **P19: go-taskqueue claims hour — ALL 232 sites claimed.** facts refreshed
   first (6 calibration-feature newcomers appeared since the last sweep; the
   gate caught them as undeclared-site errors — exactly its job). Every site
   carries resource + trigger; 12 load-bearing sites carry
   `criticality: critical` (radar escalates them); 9 scratch/lock sites carry
   `durability: ephemeral` (radar drops them per the T21 contract). The
   write-non-atomic triage: 4 real-class final-path writes routed to
   go-taskqueue TODO_LIST as an atomic-write batch (systemd unit + pool.conf,
   verdict file, migration restore); `tryFoldLock` triaged false-positive
   (content-free lock, flock owns exclusion) and DROPPED warning→info via the
   ephemeral claim — the modulation contract demonstrably working end-to-end.
   Gate on go-taskqueue: 0 errors / 0 warnings / 5 advisory infos; lint exit 0.
   Committed (`40468e95` + daemon `70f84e26`). TODO #140 closed.
10. **P20: dnsblockd consolidated into ONE owner-decision packet.**
    `docs/planning/2026-11-15_dnsblockd-manifest-first-regen-packet.md`:
    what (manifest-first `init --force`), why-blocked (empirical proof, both
    v0.5.1 and master), three options (A manifest-first regen — recommended,
    ~1 session, schema unchanged; B keep the fork forever; C manual merge
    ritual), acceptance test (round-trip on the dnsblockd manifest itself).
    **TODO #146 CLOSED — premise dissolved:** the 3 claim-conflict warnings no
    longer exist (concurrent session's #139 overlay + claim fills resolved
    them; verified today: 0 errors / 0 warnings / 8 infos, 0 claim-conflict
    findings; MIGRATION-v1.2.md already carries the EXECUTED/REVIEWED banners).
    TODO #27 updated to point at the packet; stays BLOCKED(upstream) pending
    the owner's A/B/C call.
11. **P21: M20 capability-promotion evidence summary delivered, #132 closed.**
    Five-point input for the v1.3 session: blind-spot run numbers (runtime-power
    caps reach every package, ~90% of 236 edges transitive), mapped vocabulary
    sufficiency (1/3 of Capslock DIRECT labels were FP → advisory-first stays
    mandatory), stored state (8 packages, mapped-only, pinned v0.3.4, 2
    justified-alive baselines), exact v1.3 scope for #136, cost statement. The
    decision leg was pre-satisfied (owner ruled FIRST-CLASS 2026-11-08,
    ROADMAP Q5).
12. **P22: schema v1.3 spike written (design-only, as ordered).**
    `docs/planning/2026-11-15_schema-v13-spike.md`: version-gate shape,
    `claims.ownership` enum + conditional helper name + per-value closer-missing
    semantics + policy-not-suppression ruling, closed-TEN capability enum +
    `runtime-power-direct` advisory rule, taxonomy-freeze accounting (families/
    disciplines untouched), session test plan (5 items), explicit non-goals.
    #135/#136 evidence updated; implementation rows stay open (unscheduled).
13. **P23 markdown leg: NEW reusable auditor `scripts/check-md-render.sh`** —
    three GitHub-render breakers checked mechanically (unbalanced fences, table
    blocks without separator rows, unresolved local badge/image paths): all 33
    docs/status files CLEAN. (It is a live script, not a one-off.)

## §b — PARTIALLY DONE

1. **P23 browser leg — screenshots LANDED, visual review NOT done.** Found
   hyperframes' chrome-headless-shell (152.0.7977.30), provisioned its shared
   libs through nix (5 rounds: glib → dbus.lib → mesa-libgbm 64-bit → libxcb →
   libudev), and captured BOTH reports at 320px:
   `docs/evidence/320px-audit/libs-index-320.png` (182 KB) and
   `docs/evidence/320px-audit/full-code-review-320.png` (151 KB). The actual
   audit — looking at the images, judging reflow/overflow/edge-cutting — is
   the remaining 10%. TODO #127 not closed.
2. **check-md-render.sh not yet wired** into self-gate.sh / ci.yml (it runs
   green; wiring is a 5-minute follow-up when #127 closes).
3. **The 320px dependency chain is tribal knowledge** — the working
   LD_LIBRARY_PATH recipe lives in /tmp (`/tmp/ldpath-final.txt`), violating
   the spirit of the provenance convention for reproducible evidence; it
   should become `scripts/320px-shot.sh` (nix-declared) before #127 closes.

## §c — NOT STARTED

1. **P24** — upstream repro drafts (#50 cuelang, #133 second upstream): nothing
   drafted; verify-before-filing gate applies (source-level repro required
   before anything leaves the building).
2. **P25** — fleet packets (#40/#41/#42 upstream reports, #155
   gogenfilter/dnsblockd CI pin confirmations): untouched.
3. **P26** — confirm trio (#142 InboxClean gate state, #147 old GitHub
   Releases deletion, #148 billing blast-radius — the last partially
   superseded by #155): untouched.
4. **Push** — 37 io-surface commits (plus go-taskqueue claims commits and
   BuildFlow pin+hash commits) unpushed to BOTH remotes; CI (GitHub-only) has
   been blind all day, including today's render --lint CI-wiring change. The
   `--lint --check` freshness step and the SARIF artifact step are therefore
   UNVERIFIED IN REAL CI.
5. **Plan errata annotation** — the closing artifact (P12 inversion, #139
   rescope, P13 blocker narrative, the today-lessons) not written into the
   plan file.

## §d — TOTALLY FUCKED UP (honest incident list)

1. **I genericized an audit-all anchor with my own harvest row.** Minting
   #169/#170 pushed DF("itself") from 8 to 9 (threshold: 8) — killing the
   last distinctive token of the 14-39 §f23 item and reddening audit-all
   TWICE. I minted rows before reading the matcher source; reading it showed
   MinTokenLen ≥ 4 (so "dag"/"e2e" NEVER counted — my first fix attempt was
   aimed at tokens that were structurally invisible). Cost: ~3 red cycles.
   Fix was a one-word reword in each row. Lesson recorded: check the token
   table (`--explain-item`) BEFORE minting a harvest row.
2. **The chrome dependency provisioning was whack-a-mole for ~6 attempts** —
   sequential "cannot open shared object file" rounds (glib → dbus → gbm →
   xcb → udev), one 1 MB LD_LIBRARY_PATH that exceeded ARG_MAX ("argument
   list too long"), one 32-bit libgbm (ELFCLASS32). Only after all that did I
   automate it (ldd loop mapping missing sonames → nixpkgs attrs, 2 rounds,
   done). I should have started with the ldd loop.
3. **`t.Parallel()` in the new cmd tests raced the global `stdout` sink** —
   the seam is a package variable; existing tests never mark parallel for
   exactly this reason and I didn't internalize it before writing mine. Two
   false-red tests before removing the marker.
4. **Sloppy append into blocks_test.go** — a heredoc append left a junk
   placeholder helper (`stored()`) and a nonsense mid-test block; caught by
   the build, cleaned via multiedit. I had ONE job there and the append
   pattern was the wrong tool (summary lesson #2 from the morning said
   exactly this: no heredoc surgery on code).
5. **LintSection type error** (`check.Severity` vs string in `table.row`) —
   trivially avoidable by reading `table.row`'s signature before writing the
   call. Build caught it in seconds; still noise.
6. **Race with the daemon on c4c10e1** — my explicit-pathspec commit carried
   3 files because the daemon had already committed 14 of them minutes
   earlier; I composed the commit without a fresh `git status` and got a
   surprise. Harmless (everything landed, verified after), but the sequence
   was sloppy.
7. **Minor:** my sites-extraction one-liner crashed on its own header line
   (AssertionError on "blocks: 226") — cosmetic, fixed with a filter, but it
   wasted a cycle before the claims write.

## §e — WHAT TO IMPROVE (process lessons, owned)

1. **Read the gate's matcher/classifier BEFORE authoring artifacts for it** —
   cost me twice today (cadence classifier cell-scan semantics in the
   morning's tail; audit-all token economics now). `--explain-item` is the
   tool; use it first.
2. **Browser-style dependency provisioning: automate from `ldd` on attempt
   one**, never hand-chase sonames. The working recipe should be a script,
   not a shell history item.
3. **Fresh `git status` immediately before every commit** — the daemon moves
   the ground under you; assume nothing about which files are still mine.
4. **The stdout-sink seam is serial** — never `t.Parallel()` a `runApp` test.
   This should be a named trap in AGENTS.md testing patterns (candidate row).
5. **Run the battery after EVERY living-docs batch** — held this session and
   caught both audit-all regressions within one edit of causing them; keep
   the cadence.
6. **When a gate warning names a count (2 bare DONEs), replicate its
   classifier exactly before touching rows** — my first awk/python attempts
   used wrong column indexes and would have "fixed" the wrong rows.
7. **The `#`-in-case-pattern bash trap** (`|#*)` parses `#` as a comment
   start) cost two syntax errors in check-md-render.sh — quote the pattern.

## f) UP TO 50 THINGS TO GET DONE NEXT

Immediate finishers (this plan):

1. P23: visually review both 320px screenshots; close #127 with the evidence pointers.
2. Wire `check-md-render.sh` into self-gate.sh + ci.yml (new step, mirrors the local gate).
3. Turn the 320px recipe into `scripts/320px-shot.sh` (nix-declared deps, no /tmp reliance).
4. P24: upstream repro drafts for #50 (cuelang) and #133 — verify-before-filing gate: source-level repro or no draft.
5. P25: fleet packets #40/#41/#42 (upstream findings) + #155 (gogenfilter/dnsblockd CI pin confirmations).
6. P26: confirm trio #142 (InboxClean gate state), #147 (old GitHub Releases deletion), #148 (billing blast-radius).
7. Push both remotes (owner-gated; CI is blind until then — the render --lint CI wiring especially wants a real run).
8. Annotate the plan file with errata: P12 inversion, #139 rescope, P13 blocker narrative, today's §d lessons.

Owner decisions queued (packets/rows ready):
9. dnsblockd manifest-first regen: pick A/B/C in `2026-11-15_dnsblockd-manifest-first-regen-packet.md` (#27).
10. Renovate GitHub App install (#161 caveat — config executable, bot owner-gated).
11. #143: home-manager BuildFlow input bump + system switch (kills the stale-binary resurrection trap for good).
12. Schema v1.3 session scheduling (#135/#136 — spike ready, one train carries Q5+Q6).
13. P13-side BuildFlow PR visibility: the pin bump + vendorHash fix are daemon-committed on BuildFlow master — confirm that matches your intent for that repo's history.

New work minted today (rows exist, unscheduled):
14. #168 — report-level golden for `check` text output (45 min, pattern exists in TestGolden_LintOutput).
15. #169 — docs-gates binary freshness tripwire in self-gate/doctor.
16. #170 — provider DAG e2e through BuildFlow itself (toolsdk Detect face → BuildFlow run).
17. Candidate row: name the stdout-sink serial trap in AGENTS.md testing patterns.
18. Candidate row: docs-gates upstream idea — warn when a mint row genericizes an anchor token (DF margin print).
19. Candidate row: document docs-gates matcher facts in the docs-health skill (MinTokenLen ≥ 4, quorum 2, DF 8, skip-marker prefix form).

Fleet evidence follow-ups (from today's scan):
20. §7-compliant FP re-sample N≥30 per rule before any severity decision (policy-mandated; routed to owner).
21. Verify pma double-listing of discordsync/DiscordSync (294 sites each) before the next scan treats them as distinct.
22. Triage the err-dropped cluster in template-readme (17 of 28 fleet hits).
23. io-in-loop at 320 across 67 repos — watch; next refinement only with a new FP sample.
24. stat-then-open at 71/40 repos — the redundant-precheck carve-out class from the 2026-11-07 sample is still the dominant FP class.
25. client-no-timeout at 12 — ctx-deadline/documented-design classes, stable.
26. ctx-dropped at 5 — force-kill class; feeds Q6 ownership vocabulary eventually.
27. defer-in-loop at 1 — stays a curiosity (N too small, W4 blocks).

go-taskqueue follow-through (claims hour fallout):
28. Atomic-write batch: temp+rename the 4 escalated final-path writes (routed in their TODO_LIST today).
29. go-taskqueue: 5 weak-tier infos — declare or seam-convert the .Do sites (same shape as the self-tree's posture).
30. go-taskqueue: their own 259-unarchived-reports cadence debt — separate repo, separate session.

io-surface product surface:
31. Release train: cut v0.5.2 (or 0.6.0) from the Unreleased section (render --lint, --format canonicalization, strict-weak, renovate, self-gate) — release protocol applies.
32. Confirm today's CI-green after push (the freshness step now carries --lint — first real run matters).
33. LintSection: consider an explicit pipe-escape unit test separate from the golden (golden pins it implicitly).
34. Consider lint SARIF face: enforcement metadata for appendix-sourced runs is already "lint" — verify no consumer regressions when v0.5.x ships.
35. render: after v1.3 lands, revisit an ownership column (explicit non-goal today, deliberately).
36. INVENTORY.md self-hosting note in README could name the appendix explicitly (one line, next docs touch).

Docs hygiene:
37. docs/evidence/README.md provenance contract: register the 320px-audit artifacts (script + PNGs) once #127 closes.
38. AGENTS.md: the commands block should gain check-md-render.sh when wired.
39. docs-health skill: add today's matcher facts (see #19) so the next session doesn't re-derive them.
40. AGENTS.md sharp edges: the vendorHash recipe now distinguishes real-vs-noise — worth a matching note in the BuildFlow repo's own AGENTS (cross-repo courtesy, one line).

Deeper queues (not this plan, kept visible):
41. #131-class baseline shrink ritual at the next release (2 justified-alive entries).
42. Absorb/`capability-drift`: refresh the pinned Capslock JSON after the next toolchain bump (v0.3.4 pinned, 2026-11-06).
43. Q5-adjacent: runtime-power-direct rule volume estimate from the existing evidence JSON (cheap pre-implementation number for #136).
44. Promotion policy: the ledger's FP columns want a scripted sampler (scripts/fp-sample.sh candidate) so §7 re-samples stop being manual.
45. fleetscan: record `--dirs` fallback coverage per repo in the aggregate (partially there via `dirs` field — verify all 170 carry it).
46. Self-gate runtime cost: the docs-gates stage compiles from the pin each run — consider a cached profile once the pin stabilizes.
47. baseline.cue: the 2 shadow-capability phantoms should name their verifying command in-note (they do — re-verify at next release).
48. CONTRIBUTING.md still mentions golangci-lint which CI never runs (pre-existing; one-line fix candidate).
49. dprint.json covers non-Go files but nothing enforces it locally (pre-existing; self-gate covers INVENTORY.md only).
50. TODO_LIST cadence: 4 struck rows unarchived (threshold 20) — purge ritual not owed yet, but the next batch of closures should trigger the archive sweep in the same session.

## §g — THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **P23's last 10%:** the two 320px screenshots are on disk but unreviewed.
   Do you want me to finish the visual pass now (my judgment: both reports use
   `auto-fit` grids and stack at 320px — but that is a prediction, not a
   look), or will you eyeball the PNGs yourself and I close #127 on your
   verdict?
2. **Push timing — third time asking, now with stakes:** 37 io-surface commits
   (+ go-taskqueue + BuildFlow) are unpushed; CI has seen nothing all day and
   today's change TOUCHED the CI workflow itself (freshness step now
   `--lint --check`). Push both remotes now so CI validates the CI change,
   or hold until the plan closes (P24–P26 remaining)?
3. **The three `buildflow --fix` processes died mid-session** (~13:40, quiet
   ever since; `pgrep` clean at every check). Were those yours to run (and
   will they return), or should P17's repair-tooling verification restart
   them deliberately next session?

---

**Ledger delta this session:** closed #140, #145, #146, #150, #151, #153,
#132 (+ #156/#161 wrap repairs); minted #168, #169, #170; refreshed #27, #143,
#135, #136. Verification bar held on every code change (build/vet/test/gofmt/
self-check 0-0-0/self-gate GREEN/battery 5-5).
