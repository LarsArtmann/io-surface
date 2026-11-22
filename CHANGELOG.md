# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `render --lint`: the lint radar embeds into the inventory document as an
  appendix inside the generated block — one doc carries both the gate
  inventory and the detector findings, and `--doc --check` freshness covers
  both faces. `--check` must repeat the writing invocation's flags: a check
  that does not render the appendix flags it as drift, by design. The
  repository's own INVENTORY.md self-hosts the appendix and CI checks it.
- `extract.Severity`: detector rule severities are a named type (an
  off-vocabulary registry entry is now a compile error), with a
  registry-validation test walking every published rule.
- `check --strict-weak` (opt-in, ROADMAP Q3 decision 2026-11-08): escalates
  the `weak-site` finding (an undeclared token-tier/heuristic I/O site) from
  info to error, so a repo can adopt teeth for heuristic extraction without
  changing the default — without the flag the weak tier stays advisory
  forever.
- `baseline add --manifest`: the baseline command now accepts the same
  `--manifest` flag as every other manifest-reading command instead of being
  hardwired to the default directory.
- `fleetscan.sh` rescans a repo with `--dirs .` when neither `cmd/` nor
  `internal/` exists, so repos whose packages live at the root no longer
  scan as a silent zero (the project-discovery-sdk artifact, 2026-11-11);
  the aggregate records the scanned dirs per repo.

### Changed

- **CLI failure surfaces are coded go-error-family rejections** (owner
  directive: ALL LarsArtmann/* SDKs are to be used where useful): the
  init/baseline-add/absorb/render failures now carry a behavioral family and
  a machine-readable code (`init.no-sites`, `cli.usage`, `capability.type`,
  `render.doc-block`) and render as `[Rejection:code] message`; callers and
  tests classify with `errors.Is` (code+family) and `errorfamily.Code`. The
  0/1/2 exit contract is unchanged. `go-error-family` moved from indirect to
  direct dependency.
- **`--format` is the canonical output flag** (owner decision, TODO #154):
  every consumer and fleet script uses `--format json|sarif|github`; the
  `--json` shorthand stays functional and documented for interactive use,
  never removed (removal would churn callers for zero gain). README wording
  now leads with `--format`.
- `lint` findings now print strongest-first (severity rank, then rule, then
  ID) — the same presentation contract as `check`, so the text, JSON and
  SARIF faces of both commands order findings identically. The lint goldens
  are re-pinned to the new order (deliberate, reviewed).
- SARIF: location-less gate findings (`missing-location`, `scope-hint`,
  baseline lifecycle, `manifest-validation`) anchor at the manifest file
  where the fix happens (`facts.cue`, or `baseline.cue` for baseline rules)
  instead of a pseudo-file named `io-surface`; a SARIF consumer's click now
  lands somewhere real. The lint face keeps the tool pseudo-file for its
  defensive empty-location case (no current detector produces one).

- TODO_LIST purge (2026-11-21): `check --json` always emits JSON, incl. validation failures (as findings with a `validation` rule) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Print actual `violations` entries when CUE unification fails (not "incompatible list lengths") — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `audit.commit` = `git rev-parse --short HEAD` (+ `-dirty`), fallback `working-tree` — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `render --check` + BEGIN/END generated blocks (doc ↔ manifest drift gate) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): CI: regenerate facts and fail on drift (freshness gate against the committed snapshot) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Baseline "only shrinks": fail when it grows beyond the committed count unless a flag allows it — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Baseline entries gain `owner` + `expires`; an expired entry fails — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Hint when declared sites fall outside `--dirs` (18 silent findings on InboxClean) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Golden test for `render.Markdown` (multiple families, claims, scope, empty states) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `cmd/io-surface` tests: exit codes 0/1/2, flag parsing, `--json` shape — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `manifest` package tests: decode edge cases + planted constraint violation — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `render` package tests: claims columns, judgments section, scope OUT — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Record the enforcement policy (D4) in the manifest so a repo cannot silently opt out — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Gate io-surface outside CI (self-hosted pre-commit) (pre-commit hook or `bin/dev`) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Pin `gofmt`/`go vet` locally (pre-commit), not only in CI — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): SARIF emission via `go-finding` (suppression files + LSP diagnostics for free) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): BuildFlow provider via `go-finding/toolsdk` so `./bin/dev` runs `io-surface check` — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Add missing extraction families: `os.Stdin/Stdout/Stderr`, `net.ListenConfig` (`syscall.Mmap/Munmap`, `bufio` constructors, `http.ServeContent/ServeFile` landed 2026-11-06 with a table-reachability invariant test) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Derive seams from the manifest's `via` values instead of the hardcoded `seamIdent` table — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Cross-package seam support (a `pkg/` helper owning the raw call) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `io-surface baseline add <id> --note <text> --date <d>` (stop hand-editing CUE) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `version: "io-surface/v1"` field inside the manifest; bump on any schema change — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `--ignore` globs for generated code (`*_templ.go`, `zz_generated*`) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Gate InboxClean: commit `docs/architecture/` + a check in its CI (cross-repo) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Declare or baseline dnsblockd's 5 errors, get it to 0 errors (cross-repo) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Reconsider bare `.Do`/`.Close` tokens — classify as ambiguous rather than a family — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `extract --stats`: per-package histogram to spot outlier packages — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `io-surface version --json` with commit + build date — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Document the taxonomy freeze rule (a new family = a schema version bump) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Document ID migration for package moves (path change = a new ID set) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Register the repo in the fleet inventory (`meta` / stack-selection) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Run the checker under `-race` and on a very large repo; record results — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Run against crush-daily + ~10 daemon repos; publish a real family distribution — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Tag `v0.1.0` (version const, README Status and CHANGELOG all say 0.1.0; no git tag) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Smoke-test `go install github.com/larsartmann/io-surface/cmd/io-surface@latest` via proxy.golang.org — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Check pkg.go.dev indexes the module once the proxy caches it — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Handle the open Dependabot PR (actions group bump): review + merge or close — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Wire `docs-gates` into CI as a standing gate, pinned to a rev — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Add a CI job that installs the module from the public path (module-path tripwire) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Add "verified how" discipline to future health reports (name the command per claim) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Seed `.docs-gates.json` thresholds for this repo — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Re-derive TODO_LIST evidence line numbers at the next docs audit (they rot) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Consider `go-release` flow for v0.1.1 once P0 items #1-4 land — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Decide whether stdout/stderr writes belong in `scope.in` (currently implicit out-of-scope) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Fixture: unexported-type method site in `testdata/extract` (permanent id-form regression guard) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Unit tests for renderer `displayWidth` wide-range + narrow-column (min-3) separator paths — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Pin the cmdguard `describe` machine-readable contract with a test — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `manifest.Load` decode errors name the offending file (Dir-anchored context) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `check --strict-dirs`: fail instead of hint when declared sites fall outside `--dirs` — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `--allow-unused-baseline` grace flag before promoting unused-baseline to error — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): README: document the `init --force` + git-restore ritual for judgment files — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): CI: dprint-parity step proving INVENTORY.md survives a formatter pass — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): id-drift single-dotted pairing warning should name the correct slash form (like id-form does) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): `constraintError`: one deterministic conjunct walk instead of two overlapping dedupes — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Decide SARIF → GitHub code-scanning upload policy vs artifact-only — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): Index `docs/status/` reports (`docs/status/README.md` pattern, like dnsblockd) — DONE 2026-11-06
- TODO_LIST purge (2026-11-21): Split the oversized core files to fleet-default 350 lines (extract tables/pairing/handlers) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): T1 Design: observation channel, detector registry, `lint` exit policy — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T2 Plumbing: `extract.Observation` + `ExtractWithObservations` + AST capture contexts — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T3 `err-dropped` rule + deliberate-ignore escape (`_ =` ok, `x, _ :=` flagged) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T4 Detector fixture tree + text/SARIF goldens + round-trip assert — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T5 `lint` command (no manifest, exit 0 unless `--enforce`) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T6 `closer-missing` (ctor table + defer/Close search + escape suppression) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T7 `closer-missing` http variant (`resp.Body` close duty) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T8 `ctx-dropped` (variant-pair table + ctx-in-scope) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T9 `ctx-dropped` fixtures/goldens — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T10 Dogfood: `lint` on io-surface; fix/baseline every hit before any severity promotion — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T11 Pilot FP sampling on 5 hotspot repos; per-rule go/no-go — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T12 `defer-in-loop` — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T13 `client-no-timeout` (Client literal sans Timeout; bare http.Get/Post) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T14 `stat-then-open` (identical-path-expr TOCTOU/double-syscall) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T15 `io-in-loop` (info only, N+1 shape) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T16 Fleet harness: committed 182-repo runner + FP aggregates — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T17 Severity promotion from evidence (thresholds, verdicts, promote, re-prove self-gate) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T18 Release v0.3.0 (CHANGELOG, README, tag both remotes, proxy check, BuildFlow pickup) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T19 Claims v1.2: `criticality`/`durability` schema (both schema.cue copies byte-identical) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T20 Severity modulation (claims x observations) + `claim-conflict` rule — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T21 `write-non-atomic` (temp+rename pattern + manifest-`via` seam whitelist) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T22 `tiny-buffer` (small Read buffer in loop) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T23 Pilot onboarding: init + claims + green gate + CI in 2-3 repos — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T24 Spike: golangci focus filter vs audit module vs `--precise` boundary + SARIF upload policy (→ #61) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T25 Docs sync: TODO_LIST/FEATURES/ROADMAP after suite lands — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): Expose io-bug-radar through the BuildFlow provider (gate + radar merged; manifestless = radar-only at info, never a step error) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): CI self-lint SARIF artifact step (implements the #61 artifact-only policy) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): Triage the self-lint infos: 6 of 11 ANSWERED by the go-atomic-write adoption (5 write-non-atomic + 1 stat-then-open gone); 4 io-in-loop remain (symbolVisible post-fix residual, 2 extract stat pairs, scaffold existence guard) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): SARIF: surface the enforcement policy as a SARIF property (emit it from `internal/emit` so consumers see gate vs audit posture per run) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): T1 Promotion policy v1 (thresholds, de-promotion, ratification) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T2 Fleet re-scan with the final ruleset; evidence report regenerated — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T3 FP verification protocol: 30× write-non-atomic/tiny-buffer/client-no-timeout — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T4 Evidence hygiene: docs/evidence/ home; /tmp citations fixed — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T5 Suppression mechanism: `//io-surface:ignore <rule> <reason>` — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T6 Self-lint CI step (report-only) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T7 CHANGELOG Unreleased fill (P3 features) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T8 Rule cards: README table (detects/FP classes/examples) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T9 Pilot SARIF artifact steps (gogenfilter + InboxClean CI) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T10 Mutation corpus + `TestMutationRecall` (recall % per rule, in CI) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T11 TODO #3: per-site validation error messages — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T12 go-taskqueue triage (37 findings) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T13 write-non-atomic/tiny-buffer semantics decisions from measured data — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T14 `lint --format github` (PR annotations) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T15 Contract polish: describe lint flags + SARIF ruleDescription — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T16 io-in-loop same-path refinement — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T17 FP-class decisions via suppression (os.Remove, force-kill) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T18 Ownership claims v1.3 design spike → ROADMAP Q6 ADR — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T19 dnsblockd migration runbook (inventory + mapping template; NO migration) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T20 gogenfilter claims port (5 sites + criticality) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T21 InboxClean review bundle (diff summary + rationale; NO commit) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T22 Benchmark lint on the biggest fleet repo (display-only) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T23 Pilot retro #2 prep (behavior metric + artifact instrumentation) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T24 Docs sync (AGENTS lint row, Theme 2 link, --enforce doc, errors-help) — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): T25 Wrap: status report v3 + cadence check + push — DONE 2026-11-07
- TODO_LIST purge (2026-11-21): Same seam test for applyDoc (render doc-fingerprint capture) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Manifest `version` field: unknown-major refusal (Load/Validate rejects versions beyond the shipped contract) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Owner ratifications pending: promotion-policy numbers (policy §8) + ownership v1.3 scope (ROADMAP Q6) + InboxClean CI pin/SARIF step — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Bump `cmdguard` → v4.2.1 (later migrated to v5) + `go-atomic-write` → v0.6.2; full gates; refresh audit reports' version-currency rows — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Proxy-check latest versions for the 16 local-tag-only audit libs; resolve go-credentials-store v0.2.1-tag vs README-v0.3.1 — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Audit discoverability: README pointer to `docs/review/libs/`, screenshot link in index footer, README claims vs v0.4.0 — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Run full local CI battery (render, inventory-doc --check, SARIF emit, facts-freshness) — convert "gate green" to "CI green" — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): go-finding + go-finding/toolsdk utilization deep-dives in the `docs/review/libs/` series (+ index 22→24 rows) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): errcheck decision for the 8 `commands.go` stdout-sink warnings: adopt minimal config + fix, or documented waiver — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Version currency: `cuelang.org/go` v0.17.1 vs latest + toolsdk v1.16.0/go-finding v1.15.1 skew check — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Verify BuildFlow picked up provider v0.4.0 (sdk_imports pin; run one detect end-to-end) — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Series sweep: `docs/reviews/` + `docs/research/` overlap check + supersedes convention for the audit series — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Audit rendering checks: mobile check (render index.html at 320 px; only 1280 px was screenshotted) + visual pass of the 22 .md report files on GitHub (tables/badges render) — DONE 2026-11-21 — 320px audit done 2026-11-21: 3 reports contained, scrollWidth 305 ≤ 320
- TODO_LIST purge (2026-11-21): fleetscan silent zero-scan: repos without `cmd/`+`internal/` scan as a clean 0 (project-discovery-sdk artifact) — DONE 2026-11-11
- TODO_LIST purge (2026-11-21): `emit.positionOf`: an empty location anchors SARIF at the pseudo-file `io-surface` (RuleMissingLocation) — decide the honest shape — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): lint face prints findings in walk order; check sorts strongest-first — align lint's presentation (re-pins lint goldens deliberately) — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): `extract.RuleDef.Severity` is a raw string: a registry typo silently degrades to info in `lint.severityOf` — make it a typed enum — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): Push the 13 unpushed master commits to BOTH remotes (origin + github); confirm CI triggered on the new HEAD — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Cut v0.5.0: version const + CHANGELOG heading + README status line + full gates + tags to both remotes + proxy check — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): M14 dnsblockd migration decision packet from the T19 runbook + session schedule; map unblock path for #27 — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): M15 baseline shrink: inspect the 2 baselined sites; declare or remove; shrink-or-justify — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): M20 ROADMAP Q5 decision input: capability promotion evidence summary → owner decision → ROADMAP update — DONE 2026-11-15 — M20 evidence delivered 2026-11-15; decision leg pre-satisfied 2026-11-08
- TODO_LIST purge (2026-11-21): InboxClean: run on-demand gates their hook skipped (nix flake check/lint/test) + push decision for commit `418ff69d` — DONE 2026-11-08
- TODO_LIST purge (2026-11-21): Ask the N19 Q1–Q4 owner batch (first BLOCKING gate repo, flake timing, weak-tier `--strict-weak` policy, binary-blob purge) and record the outcomes in ROADMAP Q1–Q4 — DONE — asked+recorded 2026-11-08: ROADMAP Q1–Q4
- TODO_LIST purge (2026-11-21): v1.3 session-shape decision owed by owner: one combined schema-v1.3 train carrying runtime-power capabilities (#136) AND claims.ownership (#135), or two separate sessions — DONE — decided 2026-11-08: one combined v1.3 train→#136
- TODO_LIST purge (2026-11-21): dnsblockd owner diff review of the mechanical v1.2 port (92 merged / 42 dropped keys / 21 UNREVIEWED fills) + resolve the 3 `claim-conflict` warnings the gate surfaces (loadTrialBlocklist, processor.Run, processor.writeMapping) — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): go-taskqueue claims judgment hour: fill 230 empty claims blocks (resource/trigger; durability on write-family sites) + triage the 20 write-non-atomic radar findings (real-bug-potential batch) — DONE 2026-11-15 — claims hour done 2026-11-15: 232/232 claimed, triage routed
- TODO_LIST purge (2026-11-21): Push go-taskqueue onboarding commit `5d223a42` (repo is push-owner-gated; a concurrent session's v0.4.2 work was in-flight) — DONE — pushed 2026-11-08 21:06: rebased the 51-commit local line onto origin/master (concurrent fuzz commit 4d6250ac landed first), pushed through `aa5ab099` after the push-timing hook's 600s freshness guard; their CI: gate step PASSED, 2 failures are pre-existing v0.4.2 debt (missing internal/retryx/v0.4.2 tag + docsgates#4 windows test)
- TODO_LIST purge (2026-11-21): Exercise the flake's test check: `nix flake check` (the enableTestCheck derivation exists but has NEVER run) + one full `buildflow` battery post-flake (flake-pin-drift now triggers on flake.nix) — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): Root-cause the go-taskqueue BuildFlow fan-out count: 53 findings via `-s io-surface` vs 52 via root CLI — one leaf module re-counts the harvest site; advisory only but the evidence chain should explain the +1 — DONE 2026-11-15 — fan-out count RCA done 2026-11-15: 54==54 verified, no double-count at v0.5.1
- TODO_LIST purge (2026-11-21): dnsblockd MIGRATION-v1.2.md §5: append the 3 live `claim-conflict` warnings (loadTrialBlocklist atomic-write-on-outbound, processor.Run + processor.writeMapping buffered-write-on-stream-copy) so the review deck matches the gate output — DONE 2026-11-15 — claim-conflict warnings dissolved by #139 overlay, verified 0 today
- TODO_LIST purge (2026-11-21): Purge follow-through: confirm old GitHub Releases (v0.2.0/v0.4.0/v0.5.0) source archives resolve on rewritten tags; confirm proxy `@master` serves a post-purge zip; backup DECIDED+DELETED 2026-11-08 (owner chose delete; mirror trashed) — DONE 2026-11-21 — purge follow-through verified 2026-11-21: tags+archives+proxy all resolve
- TODO_LIST purge (2026-11-21): GitHub billing blast-radius audit (SUPERSEDED-BY-#155 for the fix; the audit survives as the post-migration inventory): after the billing failure is fixed, enumerate which fleet repos/jobs were silently blocked (InboxClean's Nix Flake Check + Secret Scanning never started 2026-11-08: "recent account payments have failed") — DONE 2026-11-21 — audit-as-of-2026-11-21 recorded: InboxClean jobs still scheduler-blocked, fix rides #155
- TODO_LIST purge (2026-11-21): cmd-level e2e test for `--strict-weak` flag parsing/exit code (check-level `TestRun_StrictWeakEscalates` exists; the `--strict-dirs` precedent also lacks a cmd test) — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): Sweep BuildFlow's io-surface provider pin to v0.5.1 (its go.mod requires an older published provider tag; both faces ship through that loop) — DONE 2026-11-15 — sweep verified 2026-11-15: BuildFlow go.mod @ v0.5.1, commit 4e5def7b9
- TODO_LIST purge (2026-11-21): W3 promotion re-scan per docs/planning/2026-11-07_promotion-policy.md §9 (detector evidence re-derivation at the next release train; feeds the D4 evaluation) — DONE 2026-11-15 — W3 re-scan done 2026-11-15: evidence committed, verdict NO PROMOTIONS
- TODO_LIST purge (2026-11-21): AGENTS.md slim to ≤220 lines (fleet budget; ~335 now, flagged by buildflow preflight every run) — move detail to README/docs per the preflight fix hint — DONE 2026-11-15 — slimmed 2026-11-15: commit a8889d8
- TODO_LIST purge (2026-11-21): `render --lint` appendix option: render the lint-findings report as an appendix inside the inventory doc — one doc, both faces (gate + radar) — DONE 2026-11-15 — render --lint shipped 2026-11-15: design + implement + self-hosted + CI-wired
- TODO_LIST purge (2026-11-21): Decide `--json` vs `--format` consolidation now that SARIF has shipped and stabilized (current: `--json` is shorthand for `--format json`; README documents both) — keep the shorthand or deprecate it — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): `scripts/self-gate.sh` mirroring ci.yml's test job as ONE local command (facts re-check after last edit → self-check → self-render → inventory-doc --check → gofmt-empty) — the pre-push ritual that would have caught both of this session's CI breaks — DONE 2026-11-15 — self-gate shipped 2026-11-15: 12-step mirror verified green + failure path probed
- TODO_LIST purge (2026-11-21): Set up renovate/dependabot (or a CI pin-bump check) for the `go run …@pin` module-path lines — the docs-gates pin in ci.yml (line 111) is hand-maintained and will rot silently — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): Living-docs VERIFY + polish pass (README/FEATURES/ROADMAP/AGENTS/CHANGELOG content vs code) — the 2026-11-15 session polished crush-config's docs but only gate-fixed this repo's; the SUPERB bar is unverified here — DONE 2026-11-15 — verified 2026-11-15: commit 7d9e97d
- TODO_LIST purge (2026-11-21): CONTRIBUTING.md staleness + docs-pointer existence check — add a CI/doctor check that AGENTS.md-referenced docs (docs/release-protocol.md, docs/check-semantics.md, docs/errcheck-policy.md) actually exist — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): Rebuild `result` (`nix build .# -o result`) to clear the vulnix stale-target warning + record the result-dir policy (gitignored artifact vs checked pattern) in AGENTS.md or the plan errata — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): Give the archived reports per-report tailored `> **Live remainder:**` banners where missing (docs-health ANNOTATE rigor; the cadence ritual's banner step was skipped for some of the 28) — DONE 2026-11-15
- TODO_LIST purge (2026-11-21): Report-level golden for the `check` text output (today only lint has byte goldens: `lint-text.golden` pins the lint face, while `check` prints its prose untested — a presentation regression in printReport would ship silently) — DONE 2026-11-21 — check-text golden shipped 2026-11-21
- TODO_LIST purge (2026-11-21): docs-gates binary freshness tripwire: CI is pin-fresh by construction (`docs-gates@v0.3.1` via `go run`), but LOCAL runs have two entry points — `nix run github:…docs-gates` (flake HEAD) and the system `/run/current-system/sw/bin/docs-gates` — either can lag the pin and redden a gate the pinned rev passes; candidate shape: self-gate.sh already reads the pin from ci.yml, extend it (or doctor) to verify the invoked docs-gates version string matches the pin before running the battery — DONE 2026-11-21 — tripwire shipped 2026-11-21: both local entry points exposed off-pin
- TODO_LIST purge (2026-11-21): Push io-surface master to the GitHub public mirror: Forgejo origin is current (pushed 2026-11-15..21 by a later session) but github/master lags 44 commits waiting — CI has NEVER seen the render --lint CI-step change (`freshness` step now `--lint --check`), the lint-SARIF artifact step, or the markdown render audit step; after push, verify the GitHub Actions run green end to end (2026-11-21 session, owner ordered the whole list done) — DONE 2026-11-21 — pushed + CI green 2026-11-21: 47-commit wait ended, run 44925881521 all green

### Fixed

- `check` report ordering is now strongest-first (error, warning, info).
  The sort previously compared severity strings lexically, which buries
  warnings under infos (`"error" < "info" < "warning"`).
- `--format` validates its value: an unknown format (e.g. `--format yaml`)
  errors with the valid set instead of silently rendering text.
- `init`'s "no I/O sites found" error names the dirs actually scanned
  (respecting `--dirs`) instead of always claiming `cmd`/`internal`, and its
  success output goes through the same output sink as every other command.
- The BuildFlow provider reads the manifest directory from
  `manifest.Dir` (the single source) instead of a hardcoded copy.
- `absorb`'s CUE rendering escapes string values, so an `--engine-version`
  containing a quote can no longer produce invalid CUE.

## [0.5.1] - 2026-11-08

### Fixed

- Purged the 35 MB build binary the auto-commit daemon once committed from
  git history (`git filter-repo` over commits `05874f3`/`e15bf62`; a fresh
  clone drops from ~54 MB to ~4 MB).

### Retracted

- v0.2.0, v0.3.0, v0.4.0, v0.5.0 — published before the history purge, so
  their module zips permanently contain the 35 MB binary (proxy caches are
  immutable). `@latest` resolves here; explicit pins of the old versions
  keep resolving, with a retraction warning.

## [0.5.0] - 2026-11-08

### Added

- **Detector severity modulation by claims (claims v1.2).** Claims grow two
  typed fields — `criticality` (`normal`/`critical`) and `durability`
  (`durable`/`ephemeral`) — with CUE-enforced vocabularies. `lint` escalates
  findings one severity step on `critical` sites, and weighs `durability` for
  `write-non-atomic`: a `durable` claim escalates it, an `ephemeral` claim
  drops it (a temp file tearing at crash time is declared harmless).
- **`claim-conflict` gate rule:** a manifest claim that contradicts the
  extracted facts (family mismatch) is an error finding, not a silent lie.
- **Two new detector rules complete the P3 line:** `write-non-atomic` (a
  direct final-path write without temp+rename — a crash mid-write tears the
  file) and `tiny-buffer` (a sub-page buffer feeding a read/write helper —
  syscall amplification). The shipped ruleset is now 10 rules.
- **Suppression directive `//io-surface:ignore <rule> <reason>`:** silence a
  finding at the site, one rule at a time, reason mandatory, visible in the
  review diff. Malformed directives (unknown rule, missing reason) suppress
  nothing and emit their own `ignore-invalid` finding — a typo can never
  quietly eat a real bug. File-level ignores are rejected by design (use the
  baseline for that).
- **Every file write the tool performs is crash-safe.** Scaffolding
  (`init`), `baseline.cue` appends, `capabilities.cue`, rendered `--out`
  files and the doc-embed path now write through `go-atomic-write`
  (unique temp file, fsync, atomic rename). The two read-modify-write
  paths (baseline append, doc embed) additionally fingerprint-verify the
  file: a concurrent writer now fails the command with a re-run hint
  instead of silently clobbering the file.
- The extractor resolves `github.com/larsartmann/go-atomic-write` calls
  at the resolved tier (`Write*`/`WriteFunc*` = `file-write`,
  `FingerprintFile` = `file-read`), so repos adopting the library keep
  full gate coverage and its sites never fire `write-non-atomic`.
- **Manifest contract-version refusal (TODO #116).** A manifest declaring a
  `version` outside the shipped set (`io-surface/v1|v1.1|v1.2`) fails to load
  with `ErrUnsupportedVersion` naming the supported set and the upgrade path —
  instead of silently mis-decoding sites written for a newer contract. Absent
  stays legal (v1 predated the field). The CUE schema enum already refuses
  unknown versions when a current `schema.cue` is present; the Go-side guard
  covers stale or missing schema copies.
- **SARIF enforcement-posture property (TODO #93).** Every SARIF result
  carries `go-finding/meta/enforcement` (`gate`, `audit`, or lint-face
  `lint`), so consumers can tell which face produced the artifact without
  re-running the tool.
- **`net.ListenConfig` var-ref extraction (TODO #19).** The modern listen
  path (`var lc net.ListenConfig; lc.Listen(ctx, net, addr)`) now extracts at
  the token tier with the `listener` family hint — advisory by construction
  (a lexical var-ref tracker cannot see shadowing, so it must never be able
  to fail a build). `os.Stdin/Stdout/Stderr` stay deliberately out of scope:
  process stdio is user feedback, not durable state, and the scaffolded
  `scope.out` documents the exclusion explicitly.

## [0.4.0] - 2026-11-07

### Added

- **The BuildFlow provider runs both faces.** `buildflow -s io-surface` now
  reports the io-bug-radar detector sweep alongside the manifest gate.
  Adopted modules get gate and radar findings merged under one enforcement
  policy (audit demotion covers both faces, including radar findings a
  critical claim escalated); manifestless modules — the bulk of the fleet —
  get the advisory radar at info severity, never a step error. The one-shot
  182-repo fleet scan becomes continuous D4 promotion evidence on every
  BuildFlow run.
- CI: self-lint SARIF artifact (`io-surface lint --format sarif`), implementing
  the artifact-only upload policy — advisory, no enforcement until a rule is
  promoted on fleet false-positive evidence.

### Changed

- Provider tool description names both faces (visible in
  `buildflow explain io-surface`); rule IDs and severities still map 1:1 to
  the CLI vocabulary, so BuildFlow suppressions and SARIF consumers are
  unaffected.

## [0.3.0] - 2026-11-07

### Added

- **io-bug-radar: the detector layer (plan
  `docs/planning/2026-11-07_05-02_io-bug-radar-pareto-plan.md`, evidence
  `docs/status/2026-11-07_08-08_io-bug-radar-p0-p2-detectors-fleet-proven.md`).**
  io-surface no longer only finds *undocumented* I/O — it now finds *bad* I/O.
  Eight detector rules ship on top of the existing hermetic extraction:
  `err-dropped` (discarded error returns; deliberate `_ =` ignores respected),
  `closer-missing` (closable constructors never closed or handed off, plus the
  `resp.Body` variant), `ctx-dropped` (context-blind variant while a
  context.Context is in scope), `defer-in-loop`, `client-no-timeout` (bare
  http helpers and `http.Client` literals sans Timeout), `stat-then-open`
  (identical path expr stat'd then opened: double syscall + TOCTOU),
  `io-in-loop` (N+1 shape), `tiny-buffer` pending P3. Rules run on the
  resolved tier only (D3) and ship at `info` — severity promotion is
  evidence-gated and none met the FP≈0 bar for this line (T17 verdict table
  in the status report).
- **`io-surface lint` — the manifestless face (D5):** detector findings for
  any Go tree, no manifest required. Exit 0 unless `--enforce` and
  error-severity findings exist; `--format text|json|sarif` reuses the gate's
  report shape and SARIF pipeline; rule IDs are additive and stable
  (`io-surface/err-dropped`, …). Fleet-proven: `scripts/fleetscan.sh` ran the
  lint face over 160 pma-discovered repos (72 with findings) and the pilot
  sampling surfaced one confirmed fd-leak bug in InboxClean plus one real N+1
  in io-surface itself (fixed).
- **Observation channel:** `extract.ExtractWithObservations` returns detector
  observations alongside sites that stay byte-identical to plain `Extract`
  (D1: the manifest facts contract is untouched — every committed facts.cue
  keeps working unchanged).

### Changed

- Evidence-based err-dropped carve-outs from the fleet runs: `io.ReadAll`
  (capped-LimitReader best-effort reads were 9/9 sampled FPs), `filepath.Glob`
  (error fires only on malformed patterns; 14/27 sampled FPs), and
  `crypto/rand.Read` (documented never to error). `check.symbolVisible` now
  memoizes per-path reads (N+1 fix found by the tool's own lint face).

## [0.2.0] - 2026-11-06

### Changed

- The CLI is now built with **cmdguard v4** (`github.com/larsartmann/cmdguard/v4`):
  commands and flags are typed structs, help and errors render through
  cmdguard/fang (actionable hints, typo suggestions), and the auto-registered
  `version`, `doctor`, and hidden `describe` commands join the surface. The
  user-facing contract is unchanged: same commands, same flags, same exit
  codes (`0` clean / `1` gate findings / `2` usage or internal), same
  text/JSON/SARIF output bytes; `io-surface version` still prints
  `io-surface <version>`. New: `io-surface doctor` (DI/config health, trivially
  green) and `io-surface describe` (machine-readable CLI
  contract, JSON).

### Added

- **Unified I/O insights (io-surface/v1.1, advisory):** the manifest gains a
  `capabilities` section — per-package reachability sets (`direct` through the
  package's own code, `deep` including transitive dependency paths) observed by
  a pinned external engine (Google Capslock), plus a `capabilityAudit`
  provenance block and an optional manifest `version` field. The closed
  vocabulary (`FILES`/`NETWORK`/`EXEC`) and the capability→families mapping
  live in exactly one place (`internal/capability`); runtime-power
  capabilities stay out-of-scope (ROADMAP Q5). New `io-surface absorb`
  command: folds a Capslock JSON report into per-package sets and writes
  `capabilities.cue` — that file ONLY (claims, baseline and facts are never
  machine-written); `--check` reports advisory drift
  (`io-surface/capability-drift`) and never exits 1. New gate rule
  `io-surface/shadow-capability` (info): a package whose reachability facts
  name an I/O class no declared site covers. The hermetic gate never runs the
  engine — capabilities enter only through `absorb`. Evidence for the design:
  the v0.3.4 blind-spot run (`docs/status/2026-11-06_06-45_capslock-v034-blind-spot-run.md`)
  — 10/11 real direct sinks were already declared; Capslock false-positive
  classes are why the tier is advisory.
- **Recorded enforcement policy (decision D4):** the manifest gains
  `enforcement: "gate" | "audit"` (absent = `gate`, scaffolded explicitly).
  `audit` reports error findings but exits 0 — the adoption posture — and
  `check --enforce` overrides it for one run. The policy lives IN the
  manifest so weakening the gate is a committed, reviewable diff, never a
  silent CI edit; the JSON/SARIF report carries the policy (`enforcement`
  field) and the human note goes to stderr so machine output stays
  parseable.
- SARIF output: `check --format sarif` emits SARIF 2.1.0 via the house
  `go-finding` library (rule IDs and severities pass through unchanged, so
  GitHub renders them natively); CI uploads the self-hosted SARIF as an
  artifact. `--format json|text` also joins `--json` (kept as shorthand)
- `render --doc FILE` embeds the inventory into hand-written markdown between
  `<!-- io-surface:begin -->` / `<!-- io-surface:end -->` markers, rewriting
  only the generated block; `--check` exits 1 when the block is stale. This
  repository's `docs/architecture/io-surface/INVENTORY.md` is gated by it in
  CI, next to a new facts-freshness gate (`init --force` + diff)
- Golden test for the renderer (`-update` regenerates), determinism and
  empty-manifest tests, and first direct tests for the CLI (JSON contract,
  doc embedding, drift)
- `path/filepath` extraction (`Walk`/`WalkDir`/`Glob` → `file-read`, resolved
  tier): directory traversal and globbing are now visible I/O. The
  self-dogfood audit found the extractor's own `filepath.Walk` invisible to
  the gate (a gate cannot see its own blind spots); it is now taught and
  declared in this repo's manifest
- Memory-map, buffered-stream and file-serving extraction (resolved tier):
  `syscall.Mmap`/`Munmap` → `file-read`, `bufio.NewReader`/`NewReaderSize`/
  `NewWriter`/`NewWriterSize`/`NewReadWriter` → `stream-copy`, and
  `http.ServeContent`/`ServeFile` → `stream-copy` — mmap and buffered
  constructors were real I/O the tables never watched (Capslock blind-spot
  run evidence)
- Table-reachability invariant test: every package key in the extractor's
  call table must be resolvable from an import path (exact map or fragment
  match). It exists because `bufio` was watchable but unreachable — the
  import was dropped before lookup, so the whole rule table was dead code
  and `bufio.NewReader` call sites silently never extracted
- House documentation set: `AGENTS.md` (agent context), `FEATURES.md`,
  `TODO_LIST.md`, `ROADMAP.md` — harvested from the birth status report
  (`docs/status/2026-11-06_00-14_io-surface-birth-bench-matrix-and-first-pilot.md`)
  and verified against code
- Repository publishing: Forgejo primary (`forgejo.home.lan/lars/io-surface`)
  and public GitHub mirror (`github.com/LarsArtmann/io-surface`, CI green)
- Constraint failures now name the violations: when a `disciplines.cue` rule
  fires, the load error lists the actual `id: msg` entries (recovered from
  the CUE conjuncts on "incompatible list lengths" conflicts, checked
  explicitly otherwise) instead of CUE's opaque list-length message; the
  `io-surface/manifest-validation` sentinel is `errors.Is`-matchable
- New gate rule `io-surface/id-form` (error): a declared ID that encodes a
  subpackage with a dot (`internal/server.sso.Manager`, the dnsblockd
  adoption lesson) can never pair with the extractor's slash-package form —
  the gate now names the correct form instead of reporting a generic
  undeclared site, without false-positiving on legal unexported-type IDs
  (`internal/extract.extractor.parseFile` stays legal)
- Baseline lifecycle rules: an entry suppressing nothing today warns
  (`io-surface/unused-baseline` — the only-shrinks rule, enforced), and an
  entry past its `expires` date fails (`io-surface/baseline-expired`,
  error). `#BaselineEntry` gains optional `owner` and `expires` fields —
  additive, backward-compatible: manifests that use them need this schema
  version or newer
- New gate rule `io-surface/scope-hint` (info): a `--dirs` narrowing that
  excludes declared sites now names the flag as the cause instead of
  producing findings that look like manifest rot
- `audit.commit` in freshly scaffolded/refreshed manifests records the real
  short HEAD hash (`-dirty` when the worktree carries changes; the literal
  `working-tree` survives only outside a git repo), and the CI
  facts-freshness gate ignores the audit `commit:`/`date:` lines so HEAD
  drift between refresh and CI cannot structurally redden the gate
- Direct test suite for `internal/manifest` (decode edge cases, sentinel
  errors, planted constraint violations through both the conflict and the
  explicit path) and `auditCommit` git-repo tests for the scaffold
- `io-surface baseline add <id> --note WHY [--date YYYY-MM-DD]`: grandfather
  an undeclared site without hand-editing CUE. The surgical write is the
  machine's only sanctioned insertion into a judgment file (comments and
  formatting preserved); it refuses duplicates and noteless entries and warns
  when the ID does not extract today (an entry suppressing nothing)
- `--ignore` globs (root flag, honored by init/check/extract/render/
  baseline add): skip generated code (`*_templ.go`, `zz_generated*`) — globs
  match repo-relative paths and bare file names, so one pattern covers any
  depth. Generated I/O is not authored I/O; declaring it would be inventory
  noise the generator rewrites anyway
- Manifest-derived seams (#20): `check` learns the repo's wrapper vocabulary
  from declared sites' `facts.via` values — a repo-specific seam
  (dnsblockd's `writeMapping`) surfaces its callers without an extractor
  edit. The hardcoded table shrinks to the built-in default vocabulary that
  keeps first `init` scaffolds useful; seam callers now also record the
  wrapper name in their `via` fact instead of leaving it empty
- Cross-package seams (#21): a caller in ANOTHER package
  (`cmd/app.mirror` calling `store.genWrite`) resolves against the seam
  vocabulary by import-path suffix matching — a two-pass hermetic walk, no
  go/packages, token confidence like every seam attribution
- `--strict-dirs`: the scope-hint gains teeth on demand — a `--dirs`
  narrowing that blinds the gate to declared sites then FAILS (error) instead
  of hinting (info); default stays advisory because narrower scans are often
  deliberate
- `--allow-unused-baseline`: grace flag exempting unused-baseline findings
  from `--strict` failure (they still report) while a team works the baseline
  down — the escape hatch that makes the future unused-baseline promotion
  survivable
- Permanent id-form false-positive guard: the shared fixture gains a method
  on an UNEXPORTED type (`internal/store.ttl.Sweep`) whose dotted ID is
  legal, so the id-form rule's package-dot-only scope can no longer regress
  silently
- Ambiguous `.Do` token: the method-name heuristic records a `.Do` call
  site but claims NO family — http.Client.Do, task runners and DB helpers
  share the name, so the manifest's family is the claim (was: hardcoded
  `outbound`)
- `extract --stats`: per-package histogram (sites, resolved/token split,
  TOTAL) to spot outlier packages in large trees
- `version --json`: {version, commit, date} from the build info (VCS-stamped
  builds carry the real values); the plain one-liner is unchanged
- `baseline add`, `--strict-dirs` and `--allow-unused-baseline` flags,
  `manifest.Load` decode errors anchored to the manifest directory, a
  single-dedupe rewrite of the constraint-violation recovery, describe-
  contract and displayWidth unit tests, and the describe contract now lists
  `baseline add` as a nested subcommand path

### Fixed

- `check --json` now emits a machine-parseable report on stdout for EVERY gate
  outcome, including manifest-validation failures (new rule
  `io-surface/manifest-validation`); previously that path printed text to
  stderr with empty stdout, breaking JSON consumers
- `init --force` refreshes ONLY `facts.cue`: `claims.cue`, `baseline.cue` and
  `schema.cue` survive once present, so human judgment can no longer be
  clobbered by a facts regeneration (guarded by a plant-a-claim test)
- `LICENSE` contradicted the README's MIT claim (a proprietary placeholder had
  landed post-birth); replaced with the standard MIT text, matching the
  birth-session intent and the house pattern for public tools
- A bare `io-surface` line at the top of `.gitignore` (meant for the compiled
  binary) silently swallowed every NEW file under `cmd/io-surface/` and
  `docs/architecture/io-surface/` — the inventory doc and the CLI test file
  went uncommitted past two commits until CI caught it. The pattern is now
  anchored (`/io-surface`), joining the existing line-61 entry
- Generated inventory tables are emitted in the pipe-aligned shape dprint and
  prettier produce, so a formatter pass over an embedded `--doc` block is now
  a byte-level no-op and the `render --check` gate can no longer fail on a
  doc nobody hand-edited

### Removed

- A 35 MB compiled binary (`cmd/io-surface/io-surface`) that the auto-commit
  daemon had committed twice into history is now untracked and gitignored
  (`/cmd/io-surface/io-surface`, `/bin/`; rebuild-proven: a fresh build no
  longer appears as a pending change). Purging the ~67 MB of historical blobs
  vs. keeping them is an owner decision (ROADMAP Q4); tagged `v0.1.0` predates
  the blobs and is unaffected.

## [0.1.0] - 2026-11-05

Initial release: the I/O surface manifest gate (io-surface/v1).

### Added

- Hermetic stdlib-only AST extractor (`internal/extract`): resolved tier
  (import-qualified calls), token tier (method-name heuristics, advisory),
  seam attribution, `//go:embed` detection, `net.Resolver` Dial-field funcs
- Bidirectional check gate (`internal/check`): 8 stable rules, exact-first
  then canonical pairing, severity model where only `error` fails
- In-process CUE manifest loading (`internal/manifest`) with constraint
  enforcement at load time — no `cue` binary needed
- Scaffolder (`internal/scaffold`): `init` writes the four-file CUE package,
  green by construction
- Claims-aware markdown renderer (`internal/render`)
- CLI `check` / `init` / `render` / `extract` / `version` with the 0/1/2
  exit-code contract
- 18 tests (scaffold 92.6%, check 91.0%, match 85.4%, extract 80.2% coverage);
  proven against dnsblockd (hand-ported 131-site manifest), InboxClean
  (from-scratch 126-site pilot) and self-hosting
- GitHub Actions CI: build → vet → test → gofmt gate → self-check →
  self-render
