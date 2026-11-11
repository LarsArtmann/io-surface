# AGENTS.md

`io-surface` is the I/O surface manifest gate: a Go CLI that extracts every I/O
call site from a Go tree via a stdlib-only AST walk, compares it against a CUE
manifest, and fails the build on I/O nobody documented — AND (since v0.3.0,
io-bug-radar) a detector layer that finds bad I/O via `io-surface lint`
(manifestless, exit 0 unless `--enforce`; rules ship at info, promotion is
evidence-gated). Single Go module (`go 1.27`), four direct dependencies
(`cuelang.org/go` for the manifest, `cmdguard/v5` for the CLI, `go-finding`
for SARIF, `go-atomic-write` for the tool's own file writes — the gate core
`internal/extract` itself stays stdlib-only). `flake.nix` (fleet
`go-standard` module, go 1.27, linux-only): `nix run .# -- check --root .`
is the sandbox-native gate. No Makefile/justfile.

`README.md` documents the tool for users. This file documents the repo for
agents working ON the tool.

## Commands

```bash
go build ./...                          # build
go vet ./...                            # vet
go test ./...                           # tests (fast, hermetic)
go test -cover ./...                    # with coverage
gofmt -l .                              # must print NOTHING (CI gate, whole tree incl. testdata)
go run ./cmd/io-surface check --root .  # SELF-HOSTING GATE (CI runs exactly this)
go run ./cmd/io-surface render --root . # render must not crash (CI checks)
go run ./cmd/io-surface lint --root .   # detector radar (report-only; CI emits SARIF artifact)
```

CI (`.github/workflows/ci.yml`): build → vet → test → gofmt gate → self-check
→ facts-freshness (`init --force` + filtered `git diff`) → self-render →
inventory-doc freshness (`render --doc … --check`) → SARIF emit → lint-radar
SARIF (artifact-only, TODO #61 policy — advisory until D4 promotion). That
is the whole quality bar. `CONTRIBUTING.md` mentions `golangci-lint` but no
`.golangci.yml` exists and CI does not run it; `dprint.json` covers non-Go
files, unenforced. The `.gitignore` has a `>>> buildflow-managed >>>` block
— never hand-edit inside those markers; `buildflow --fix` regenerates it.

## Remotes & mirrors

`origin` = `ssh://forgejo@forgejo.home.lan/lars/io-surface.git` (primary),
`github` = `git@github.com:LarsArtmann/io-surface.git` (public mirror).
Branches track `origin`; `master` is default; CI runs on GitHub pushes only.
Until the Forgejo push mirror is configured (server UI, needs an API token
not on this machine), every publish pushes BOTH remotes — the auto-commit
daemon creates commits, it does not push. GitHub stays public deliberately
(`go install …@latest` requires it); Forgejo defaults private.

## Releases

Protocol, provider coupling, tag/proxy/verify steps, and the blocked Capslock
upstream issue live in `docs/release-protocol.md`. Releases are 0.x SemVer
(features = MINOR); version const in `cmd/io-surface/main.go`.

## Self-hosting: editing Go code updates the gate contract

This repo is gated by its own tool. The committed manifest
(`docs/architecture/io-surface/{facts,claims,baseline,schema}.cue`) is a
snapshot of the tree. Consequences of touching any `.go` file under `cmd/`
or `internal/`:

- Added an I/O call → next `check` reports `undeclared-site` (error) → CI
  fails. Declare it in `facts.cue` (or `claims.cue`).
- Moved lines around declared sites → `location-drift` (info only, does NOT
  fail CI).

Refresh facts after an I/O-adding change: `go run ./cmd/io-surface init
--root . --force` — safe since 2026-11-06: refreshes ONLY `facts.cue`
(judgment/standard files are skipped when they exist — guarded by
`TestWrite_ForceRefreshesFactsOnly`).

## Architecture

Data flow: `cmd/io-surface` (flag dispatch) → `internal/extract` (AST walk) →
either `internal/scaffold` (`init`) or `internal/check` (gate) →
`internal/render` (markdown).

| Package               | Role |
| --------------------- | ---- |
| `cmd/io-surface`      | cmdguard-wired CLI: `NewApp` + batteries, typed flag structs (`commands.go`), handlers keep the exact output contract. Exit codes: 0 clean, 1 gate findings (`errFindings`), 2 usage/internal (`exitCode` in `main.go`). Version const lives here. |
| `internal/extract`    | Hermetic AST walker (`go/parser` + `ast.Inspect`, deliberately NO `go/packages`, no network) so it runs in a Nix sandbox with `GOPROXY=off`. Plus the detector layer: `observations.go` (Observation channel + capture contexts) and `detectors.go` (8 rules, registry + metadata in one table); detection tables live in `tables.go`. `Extract` stays byte-identical to pre-detector behavior — `ExtractWithObservations` is the observing entrypoint (D1). |
| `internal/manifest`   | Loads the CUE package in-process (no `cue` binary). `Load` doubles as constraint enforcement: non-empty `violations` fails unification. Sentinels: `ErrNotInitialized`, `ErrEmpty`. |
| `internal/check`      | The bidirectional gate. 14 rules (incl. lifecycle + scope-hint + shadow-capability); pairs exact IDs first, then 1:1 canonical leftovers (= `id-drift`). |
| `internal/match`      | Receiver-insensitive ID canonicalization (`internal/server.Server.Save` ≡ `internal/server.Save`). |
| `internal/scaffold`   | `init` writer; `schema.cue` embedded via `go:embed` — edit `internal/scaffold/schema.cue`, not deployed copies. |
| `internal/render`     | Claims-aware markdown renderer (facts from manifest, prose from claims, verbatim). |
| `internal/absorb`     | Pinned Capslock JSON → `capabilities.cue` (its ONLY writer); `absorb --check` reports advisory drift (`io-surface/capability-drift`), never fails the gate. |
| `internal/capability` | Reachability vocabulary: name validity, capability→family mapping (`FamiliesFor`), engine edges folded into per-package sets. |
| `internal/emit`       | Findings → go-finding model for consumer formats (SARIF via `--format sarif`); rule IDs/severities map 1:1, confidence 1.0. |
| `internal/lint`       | Manifestless face: `lint.Run` = `ExtractWithObservations` → `check.Report` (Enforcement "lint"). Claim modulation when a manifest exists (`criticality` escalates, `durability: ephemeral` drops write-non-atomic). Exit 0 unless `--enforce`. |

### Where detection rules live

Everything the extractor "knows" is hardcoded tables in
`internal/extract/extract.go`: `pkgOf`/`importSubstring` (import path →
classification key), `callFamily` (package-qualified calls → family,
**resolved** tier, can fail builds), `methodToken` (name heuristics on
unresolvable receivers, **token** tier, advisory only), `seamIdent`
(same-package wrapper identifiers), `familyPrecedence`. Teaching the
extractor a new call = edit these tables + add a fixture case. Type-fact
resolution is out of scope BY DESIGN: such sites get
`facts.extractable: false`.

### Invariants that must stay in sync

- The 10 families and 7 disciplines exist in THREE places: the CUE closed
  enums in `internal/scaffold/schema.cue`, the `Families`/`Disciplines` vars
  in `internal/manifest/manifest.go`, and README prose. A new family is a
  schema version bump.
- `manifest.Dir` is the ONE manifest-dir constant (scaffold and the
  BuildFlow provider join it — no hardcoded copies). Default scan dirs exist
  ONCE: `extract.DefaultDirs()` (`check.effectiveDirs`, `init`'s empty-scan
  error, fleetscan fallback read or mirror it).
- Schema is v1.2: `claims.criticality` and `claims.durability`. The deployed
  copy (`docs/architecture/io-surface/schema.cue`) stays byte-identical to
  `internal/scaffold/schema.cue` (updated by hand — `init --force` never
  touches it); pilots own their copies (InboxClean v1.2, dnsblockd v1,
  migration pending owner decision).
- Capability vocabulary: `#CapabilityName` CUE enum in `schema.cue` + closed
  enum/mapping in `internal/capability` (the single-source mapping — rules
  and absorb derive from it). New capability names = schema version bump +
  ROADMAP Q5 decision.
- `capabilities.cue` is written ONLY by `absorb`; the hermetic gate never
  runs the reachability engine (`capability-drift` is emitted by
  `absorb --check`, not `check`). Capslock in CI installs with the repo's
  own Go toolchain (a version-skewed analyzer fails to type-check stdlib).
- Rule constants in `internal/check/check.go` are a stable contract (SARIF
  ruleIds, suppression keys) — renaming is breaking.
- Extracted IDs and locations are **repo/root-relative** (relative to
  `--root`), fixture-verified — the manifest contract keys on this.
- `testdata/` is formatter-frozen: `flake.nix` excludes `testdata/**` from
  treefmt (gofumpt/goimports) because goimports prunes the fixture's
  unresolvable seam import (breaking `TestExtract_CrossPackageSeam`) and
  rewrites the embed import (shifting line-pinned locations). CI's
  `gofmt -l .` stays whole-tree. treefmt v2.6 global excludes match
  root-relative globs: `testdata/**` works, `**/testdata/**` does not.

### Check semantics worth remembering

Pairing order, severity tiers, scan-dir rules, the facts-freshness `-I`
filters, and baseline lifecycle live in `docs/check-semantics.md` (relocated
2026-11-11 for the 220-line cap). The two most load-bearing: only
resolved-tier extraction fails builds; the CI facts-freshness `git diff -I`
filters on `commit:`/`date:` are structural, do not remove them.

## Testing patterns

- `testdata/extract/` is a fixed miniature tree (`cmd/app`,
  `internal/store`) shared by extract and check tests; sites are referenced
  by repo-relative ID with pinned line numbers; determinism asserted.
- `testdata/detectors/` is the SECOND miniature tree: deliberately
  bug-shaped, OUT of the manifest contract, never compiles (testdata is
  invisible to the go tool — parseable is the only requirement). Lint
  output is byte-pinned by goldens (`cmd/io-surface/testdata/lint-*.golden`);
  intentional changes re-run
  `go test ./cmd/io-surface -run TestGolden -update`.
- `internal/scaffold`'s test is the round-trip proof: copy fixture → extract
  → scaffold → `manifest.Load` → `check.Run` → `render.Markdown`, green by
  construction.
- `cmd/io-surface` has e2e tests (`main_test.go`): every test drives the
  real cmdguard execution via `runApp` with a swappable `stdout` sink, plus
  the exitCode 0/1/2 table. **No direct tests** for `internal/manifest` and
  `internal/render` — known gap, tracked in TODO_LIST. Table-driven stdlib
  tests only, no mocks (hermetic). A mangled fixture is restored from git
  blobs (`git cat-file blob <hash> > path`), never hand-reformatted.

## Sharp edges (current constraints)

- **BuildFlow repair tools rewrite files mid-run and the daemon commits
  after gate builds**: `go-fix` reorders imports, inserts aliases
  (`atomicwrite "github.com/larsartmann/go-atomic-write"`), and HAS mangled
  `testdata/` fixtures (2026-11-11: seam import pruned, embed import added —
  3 tests red until restored from blobs). Gate reds from mid-run snapshots
  can vanish once the run settles — re-run builds before acting on them.
- A STALE system `buildflow`/`docs-gates` binary produces FALSE gate reds.
  Signature seen 2026-11-11: `go-mod-parity` step failing with
  `wait for go-mod-parity: exit status 1` — the old binary's probes executed
  a literal `go-mod-parity` binary (fixed in BuildFlow `8936ca0a2`, probes
  now use real `git`/`go`). Fix: rebuild from the repo
  (`nix build .# && nix run .#reinstall` in BuildFlow) and refresh the
  `~/.local/bin` shims (they resolve BEFORE the system profile);
  `buildflow doctor` names the binary-freshness delta.
- A `nix-checker` "vendorHash may be stale" finding can be mtime-heuristic
  noise (go.sum rewritten with unchanged content). Prove validity with
  `nix build .#` before touching the hash.
- docs-gates: BuildFlow findings name INTERNAL gate names ("docs gate
  audit-all is red"), not CLI verbs (`run`/`strike`/`purge`). Archive ritual
  when cadence is red: verify rows struck/routed, prepend a
  `> **Live remainder:**` banner, `git mv` to `docs/status/archived/`.
- Baseline suppression is canonical-key based; the hiding is by design
  (`internal/match/collision_test.go`).
- The extractor's unexported-type dotted IDs are lexically indistinguishable
  from dotted-subpackage mistakes — the `id-form` rule confirms the slash
  variant against the extracted set instead of guessing; a declared dotted
  ID with NO extraction at all classifies stale/extraction-blind as before.
- Multiple agent sessions may work this repo concurrently: never revert
  in-flight changes you didn't author. A running `buildflow --fix` loop can
  revert uncommitted working-tree files mid-edit (2026-11-07) — when one is
  live (`pgrep -af buildflow`), re-read files before editing and verify
  edits landed on disk.
- Fleet evidence: FP classes, T17 severity verdicts (nothing promoted — D4
  bar is FP≈0) and pilot retro live in
  `docs/status/archived/2026-11-07_08-08_io-bug-radar-*.md`; reproduce via
  `./scripts/fleetscan.sh docs/evidence/fleetscan-<date>.json` (outputs in
  `docs/evidence/`, never /tmp).

## Conventions

- Package doc comments are load-bearing (why hermetic, why tiers, why
  pairing order) — extend them when changing design rationale.
- Go code: gofmt-clean, tabs, stdlib-first. Errors wrap context
  (`fmt.Errorf("…: %w", err)`); sentinels live in per-package `errors.go`.
- errcheck policy (TODO #123): report writes are deliberately unchecked
  (`_, _ =`); full rationale and scope in `docs/errcheck-policy.md`. No
  `.golangci.yml` by design — do not "fix" the `_, _ =` sites or add config
  without reopening the decision.
- Manifest editing: `facts.cue` is mechanical (regenerate, never hand-edit
  locations); `claims.cue`/`baseline.cue` are human judgment (never
  overwrite). `schema.cue` source of truth is `internal/scaffold/schema.cue`.
- `cue` binary is never needed at runtime or for tests; `cue fmt` is only a
  hand-editing habit.
- CHANGELOG follows Keep a Changelog; version const in
  `cmd/io-surface/main.go`.
- Every health/report claim names its verifying command — a number without
  the command that produced it is unverified and says so.
- Docs ownership: `README` (users) / `AGENTS` (agent context) / `FEATURES`
  (status inventory) / `TODO_LIST` (open work) / `ROADMAP` (vision) — one
  fact, one home.

## Library utilization audit (2026-11-07)

`docs/review/libs/` holds the point-in-time audit of 22 LarsArtmann
libraries: cmdguard and go-atomic-write are the only direct house deps
(at/near ceiling); go-error-family/go-output ride the graph via
go-finding/cmdguard, deliberately unimported. Adoption triggers on record
(`check --watch` via go-filewatcher/v2 if pilots ask; go-error-family only
for a networked mode). Re-audit when a new direct dep lands or a pinned dep
falls 2+ versions behind.
