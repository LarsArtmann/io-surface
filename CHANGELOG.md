# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
