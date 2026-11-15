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
scripts/self-gate.sh                    # THE WHOLE ci.yml test job locally — run before pushing
go build ./...                          # build
go vet ./...                            # vet
go test ./...                           # tests (fast, hermetic)
gofmt -l .                              # must print NOTHING (CI gate, whole tree incl. testdata)
go run ./cmd/io-surface check --root .  # SELF-HOSTING GATE (CI runs exactly this)
go run ./cmd/io-surface render --root . # render must not crash (CI checks)
go run ./cmd/io-surface lint --root .   # detector radar (report-only; CI emits SARIF artifact)
```

CI = the `self-gate.sh` stages above (`.github/workflows/ci.yml`) plus the
pinned docs-gates battery (`ci.yml:111`) and non-blocking `capabilities`/
`public-install` jobs. That is the whole quality bar. No `.golangci.yml` and
CI does not run golangci-lint (deliberate; waiver in `docs/errcheck-policy.md`);
`dprint.json` covers non-Go files, unenforced. The `.gitignore` has a
`>>> buildflow-managed >>>` block — never hand-edit inside those markers.

## Remotes & publish

`origin` = Forgejo (primary, private), `github` = `LarsArtmann/io-surface`
(public mirror — `go install …@latest` needs it). Branches track `origin`;
CI runs on GitHub pushes only. Until the Forgejo push mirror exists (needs
an admin token not on this machine), every publish pushes BOTH remotes; the
auto-commit daemon creates commits, it never pushes.

## Self-hosting: editing Go code updates the gate contract

The committed manifest (`docs/architecture/io-surface/*.cue`) is a tree
snapshot. Touching `.go` under `cmd/`/`internal/`: added an I/O call →
`undeclared-site` (error, CI red) — declare it; moved lines →
`location-drift` (info, does NOT fail CI). Refresh facts with
`go run ./cmd/io-surface init --root . --force` — refreshes ONLY `facts.cue`
(guarded by `TestWrite_ForceRefreshesFactsOnly`). Release protocol:
`docs/release-protocol.md` (0.x SemVer; version const in `cmd/io-surface/main.go`).

## Architecture

Data flow: `cmd/io-surface` (flag dispatch) → `internal/extract` (AST walk) →
`internal/scaffold` (`init`) or `internal/check` (gate) → `internal/render`.

| Package               | Role                                                                                                                                          |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/io-surface`      | cmdguard CLI, typed flag structs, exit codes 0/1/2 (`exitCode` in `main.go`), version const.                                                    |
| `internal/extract`    | Hermetic AST walker (NO `go/packages` — Nix-sandbox-safe, `GOPROXY=off`); detectors: `observations.go` + `detectors.go` (10 rules), tables in `tables.go`. |
| `internal/manifest`   | In-process CUE load (no `cue` binary); non-empty `violations` fails unification; sentinels `ErrNotInitialized`/`ErrEmpty`.                       |
| `internal/check`      | The bidirectional gate: 14 finding rules, exact-ID pairing first then canonical (`id-drift`).                                                   |
| `internal/match`      | Receiver-insensitive ID canonicalization (`pkg.Type.Method` ≡ `pkg.Method`).                                                                   |
| `internal/scaffold`   | `init` writer; `schema.cue` embedded — edit `internal/scaffold/schema.cue`, not deployed copies.                                                |
| `internal/render`     | Claims-aware markdown renderer; BEGIN/END doc embedding with `--check`.                                                                        |
| `internal/absorb`     | Pinned Capslock JSON → `capabilities.cue` (its ONLY writer); `--check` = advisory drift, never fails.                                           |
| `internal/capability` | Reachability vocabulary: name validity, capability→family mapping, per-package sets.                                                           |
| `internal/emit`       | Findings → go-finding model (SARIF); rule IDs/severities map 1:1, confidence 1.0.                                                              |
| `internal/lint`       | Manifestless radar face; claim modulation when a manifest exists; exit 0 unless `--enforce`.                                                   |

### Where detection rules live

Everything the extractor "knows" is hardcoded tables in
`internal/extract/tables.go`: `pkgOf`/`importSubstring`, `callFamily`
(**resolved** tier, can fail builds), `methodToken` (**token** tier,
advisory only), `seamIdent` (built-in seams; manifest `via` values override),
`familyPrecedence`. Teaching a new call = edit these tables + add a fixture
case. Type-fact resolution is out of scope BY DESIGN: such sites get
`facts.extractable: false`.

### Invariants that must stay in sync

- 10 families / 7 disciplines exist in THREE places: `internal/scaffold/schema.cue`
  closed enums, `Families`/`Disciplines` vars in `internal/manifest`, README
  prose. A new family = schema version bump.
- `manifest.Dir` is the ONE manifest-dir constant; default scan dirs exist
  ONCE (`extract.DefaultDirs()`).
- Schema is v1.2; the deployed copy (`docs/architecture/io-surface/schema.cue`)
  stays byte-identical to `internal/scaffold/schema.cue` (hand-updated —
  `init --force` never touches it); pilots own their copies.
- Capability vocabulary: `#CapabilityName` enum in `schema.cue` + the closed
  mapping in `internal/capability` (single source). New names = schema bump +
  ROADMAP Q5. `capabilities.cue` written ONLY by `absorb`; the gate never runs
  the engine. Capslock in CI installs with the repo's own Go toolchain.
- Rule constants in `internal/check/check.go` are a stable contract (SARIF
  ruleIds, suppression keys) — renaming is breaking.
- Extracted IDs/locations are **repo/root-relative**, fixture-verified.
- `testdata/` is formatter-frozen: treefmt excludes `testdata/**` (root-relative
  globs only — `**/testdata/**` does not work in treefmt v2.6) while CI's
  `gofmt -l .` stays whole-tree; a mangled fixture is restored from git blobs.
- Check semantics (pairing order, severity tiers, scan-dir rules, the
  facts-freshness `-I` filters, baseline lifecycle): `docs/check-semantics.md`.
  Load-bearing: only resolved-tier extraction fails builds; the CI `-I`
  filters on `commit:`/`date:` are structural — do not remove them.

## Testing patterns

- `testdata/extract/` is a fixed miniature tree shared by extract and check
  tests; sites referenced by repo-relative ID with pinned line numbers.
- `testdata/detectors/` is the SECOND tree: bug-shaped, OUT of the manifest
  contract, never compiles (parseable is the only requirement). Lint output
  is byte-pinned by goldens (`cmd/io-surface/testdata/lint-*.golden`);
  intentional changes re-run `go test ./cmd/io-surface -run TestGolden -update`.
- `internal/scaffold`'s test is the round-trip proof: fixture → extract →
  scaffold → `manifest.Load` → `check.Run` → `render.Markdown`, green by
  construction.
- `cmd/io-surface` has e2e tests (`main_test.go`): every test drives the real
  cmdguard execution via `runApp` with a swappable `stdout` sink, plus the
  exitCode 0/1/2 table. Table-driven stdlib tests only, no mocks (hermetic).

## Sharp edges (current constraints)

- **BuildFlow repair tools rewrite files mid-run; the daemon commits after
  gate builds.** A live `buildflow --fix` loop can revert uncommitted files
  mid-edit — re-read before editing when one is running (`pgrep -af
  buildflow`) and verify edits landed on disk. Mid-run gate reds can vanish
  once the run settles; re-run before acting.
- A STALE system `buildflow`/`docs-gates` binary produces FALSE gate reds.
  Fix recipe: rebuild in the BuildFlow repo (`nix build .# && nix run
  .#reinstall`), refresh shims; `buildflow doctor` names the delta.
- A `nix-checker` "vendorHash may be stale" finding can be mtime noise —
  prove with `nix build .#`. Step ERROR findings can persist in BuildFlow's
  result cache after a fix — re-verify with `BUILDFLOW_NO_RESULT_CACHE=1`.
- docs-gates: BuildFlow findings name INTERNAL gate names ("docs gate
  audit-all is red"), not CLI verbs. Cadence-red ritual: verify rows
  struck/routed, prepend a `> **Live remainder:**` banner, `git mv` to
  `docs/status/archived/`.
- Baseline suppression is canonical-key based; hiding same-base-name sites
  behind one entry is by design (`internal/match/collision_test.go`).
- Unexported-type dotted IDs are lexically indistinguishable from
  dotted-subpackage mistakes — `id-form` confirms the slash variant against
  the extracted set.
- Never revert in-flight changes you didn't author (concurrent sessions are
  real — 2026-11-07 and 2026-11-15 both saw foreign edits mid-session).
- Fleet evidence (FP classes, T17 verdicts, pilot retro) lives in
  `docs/status/archived/2026-11-07_08-08_io-bug-radar-*.md`; reproduce via
  `./scripts/fleetscan.sh` — outputs go to `docs/evidence/` (gitignored:
  absolute paths + private project names; provenance contract in
  `docs/evidence/README.md`), never /tmp, never committed.

## Conventions

- Package doc comments are load-bearing (why hermetic, why tiers, why
  pairing order) — extend them when changing design rationale.
- Go code: gofmt-clean, tabs, stdlib-first. Errors wrap context; sentinels
  live in per-package `errors.go`.
- errcheck policy: report writes are deliberately unchecked (`_, _ =`);
  rationale in `docs/errcheck-policy.md`. Don't "fix" those sites or add
  linter config without reopening the decision.
- Manifest editing: `facts.cue` is mechanical (regenerate, never hand-edit
  locations); `claims.cue`/`baseline.cue` are human judgment (never
  overwrite). `schema.cue` source of truth is `internal/scaffold/schema.cue`.
- `cue` binary is never needed at runtime or tests. CHANGELOG follows Keep a
  Changelog.
- Every health/report claim names its verifying command — a number without
  the command that produced it is unverified and says so.
- Docs ownership — `README` users / `AGENTS` agent context / `FEATURES`
  inventory / `TODO_LIST` open work / `ROADMAP` vision — one fact, one home.
- Library utilization audit: `docs/review/libs/` (2026-11-07, 22 house
  libraries; adoption triggers + re-audit bar in its index).
