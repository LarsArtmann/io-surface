# Status: erraudit `--max` sweep cleared — 5 anonymous-failure functions coded stdlib-first, production-dead `capability.Families` deleted, generic_return accepted as a recorded posture

**Created:** 2026-11-22 15:19 CET · Crush agent (glm-5.3-flash) · backlog basis: owner pasted `erraudit lint ./... --max` output (16 findings, exit 2, `[Rejection:violations.found]`) with the directive "READ, UNDERSTAND, RESEARCH, REFLECT … execute and verify one step at a time".

**Scope — what this report VERIFIED (all commands run by this session, last state 2026-11-22 15:1x):** `scripts/self-gate.sh` exit 0 twice, final run AFTER every edit (all ci.yml test-job stages incl. docs-gates 5/5); `go build ./... && go vet ./...` clean; `go test ./...` all `ok` (read raw, not via a filter); `gofmt -l .` empty; `erraudit lint ./... --max --disable generic_return` exit 0 (AFTER final edits); `erraudit lint ./... --max` = 15 findings, all carrying erraudit's own "already returns distinctive values (sentinel reference, %w wrap) … treat this advisory as accepted" verdict, exit 2 (re-counted AFTER final edits); `git diff facts.cue` = audit-commit hash refresh only, zero site-location drift. **What this report did NOT verify and why:** `nix build .#` (not run; go.sum unchanged so the vendorHash red class has no trigger, but the flake build is unproven this session); GitHub CI (nothing pushed — pushes are owner-gated); the erraudit `fix ./...` dry-run (skill Step 1 — skipped, see §d.6); `golangci-lint` (repo deliberately does not run it, per `docs/errcheck-policy.md`); whether TODO_LIST/ROADMAP absorbed §f (HARVEST not yet run — §f is its input, not its substitute). **Concurrent session:** the docs-health mega-purge session (its 14:57 report) worked this repo the same day; no collision hit this session's files, and the tree at report time carries exactly two modified files (`AGENTS.md`, `facts.cue`) with all code daemon-committed.

## §a FULLY DONE

1. **All 5 "failures are anonymous" erraudit findings fixed** with coded sentinels + `%w` wrapping, following the repo's `ErrBaseline` house template (`internal/manifest/baseline_write.go:14-16`) — NOT go-error-family, which erraudit's fix template suggests but which would have promoted an indirect dep to a fifth direct dependency against the AGENTS.md architecture invariant:
   - `runInit` → `errNoSites` + `%w` (cmd/io-surface/commands.go:201);
   - `runBaselineAdd` → `errUsage` + `%w` (commands.go:520);
   - `normalizeDepth` → `ErrCapabilityType` + `%w` on both failure arms (internal/absorb/absorb.go), test substring "unknown capability type" preserved;
   - `docBlock` → `ErrDocBlock` + `%w` on all three marker failures (internal/render/blocks.go).
   Sentinels live in per-package `errors.go` files per convention: new `cmd/io-surface/errors.go` (also receives the relocated `errFindings` from main.go), new `internal/absorb/errors.go`, new `internal/render/errors.go`.
2. **`capability.Families` deleted as production-dead** instead of coded: zero callers outside its own test (verified three ways — file grep, `lsp_references`, and reading `Sets` in full); `fmt` import removed with it; `TestFamiliesUnion` deleted. Git history preserves it if the owner reserved it (question §g.2).
3. **Test coverage grew where the sentinels landed**: new `TestRunInit_NoSites` (init on an I/O-free tree → `errors.Is(err, errNoSites)` + exit class 2 — this path had NO e2e test before); new "missing ID is a usage error" subtest asserting `errors.Is(err, errUsage)`; `errors.Is` assertions added to `TestRunRejectsUnknownType` (absorb) and `TestApplyToDoc_Errors` (render).
4. **Pre-existing gopls diagnostics fixed in files this session touched**: `internal/absorb/absorb.go` `CUE()` single-concatenation `WriteString` split into 13 sequential calls (byte-identical output — `TestCUEBytes` green), clearing the `writestring` warning; `splitAnnotationLocation` modernized to `strings.CutLast` (commands.go:651 hint cleared, LSP-restarted to confirm zero diagnostics in all three touched files).
5. **The posture recorded so the sweep is reproducible**: new `docs/erraudit-policy.md` (canonical strict audit = `erraudit lint ./... --max --disable generic_return`; generic_return accepted as a standing decision because `--max` enables `--no-suppress`, which bypasses every `//nolint`, and bare `error` returns are idiomatic + load-bearing for `exitCode`'s `errors.Is` mapping); AGENTS.md conventions bullet added beside the errcheck precedent; new AGENTS.md sharp edge for the gopls `infertypeargs` false positive (§d.4).
6. **Verification battery green at end state**: self-gate exit 0 (twice — once after the main fix, once after the AGENTS.md edit), strict erraudit exit 0, plain `--max` 15/15 accepted-class, vet/test/gofmt clean, facts.cue diff hash-only.

## §b PARTIALLY DONE

1. **The 15 remaining generic_return advisories**: every one now carries erraudit's own "treat this advisory as accepted" verdict, but they are still flagged and plain `--max` still exits 2 — that is inherent (the check flags every idiomatic bare-error-returning function and ignores suppressions under `--no-suppress`). Acceptance is a documented posture, not a code state; nothing structural prevents a future erraudit version from re-wording the verdicts.
2. **gopls `infertypeargs` false positive**: worked around (explicit `[rootFlags, baselineAddFlags]` kept after the compiler rejected inference: `cannot infer T`, cmdguard v5.1.0 `command.go:333`) and documented as a sharp edge — but NOT filed upstream. Filing needs the verify-before-filing gate (minimal repro) + github-voice, deliberately out of this session's scope.
3. **§f below is written but NOT harvested**: per the status-report skill, §f is the input for docs-health HARVEST (TODO_LIST/ROADMAP rows); until that run happens the items are entombed in this timestamped file. The immediate next step after the owner reads this.
4. **Sentinel coverage is in-process only**: the four `errors.Is` test assertions prove classification, but no e2e golden pins the new user-visible error STRINGS ("usage error: …", "malformed doc block markers: …", "unreadable capability type: …") at the `runApp`/exit-code level. The strings are contract-adjacent (CLI grammar) and currently substring-guarded at best.
5. **The absorb `CUE()` rewrite trades readability for analyzer silence**: byte contract proven by `TestCUEBytes`, but 13 sequential `WriteString` calls read worse than the original single template. A deliberate-keep or revert-with-rationale decision has not been made (§f.15).
6. **Self-review questions answered here, not in the brutal-self-review HTML series**: the owner's "what did you forget / what could be better" prompt was folded into this report (§d/§e) per the single-file instruction; the skill's canonical `docs/reviews/*.html` output was not produced, so the self-review series continuity was not maintained.

## §c NOT STARTED

1. **CI wiring of erraudit** — the policy names a strict audit command but nothing runs it on a schedule; ci.yml and self-gate.sh do not include it (not requested; posture decision = §g.1).
2. **go-error-family adoption decision** — deliberately not taken this session (stdlib-first sentinels achieved the errors.Is contract without a new direct dep); if the fleet later standardizes on coded errors, this repo needs its own decision record.
3. **Upstream gopls issue** for the `infertypeargs` false positive (needs repro-first pipeline, §b.2).
4. **DOMAIN_LANGUAGE.md vocabulary check** — "malformed doc block markers" / the sentinel concepts are not domain-language entries; whether they deserve one is unexamined.
5. **Nix build** — `nix build .#` not run this session (see scope line).
6. **erraudit `fix` dry-run** — never executed (§d.6).

## §d TOTALLY FUCKED UP

1. **The absorb.go multiedit produced invalid Go.** Splitting the `CUE()` `WriteString` template, I emitted the continuation fragments as double-quoted Go strings containing literal newlines (`"⏎\tversion: "`) — `newline in string` build red across the whole package. I knew raw backtick strings were required and still wrote the wrong quoting in the JSON-encoded edit. Root cause of the blast radius: I batched three mechanical fixes (absorb split, CutLast, type-arg drop) before building, so the syntax break surfaced one round later, interleaved with an unrelated failure. **Lesson applied in-session: build after each mechanical edit batch; for multi-line Go string literals, write the file with `write`/`lsp_replace_symbol` instead of long escaped `multiedit` payloads.**
2. **The test-status oracle was brittle and nearly lied.** `go test ./... | grep -vE '^ok'; echo EXIT=$?` reports grep's exit code, not go test's — in the broken-build round it printed `nonok-grep-exit=1` (grep fed nothing) next to a red build, and only reading the raw error lines prevented a false-green claim. The pattern was used three times before being recognized as fragile. **Lesson: run `go test ./...` bare and read the output; use `set -o pipefail`/`PIPESTATUS` when filtering.**
3. **The first missing-ID test scenario was wrong.** I wrote `runApp(t, "baseline", "add", "--root", root)` and watched cmdguard's `required:"true"` `--note` flag fail at the flag layer — the test exercised cmdguard, not the `errUsage` sentinel (which lives after `ArgsFromContext` in `runBaselineAdd`). The flag struct was 3 lines above code I had already read. Fixed by supplying `--note`; the lesson is read-then-test, not test-then-diagnose.
4. **Trusted a linter hint against the compiler and paid a build red.** gopls `infertypeargs` claimed `v5.NewParentCommand[rootFlags, baselineAddFlags]` type args unnecessary; removal failed to compile (`cannot infer T`). I applied the hint without an immediate build — the exact anti-pattern the AGENTS.md "challenge tool output" principle warns about. Now documented as a sharp edge with the compiler-outranks-hint rule.
5. **CHANGELOG.md was forgotten.** The repo follows Keep a Changelog; this session changed user-visible CLI error text (three messages gained sentinel prefixes) and deleted an internal symbol, and no Unreleased entry was written. Mitigations: all touched packages are `internal/` or the CLI's own package (no public Go API break), and the changed strings are covered by tests — but the repo's own documented convention was missed and must be closed (§f.1).
6. **The erraudit `fix` dry-run (skill Step 1) was skipped.** The session's evidence that zero `errors.As`→`AsType` migrations exist is inferred from the 16-finding output, not measured by the dedicated auto-fix dry-run that distinguishes the two. Cheap, and it would have made the "no legacy-analyzer debt" claim a measurement.
7. **Small tool failures absorbed cleanly but real**: `lsp_replace_symbol` died mid-deletion (LSP connection closed) and fell back to exact-text edits; an AGENTS.md edit stripped the `**bold**` markers from the adjacent BuildFlow bullet (caught and restored one turn later). No damage escaped, but both were self-inflicted and self-caught — the process, not luck, should catch them.

## §e WHAT WE SHOULD IMPROVE

1. **Make verification commands oracle-strict** — status must come from the tool's exit code, never a filter's (§d.2). Worth a line in AGENTS.md testing patterns if it generalizes fleet-wide.
2. **Build immediately after every mechanical edit batch**, before starting the next batch (§d.1). Cheap, would have halved the failure rounds.
3. **Pin CLI error grammar with full-string tests**, not substrings, whenever an error message changes (§b.4) — the repo pins golden bytes for lint output; error strings deserve the same discipline.
4. **Exercise the posture or it rots**: `docs/erraudit-policy.md` names a strict audit nobody runs automatically. Either wire it as an advisory self-gate stage or record explicitly that it is human-run-only (§g.1).
5. **Split-brain candidates spotted and left for a dedicated pass**: `capability.Result` vs `absorb.Result` (two near-identical fold-result shapes, capability.go:113 vs absorb.go:68) and `printReport` vs `printLintReport` (commands.go:176 vs 605 — near-identical renderers). Both pre-existing; both are `deduplicate-code` fuel.
6. **Doc precision**: AGENTS.md says "four direct dependencies" while go.mod's direct require block lists five entries (go-finding and go-finding/toolsdk counted as one family) — the counting story should be exact where the invariant is load-bearing.
7. **Record the erraudit binary version** (`ee8a415`) in the policy doc — the skill and older reports cite `61a744c`/`d19cca1e`; verdict wording changed between binaries and future sessions should know which binary produced which message shape.
8. **Scope discipline worked and should stay**: the erraudit message template suggested errorfamily adoption and a retype to concrete error types — both rejected on repo-architecture evidence (four-direct-deps invariant, typed-nil landmine), not on effort. Same discipline left render/table.go's three `writestring` warnings untouched (file not touched this session, per the LSP file-ownership rule — they are §f.5).

## §f THINGS WE SHOULD GET DONE NEXT (40, brainstorm-grade — most are ROADMAP fuel, not commitments; HARVEST must route with extra rigor)

1. Add the CHANGELOG `[Unreleased]` entry: changed error-message prefixes (baseline add usage, render marker errors, absorb capability type) + internal `capability.Families` removal note. *(Closes §d.5 — highest priority.)*
2. Decide the erraudit CI posture (§g.1) and, if advisory: add `erraudit lint ./... --max --disable generic_return` as a non-gating self-gate stage so the strict audit is exercised regularly.
3. File the gopls `infertypeargs` false positive upstream: minimal repro against cmdguard v5.1.0 generics, verify-before-filing gate first, github-voice for the text (§b.2).
4. Run `erraudit fix ./...` dry-run once and record the result in the policy doc (closes §d.6's unmeasured claim).
5. Fix `internal/render/table.go:51/59/65` `writestring` concatenations (same class as absorb.go:187, untouched this session by file-ownership rule).
6. Consolidate `capability.Result` and `absorb.Result` (one fold-result shape; split brain, capability.go:113 vs absorb.go:68).
7. Deduplicate `printReport`/`printLintReport` (commands.go:176 vs 605) into one parameterized renderer.
8. Add e2e `runApp` tests pinning the FULL new error strings + exit classes for init-no-sites, baseline-add-usage, and render marker failures (§b.4).
9. Harvest this §f into TODO_LIST/ROADMAP via docs-health HARVEST, then archive this report per the cadence ritual (banner + `git mv`).
10. Add `errors.Is(err, manifest.ErrBaseline)` assertions to the baseline-add duplicate subtest at e2e level (unit tests already do this at baseline_write_test.go:73).
11. Extend `TestRunRejectsUnknownType` to also cover the empty-type arm of `normalizeDepth` (only the default arm is tested).
12. Add `CheckDoc` marker-failure coverage asserting `ErrDocBlock` (currently only `ApplyToDoc` exercises the shared `docBlock` in tests).
13. Record the erraudit binary version (`ee8a415`) + finding count + date in docs/erraudit-policy.md as the re-verification anchor (§e.7).
14. Make explicit in the policy doc that plain `--max` exiting 2 is the EXPECTED human-run result (so a future agent does not "fix" it).
15. Decide keep-or-revert for the 13-WriteString `CUE()` builder (readability vs analyzer silence; byte contract is test-pinned either way) (§b.5).
16. Reconcile AGENTS.md's "four direct dependencies" with go.mod's five-entry direct require block (go-finding/toolsdk counting) (§e.6).
17. Check whether cmdguard v5 exports its own usage-error sentinel; if so, align or reference `errUsage` to avoid a second definition of "usage error" (§f.22-adjacent split-brain check).
18. Sweep `cmd/`+`internal/` for remaining anonymous `fmt.Errorf` sites (per-path hygiene beyond erraudit's function-level verdicts).
19. Add one e2e success-path test for `init` (only the failure path got e2e this session; scaffold's round-trip covers the internals, not the CLI seam).
20. Evaluate `docs/DOMAIN_LANGUAGE.md` entries for "doc block markers" and the marker contract (§c.4).
21. Update `docs/review/libs/` index with erraudit as an adopted quality tool (adoption trigger + re-audit bar per that index's contract).
22. Note go-error-family v0.12.0 (indirect pin) in the libs audit as the standing decision point if the fleet ever standardizes coded errors (§c.2).
23. Consider tightening the `ErrCapabilityType` wording ("unreadable capability type: capability entry without a type" double-says type; cosmetic, then pin with the full-string test from §f.8).
24. Re-verify the erraudit skill's flag claims against binary `ee8a415` (skill cites older binaries; this session empirically confirmed `--disable` + exit codes only).
25. Run `nix build .#` once (flake build unproven this session; go.sum unchanged so risk is low).
26. Re-run `erraudit lint ./... --max --type-aware` (without `--max`'s other enforce flags) to record the advisory-only baseline the policy can reference.
27. Confirm the auto-commit daemon carried every session file (report-time status showed only AGENTS.md + facts.cue modified — verify once more at next session start; daemon blind spot is a known BuildFlow anti-pattern).
28. Add the brittle-filter lesson (`grep` as test oracle) to AGENTS.md Testing patterns if the owner agrees it generalizes (§e.1).
29. Check whether README documents the baseline-add refusal messages (main.go's WithLong names "refuses duplicates and empty notes" — verify the new "usage error:" prefix does not contradict user-facing docs).
30. Give `TestRunInit_NoSites` a sibling asserting the guidance substring ("widen --dirs") so the fix hint survives refactors.
31. Evaluate a shared test helper for the recurring "temp root + planted pure Go file" pattern (TestRunInit_NoSites introduces the third such fixture shape in main_test.go).
32. Schedule the erraudit re-sweep for the next erraudit binary bump (verdict wording is version-coupled — §e.7).
33. Check docs/status/README.md index: add this report's row (the 14:57 session backfilled the index; new reports should land there same-day).
34. Decide whether `errors.go` should also host future cmd-package sentinels by default (errFindings' old main.go home is now free — one home per fact).
35. Verify the sharp-edge claim in AGENTS.md stays true on the next gopls release (false positives get fixed upstream; the entry should be annotated rather than trusted forever).
36. Re-check `go vet ./...` in CI ordering: self-gate runs it, but confirm the new CutLast/WriteString shapes stay vet-clean under the Go 1.27 toolchain bump cadence.
37. Mine `git log -S "not in the closed enum"` when writing the CHANGELOG entry to date the Families introduction precisely (helps the "removed in" line).
38. Consider `errors.Join` for multi-failure paths? — examined and rejected this session (single-cause paths only); record the rejection in the policy doc so it is not re-litigated.
39. Add `erraudit` to the session-start environment checklist in AGENTS.md commands block (it is now part of the repo's quality loop; the commands section does not mention it).
40. Long-term ROADMAP candidate: per-site (not per-function) error-coding audit — this session's fixes satisfy erraudit's function-level verdict, but runCheck/runLint still carry the same anonymous `unknown --format` pattern that only coexists with a sentinel; a stricter tool version would flag them.

## §g THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Should the strict erraudit audit run in CI, and with what teeth?** Advisory stage in self-gate/ci.yml, gating stage, or stay human-run like errcheck? The errcheck precedent is documented-waiver-no-CI, but erraudit has a clean strict mode (`--disable generic_return` exits 0) that errcheck never had — I cannot weigh enforcement posture for the fleet, only execute it.
2. **Was `capability.Families` reserved for planned work?** I deleted it on triple-verified zero-caller evidence (grep + lsp_references + reading `Sets`), but intent does not live in code. If it was staged for the shadow-capability/render evolution, I revert it from git and re-land it with a coded sentinel instead.
3. **Do changed CLI error strings warrant a CHANGELOG entry in this repo's 0.x practice?** Keep a Changelog is the stated convention and I missed it (§d.5) — but before writing the entry I want the granularity call: are sentinel-prefixed internal CLI messages ("usage error: …") changelog-worthy, or below the bar like `_, _ =` policy detail? One line from you sets the precedent for every future error-text change.

---
*Point-in-time snapshot — goes stale by design. §f is the input for docs-health HARVEST (TODO_LIST/ROADMAP), not a commitment list; most items beyond the top five are ROADMAP fuel. Verifying commands for every claim are named inline.*
