# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

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
