# 2026-11-08 22:45 — errcheck sweep to zero, the max-same-issues trap, dogfood battery green

Session: "io-surface dogfood!" — intercepted by a BuildFlow paste: 5 golangci-lint
errcheck findings remained after auto-fix. The session became: fix the 5 → discover
the linter was showing 3-at-a-time → sweep the whole repo → extend the errcheck
policy → run the full CI-parity dogfood battery → close two preflight advisories
through their owning BuildFlow steps.

## Scope — what this report verified vs. did NOT

**Verified this session** (claim → command → result):

| # | Claim | Verifying command | Result |
| - | ----- | ----------------- | ------ |
| V1 | Zero golangci findings remain | `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s golangci-lint --format finding --budget 5m --max-time 5m` | `"findings": []`, `"total": 0` |
| V2 | No bare report write remains | `rg -n 'fmt\.Fprint' --type go \| grep -v '_, _ ='` | `NONE` (22 sites fixed) |
| V3 | Build/vet clean | `go build ./... && go vet ./...` | both exit 0 |
| V4 | Tests + race clean | `go test -race ./...` | all 12 packages `ok` |
| V5 | gofmt gate clean | `gofmt -l .` | empty output |
| V6 | Self-check gate green | `go run ./cmd/io-surface check --root .` | 17 declared, 17 extracted, 0 errors/warnings/infos, exit 0 |
| V7 | Facts fresh | `init --force` + CI's filtered `git diff -I` | diff empty (`FACTS-FRESH`) |
| V8 | Render + doc fresh | `render > /dev/null`; `render --doc docs/architecture/io-surface/INVENTORY.md --check` | both clean |
| V9 | Both SARIF emits valid | `check --format sarif` / `lint --format sarif` | 0 results each, parseable JSON |
| V10 | Lint radar severity-clean | `go run ./cmd/io-surface lint --root .` | 17 sites, 0 findings |
| V11 | gomod freshness cleared | `buildflow -s gomod-check --format finding` | total 0 |
| V12 | Nix FOD hashes witness green | `buildflow -s nix-hash-fix --fix` | "2 targets built, 2 succeeded, 0 findings" |
| V13 | Push state | `git status -sb`, `git log origin/master..HEAD --oneline` | ahead 2: `2faa439` (TODO_LIST, concurrent session), `6a2c130` (facts.cue audit-line drift from V7's refresh — expected per AGENTS.md). Same 2 on `github/master`. |

**NOT verified / out of scope** (stated, not hidden):

- SARIF zero-result runs omit the `rules` array from the driver (observed:
  driver keys `['name', 'version']` in both emits). Spec-legality NOT researched —
  session was scoped to the errcheck findings; item 8 in §f.
- BuildFlow finding-summary reports `filesScanned: 0` with `filesAffected: 1` —
  contradictory-looking reporting quirk, observed only, not investigated (§f item 7).
- Fleet faces (`scripts/fleetscan.sh`) — not this session's ask.
- BuildFlow binary rebuild — upstream repo work, not attempted.
- LSP-stale-diagnostic claims: the in-editor warnings kept naming already-fixed
  lines; disk state (V2) is the authority, LSP was never restarted.

## a) FULLY DONE

1. **The 5 pasted findings fixed.** `commands.go:236/238/249` got `_, _ =` (stdout
   report writes per the decided policy); `scaffold_test.go:226/233` got real
   `t.Fatalf` error checks on `os.WriteFile` fixture writes (policy: everything
   else is checked). Verify: V1 + V2.
2. **The full sweep — 22 sites, 7 files.** golangci's `issues.max-same-issues`
   default cap of 3 had been revealing the debt three findings at a time; a
   repo-wide `rg` surfaced all of it: `commands.go` 17 report writes, `render.go` 7,
   `scaffold.go` 5, `manifest.go` 6, `absorb.go` 1, `render_test.go` 2 (all
   `strings.Builder` assembly — provably non-failing, same `_, _ =` convention),
   plus `main.go:27` (unflagged stderr write, brought on-convention). Verify: V2.
3. **AGENTS.md errcheck policy extended** (Conventions section, ~line 310): the
   in-memory `strings.Builder` class is now named, the 2026-11-08 extension is
   dated, and the max-same-issues trap is recorded. Verify: `sed -n '310,320p' AGENTS.md`.
4. **Zero findings confirmed with the cache bypassed.** The first re-run served a
   stale cached result (1 hit, 0 misses — findings from BEFORE the fix, at stale
   line numbers). `BUILDFLOW_NO_RESULT_CACHE=1` produced the true zero (V1).
   Lesson: a green-looking re-run can be a cache replay.
5. **CI-parity dogfood battery green** — every `.github/workflows/ci.yml` face
   except the docs-gates step and the Capslock/absorb CI step: build, vet,
   `test -race`, gofmt gate, self-check, facts-freshness, self-render,
   inventory-doc `--check`, gate SARIF, lint-radar SARIF (V3–V10). Version 0.5.1.
6. **Two preflight advisories closed via their owning steps** (not by hand, per
   the BuildFlow doctrine): `gomod-freshness` → `buildflow -s gomod-check --fix`
   (V11); `vendorHash`/FOD warn → `buildflow -s nix-hash-fix --fix`, which proved
   the warn was a mtime-heuristic false alarm — both nix targets build green (V12).
7. **Loop closed per the BuildFlow doctrine**: every surfaced problem ended
   fixed-in-repo; nothing absorbed, nothing left as "mentioned in the run summary".

## b) PARTIALLY DONE

1. **Preflight advisory cleanup**: 2 of the surfaced warns closed (V11, V12);
   these remain, all advisory/non-gating: `env/binary-freshness` (BuildFlow repo
   HEAD ahead of the installed binary), `docs/agents-md-size` (347 lines vs max
   220), `quality/dprint-remote-plugins` (4 network-fetched wasm plugins),
   `quality/formatter-fingerprint-drift` (toolchain upgrades; doctor suggests one
   `buildflow format` pass to re-verify cached verdicts).
2. **The TODO ledger loop**: AGENTS.md carries the extended policy, but I did not
   reconcile it with TODO_LIST.md — `grep '#123' TODO_LIST.md` found no row, so
   the policy's own citation ("TODO #123") references a ledger entry that either
   uses different numbering or is already gone. Unresolved whether the convention
   wants a ticked/annotated row for the sweep.
3. **The original ask ("dogfood")**: the lint detour consumed the session. The
   self-hosting faces all ran (a-5), but the fleet face (`scripts/fleetscan.sh`)
   and the sandbox-native `nix run .# -- check --root .` face did not.

## c) NOT STARTED

1. **Push** — `master` is ahead 2 on BOTH remotes (V13); never push without the
   owner's ask.
2. **BuildFlow + docs-gates binary rebuild and shim refresh** (the standing sharp
   edge; `buildflow doctor` names the delta).
3. **AGENTS.md size decision** (trim to ≤220 vs. record deliberate density like
   the `max_file_size: 650` pin).
4. **TODO_LIST harvest** of this report's §f (docs-health HARVEST).
5. **max-same-issues pin decision** (owner call — see §g Q2).
6. **Cross-project lesson recording** (`max-same-issues` trap → crush-config
   `references/lessons.md`, by commit — it generalizes to every fleet repo).

## d) TOTALLY FUCKED UP

No data loss, no false greens, no broken gate — but brutally honest failures:

1. **Whack-a-mole instead of a sweep.** I fixed the flagged 3 and re-ran. The
   correct first move was the repo-wide `rg` — the per-message cap made the
   "5 findings" paste structurally incomplete (22 real sites). Cost: 2 extra
   fix→verify cycles and a second discovery round.
2. **5 edit calls rejected.** I read render/scaffold/manifest/absorb/render_test
   via bash `sed` and tried to edit; the tool correctly refused (View-before-edit
   is the rule). Wasted a round trip; re-read all five, re-applied.
3. **Fragile JSON handling.** My buildflow-findings inspection crashed twice
   (stdout/stderr interleaving broke `json.load`; then a `KeyError` on `rules`).
   jq exists for exactly this and the jq skill was loadable. V1's final verdict
   came from the raw tail, not a parsed summary — acceptable, but sloppy.
4. **Stale LSP diagnostics for the rest of the session** — the editor kept
   reporting the 5 original (fixed) findings. I trusted disk (right), but never
   restarted the LSP, so every subsequent tool result re-served ghost warnings.
5. **Concurrent-session timing luck.** A second session was mutating
   `.buildflow.yml`, `TODO_LIST.md`, and `docs/status/` while I edited Go files;
   its TODO_LIST edit landed in the same daemon commit family as my files
   (715f3ac). No conflict occurred — but I checked for buildflow loops only once
   at session start, not before each doc-adjacent edit.
6. **The rot the sweep exposed.** The 2026-11-08 errcheck policy decision wrote
   the rule ("stdout/stderr report writes carry `_, _ =`") but the codebase had
   22 violating sites and one off-convention stderr write in `main.go` — the
   decision shipped without a mechanical sweep. That gap is the actual fuckup
   this session inherited; my first fix batch repeated its shape (patch the
   visible, miss the systemic).

## e) WHAT WE SHOULD IMPROVE

1. **Sweep before patch**: for any linter with per-message caps
   (`max-same-issues`), the pasted finding list is a sample, not an inventory —
   repo-wide grep is step one.
2. **View-before-edit as reflex**: partial reads for edit targets go through the
   View tool, never bash text tools.
3. **jq for JSON**, especially buildflow `--format finding` output.
4. **Restart the LSP when disk and diagnostics disagree** — stale warnings
   poison every later tool result.
5. **Policy decisions get a same-day mechanical sweep or an enforcement hook.**
   A decided convention with no sweep and no hook is a wish, not a policy.
6. **Bypass caches explicitly when re-verifying a fix** (`BUILDFLOW_NO_RESULT_CACHE=1`)
   — a cache replay can masquerade as a stale tree.
7. **Record cross-project lessons where they generalize** (the cap trap affects
   every fleet repo with golangci; it belongs in crush-config `lessons.md`, not
   just this repo's AGENTS.md).

## f) NEXT — 25 things (honest count; padding toward 50 would be invention beyond this session's evidence)

Ranked by impact; every item traces to something this session observed.

1. Push `master` to `origin` AND `github` when the owner says go (V13: 2
   mechanical daemon commits pending on both).
2. Harvest this report's §f into TODO_LIST (docs-health HARVEST; strike what's
   already tracked).
3. Rebuild BuildFlow + docs-gates binaries (`nix build` in both repos), refresh
   `~/.local/bin` shims, re-run `buildflow doctor` — clears `binary-freshness`
   and de-risks the false-gate-red sharp edge.
4. Decide AGENTS.md: trim toward the 220-line doctor max, or pin the density
   with rationale in `.buildflow.yml` (owner call — §g Q3).
5. Reconcile the errcheck sweep with TODO_LIST: find what "TODO #123" cites,
   tick or annotate it; add a DONE-row if the ledger expects one.
6. Owner decision on `issues.max-same-issues: 0` (§g Q2): pin it fleet-wide
   (BuildFlow invocation) so full finding lists surface at once.
7. Investigate buildflow's finding-summary `filesScanned: 0` with
   `filesAffected: 1` — reporting quirk or real scan-scope bug (the skill
   explicitly warns: don't trust a green step that scanned zero files).
8. Check go-finding's SARIF emitter: zero-result runs omit `driver.rules`
   (observed V-emits). If SARIF 2.1.0 expects declared rules regardless of
   result count, file upstream (verify-before-filing gate first).
9. Run the sandbox-native gate face: `nix run .# -- check --root .` (exists per
   AGENTS.md; never exercised this session).
10. Run `buildflow format` once so cached formatter verdicts re-verify under the
    drifted fingerprint (doctor's own suggested fix).
11. Decide dprint remote plugins: local wasm paths via nix
    (`BUILDFLOW_DPRINT_PLUGINS=…`) or accept the cold-run fetch — 4 plugins
    currently fetched over the network.
12. One-shot audit for OTHER unchecked-write classes beyond `fmt.Fprint*`
    (e.g. `os.Stdout.Write`, `w.Write`) — the cap trap means more debt may hide
    under different messages.
13. Add the max-same-issues lesson to crush-config `references/lessons.md`
    (cross-project, by commit — global memory is read-only in-session).
14. Add a sharp-edges line to this repo's AGENTS.md: cached BuildFlow step
    results replay stale findings; use `BUILDFLOW_NO_RESULT_CACHE=1` when
    re-verifying a fix.
15. Confirm explicit golden-test health after the sweep:
    `go test ./cmd/io-surface -run TestGolden` passed as part of V4, but a
    deliberate `-update` dry check would prove no golden drifted silently.
16. Verify `init --force` (V7) touched ONLY `facts.cue`'s audit lines — the
    plant-a-claim test covers it, and the daemon commit `6a2c130` shows exactly
    one file; a `git show 6a2c130 --stat` eyeball is already done — close the
    loop in the ledger.
17. CHANGELOG decision: the sweep is internal-only (no behavior change) —
    record the decision NOT to add an entry, so the next release cut doesn't
    have to guess.
18. Confirm the provider face is untouched by the sweep (internal/ changes only;
    no release needed) — state it once so the next session doesn't re-audit.
19. Add `docs/status/README.md` index row for THIS report (newest-first).
20. Re-check `TODO_LIST.md` for conflict markers after the concurrent session's
    commits settle (the buildflow-stash conflict-marker failure mode is
    documented in AGENTS.md sharp edges).
21. After the BuildFlow rebuild (item 3), re-run the full `buildflow` battery
    once — the fleet-standard verdict, not just single steps.
22. Consider a `--strict` BuildFlow posture for this repo (fail on
    warning-severity findings) — the 22 latent sites were all warnings, so the
    gate never tripped on them (§g Q1).
23. Dogfood session 2 scope: `scripts/fleetscan.sh` fleet face + InboxClean's
    v1.2 gate — the part of "dogfood" this session didn't reach.
24. Watch the `/result` symlink regression (commit e7a7aab class): after nix
    builds from items 3/9, `git status` must stay clean.
25. Errcheck policy follow-through discipline: if TODO_LIST gains a policy row
    (item 5), also record the sweep's verification command there, so the row is
    auditable the way every health claim in this repo must be.

## g) QUESTIONS (3 — cannot be figured out from the repo alone)

1. **Enforcement posture**: all 22 latent errcheck sites were warning-severity,
   so the BuildFlow findings gate passed while the debt existed — do you want
   `--strict` (fail on warnings) for this repo, or keep warnings advisory?
2. **Linter config policy**: no `.golangci.yml` by design (AGENTS.md) — should
   `issues.max-same-issues: 0` be pinned through BuildFlow's invocation
   (fleet-wide or per-repo), or do we accept the 3-at-a-time reveal?
3. **AGENTS.md size**: the doctor warns at 347 lines (max 220). Trim it toward
   the fleet default, or pin deliberate density as an exception (same pattern as
   the `max_file_size: 650` pin)?

## Verify block

All claims in §a/§b carry their verifying command inline (V1–V13 table). Git
state at report time: working tree clean; ahead 2 on `origin/master` and
`github/master` (both daemon-committed, neither pushed). This report itself is
uncommitted — the auto-commit daemon picks it up; no manual commit (harness
contract).
