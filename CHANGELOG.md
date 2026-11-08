# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Nothing yet.

### Fixed

- Nothing yet.

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
