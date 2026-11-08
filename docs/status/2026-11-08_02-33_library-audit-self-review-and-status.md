# Library-utilization audit — status & brutal self-review

> **Session scope:** the 2026-11-07/08 run that produced `docs/review/libs/`
> (22 per-library reports + `index.html` + screenshot + AGENTS.md memory).
> **Verified:** all 22 reports' verify blocks replayed; all internal links and
> anchors resolve; gates run at HEAD (`go build ./...`, `go vet ./...`,
> `gofmt -l .` empty, `go test ./...` all ok, `check --root .` exit 0, `lint`
> report-only); index.html rendered in headless chromium (DOM dump + 431 KB
> screenshot at `docs/review/assets/libs-audit-index-2026-11-07.png`); module
> proxy checked for the 6 in-graph modules.
> **NOT verified:** the remaining CI battery (render, facts-freshness diff,
> inventory-doc `--check`, SARIF emit) was not run — docs-only change made it
> low-risk, but "gate green" ≠ "CI green"; the latest-version claims for 16
> of 22 libs rest on local git tags, not the proxy; `docs/reviews/` and
> `docs/research/` were never swept for a prior overlapping dependency study
> (only `docs/review/` was confirmed absent and AGENTS.md consulted).
> Author: Crush (session of 2026-11-07 → 11-08). Date-time: 2026-11-08 02:33.

---

## a) FULLY DONE

1. **Skill discipline at entry**: `library-deep-dive` + `html-report-kit` loaded before any task action; user's explicit output paths (`docs/review/libs/*.md`, `index.html`) correctly overrode the skill's defaults.
2. **Usage mapping, evidence-grade**: direct imports (`rg` over all `*.go`), direct+indirect `go.mod` lines, and `go mod why -m` chains for every LarsArtmann module in the graph (go-error-family via go-finding; go-output/go-branded-id/go-ndjson/samber-do-auditlog via cmdguard).
3. **Version research**: local tags + dates + commits-ahead for all 20 repos / 22 modules; module-proxy confirmation for cmdguard (v4.2.1), go-atomic-write (v0.6.2), go-finding (v1.15.1), toolsdk (v1.16.0), go-error-family (v0.12.0), go-output (v0.38.7); changelog/git-log reading for both bump gaps (cmdguard `887dcae` fix in v4.2.1; atomic-write v0.6.1 temp-leak fix).
4. **22 reports written** (`docs/review/libs/*.md`), each with status, verdict, evidence at file:line, capability table (N/A rows all carry reasons), recommendation, and a verify block. Every verify block was replayed literally before shipping.
5. **`index.html` dashboard** built from the kit template by copy+in-place-edit (CSS untouched): 22-row verdict matrix, findings, strengths, actions, module graph, verify section; all 22 links resolve, all 6 anchors exist, zero template demo residue; rendered in headless chromium, screenshot preserved in-repo.
6. **Gates green** at audit HEAD: build, vet, gofmt (prints nothing), full test suite, self-check exit 0, lint 0 errors/4 infos (report-only by design).
7. **AGENTS.md** gained the durable audit outcome (verdicts, pending bumps, two recorded triggers, re-audit trigger).
8. **Consistency pass executed**: both adoption scores re-derived from their tables (cmdguard 93/100 = 6.5/7; atomic-write 80/100 = 4/5), badge classes checked against the kit CSS, submodule counts corrected against go.mod.

## b) PARTIALLY DONE

1. **"Latest versions" guarantee** — proxy-verified for the 6 in-graph modules only. The other 16 cite local git tags; for Lars repos local tags are the release source, but the user demanded "LATEST VERSIONS" and I did not proxy-confirm them. Worst case: `go-credentials-store` — local tag v0.2.1, its own README claims published v0.3.1; discrepancy unresolved in the report (cited both, no resolution).
2. **Done-bar "Index reachable"** — index.html exists and AGENTS.md points to the series, but nothing links there from README or a docs root; a session that skips AGENTS.md won't find the audit.
3. **"Gate green" was claimed, CI battery wasn't fully run** — missing locally: `render --root .`, facts-freshness (`init --force` + filtered `git diff`), inventory-doc `--check`, SARIF emit. Docs-only change makes breakage unlikely; the claim was still broader than the evidence.
4. **Skill Phase 7 (commit)** — correctly skipped per harness never-commit rule; auto-commit daemon picked the files up (verified via `git status`).

## c) NOT STARTED

1. **The two patch bumps themselves** (cmdguard v4.2.0→v4.2.1, go-atomic-write v0.6.0→v0.6.2) — reported as the audit's only action item, not executed (one `go get` + gates each).
2. **TODO_LIST.md routing** — the bumps and two adoption triggers live in AGENTS.md prose and this report, NOT in the canonical TODO_LIST. That's both an omission and a docs-ownership violation (AGENTS.md must not be a TODO).
3. **docs-health HARVEST of this report's §f** — the status-report skill mandates routing §f into TODO_LIST/ROADMAP when the session continues; user ordered WAIT, so it is parked as the first candidate action.
4. **Prior-series sweep** — `docs/reviews/*.html` (brutal-self-review's home) and `docs/research/` never listed; no confirmation that no overlapping dependency study exists.
5. **Deep-dives for go-finding + go-finding/toolsdk** — direct LarsArtmann deps that were off the user's list of 22; no report exists for them.

## d) TOTALLY FUCKED UP (all caught and fixed in-session, but each shipped to disk wrong first)

1. **Scratch arithmetic shipped in a report**: `go-atomic-write.md` was written with mid-computation stream-of-thought ("83? No — … 80? Recount…") AND a header claiming 90/100 against a derived 80/100 — precisely the denominator-drift failure the deep-dive skill's self-consistency gate names. Two edits to repair. The worst quality miss of the session.
2. **Three subagents killed by rate limits**: dispatched 4 parallel read-only agents on one provider; candice/amy/jacopo died instantly ("too many requests"). Retried nothing; redid all three scopes by hand (the fourth, luka, succeeded). A staggered dispatch or direct research from the start would have cost one round-trip less.
3. **`rg -rn` typo in a verify block** (credentials-store) — the done-bar names exactly this class (`rg --g`-style verify-command typos) as a check; I still wrote one and caught it only in review.
4. **`badge-warn` CSS class that does not exist** in the kit — used in 2 matrix rows before checking the component vocabulary; fixed to `badge-medium` after grepping the template CSS.
5. **Two inaccurate verify-block counts** — "2 files" (actual: 4 mention cmdguard in cmd/) and "root + 10 submodules" (actual: 11) — denominator drift AGAIN, twice, in the same series, after the skill had already warned me once. Both caught in the replay/consistency pass.
6. **Overclaimed final summary**: last turn's "audit complete — gate green" was true for the gate but presented as full assurance; the CI battery gap (b.3) went unstated.

**Pattern, named honestly:** I compose first and verify afterwards. The kit and skill both prescribe verification as a pre-ship step; I ran it as a post-ship pass. This session it cost five fix-up edits; on a code change the same habit costs bugs.

## e) WHAT WE SHOULD IMPROVE

1. **Pre-ship report checklist**: replay every verify command, recompute every X/N from its table, grep component vocabularies — BEFORE the file is written, not after.
2. **Subagent dispatch**: never fan out N parallel agents on one provider; stagger or serialize (rate limits are deterministic, not luck).
3. **Version claims**: a "latest version" claim about a published Go module requires a proxy check, regardless of whether the module is in our graph.
4. **Docs ownership at creation time**: action items go to TODO_LIST.md when discovered, not into AGENTS.md prose or report footers.
5. **Gate honesty**: state exactly which gates ran ("build/vet/test/check/lint") instead of the umbrella "gate green"; or run the full CI battery locally.
6. **Component kits**: read the kit's available classes/components before composing content against them.

## f) Next things (35, impact-sorted; 1–9 are this session's direct fallout, 10–35 are what the session noticed)

1. Execute the cmdguard bump `v4.2.1` + full gates (free, API-stable).
2. Execute the go-atomic-write bump `v0.6.2` + full gates (temp-leak fix).
3. HARVEST this report's §f into TODO_LIST.md; move the bump/triggers out of AGENTS.md prose (one fact, one home).
4. Proxy-check `go-credentials-store@latest`; resolve the v0.2.1-tag vs README-v0.3.1 discrepancy; annotate its report.
5. Proxy-check the remaining 16 libs; annotate any report whose local tag ≠ published latest.
6. Link `docs/review/libs/` from README or a docs index (discoverability without AGENTS.md).
7. Run the full local CI battery once (render, facts-freshness diff, inventory-doc `--check`, SARIF emit) to convert "gate green" into "CI green".
8. Sweep `docs/reviews/` + `docs/research/` for prior dependency studies; if one exists, mark supersession.
9. Link the screenshot from index.html's footer (it's in `docs/review/assets/` but unlinked).
10. Deep-dive reports for `go-finding` + `go-finding/toolsdk` (direct deps, off the original list; `internal/emit` and `provider/` are real consumers).
11. Check toolsdk v1.16.0 vs go-finding v1.15.1 skew — should they be aligned?
12. Triage the 4 self-tree lint infos (e.g. the `os.Stat` loop notice) — fix or suppress with `//io-surface:ignore` + reason.
13. Decide the 8 pre-existing errcheck warnings in `commands.go` (stdout-sink `fmt.F*`): fix, waive with rationale, or adopt errcheck into CI (no `.golangci.yml` exists today).
14. Check CUE version currency (`cuelang.org/go v0.17.1`) — direct dep, never version-checked this session.
15. Confirm BuildFlow picked up provider v0.4.0 (sdk_imports blank import ships with the release protocol).
16. ~~Verify README install/status lines still match version const `0.4.0`.~~ RESOLVED 2026-11-08 (superseded by two releases since: v0.5.0 and v0.5.1 — the README status line was re-cut for both (`grep -n "v0.5.1" README.md`), the const is `0.5.1` (`cmd/io-surface/main.go`), and the install line is version-agnostic `@latest`).
17. Consider the 2 baselined sites — can either be declared or removed yet (baseline only shrinks)?
18. ROADMAP Q5 (capability vocabulary promotion) — still open; audit's absorb notes touch it.
19. Mobile check: render index.html at 320 px (only 1280 px was screenshotted).
20. Visual pass of the 22 .md files on GitHub (tables/badges render).
21. Decide whether dprint should cover `docs/review/**` (config exists, CI doesn't enforce).
22. CHANGELOG entry for the audit? (docs-only — probably no; decide explicitly.)
23. If reports ever get refreshed: record a "supersedes" line (series discipline).
24. `check --watch` prototype on go-filewatcher/v2 — ONLY if a pilot asks (recorded trigger; do not build unprompted).
25. Re-verify the two deep reports at bump time (their version-currency rows go stale the moment §f.1–2 land).
26. go.mod graph hygiene: 80+ indirects for a "minimal" CLI — mostly cuelang.org/go's tree; document that this is accepted doctrine (or trim CUE usage).
27. Why is `github.com/a-h/templ` an indirect at all (no .templ in repo)? Trace and note the chain.
28. Add a series README (`docs/review/libs/README.md`) if more review series will accumulate.
29. Status-report follow-through: after WAIT, run docs-health HARVEST + ANNOTATE routing for this file per the skill.
30. Consider whether `provider/` two-face Detect needs a test asserting audit-demotion for BOTH faces (posture change is one knob — is it covered?).
31. The 2026-11-07 buildflow-revert sharp edge: re-verify no working-tree files were silently reverted during this session (git status showed only expected files — repeat after daemon settle).
32. errcheck aside: consider `go test -cover ./...` baseline capture for the audit's next rerun.
33. If a website ever launches (website-launch skill), the audit dashboard belongs in the docs tree — parked.
34. Question the doctrine itself at v0.5.0: is "four direct deps" still the right marketing number once go-finding+toolsdk count as two of them?
35. Revisit this status report after items 1–9 land; annotate rather than rewrite (docs-health ANNOTATE).

## g) Questions I cannot answer myself

1. **Execute the two patch bumps now?** They are one command + gates each, but this repo's release protocol treats dependency/version moves as deliberate owner decisions — say the word and I'll run them.
2. **HARVEST this report's §f into TODO_LIST.md, or do you curate manually?** The skills say harvest; you said wait — your call.
3. **Should go-finding + go-finding/toolsdk get the same deep-dive treatment** (they are direct LarsArtmann deps that your list of 22 omitted), or is their coverage in AGENTS.md sufficient?

---

**Ghost systems check:** none created — everything written this session is either wired (reports → index → AGENTS.md pointer) or a run trace (screenshot). **Split brains found:** one small — audit action items currently exist in AGENTS.md prose AND this report's §f but in no canonical TODO list; fixed by item 3 above. **Did I lie?** No intentional falsehoods; five shipped-then-corrected inaccuracies (§d) and one unresolved version discrepancy (§b.1) — all disclosed above.
