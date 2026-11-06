# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, use ROADMAP.md.
> Items are ranked by impact. Status is verified, not assumed.
>
> Harvested from `docs/status/2026-11-06_00-14_io-surface-birth-bench-matrix-and-first-pilot.md` (§f P0-P4, §b, §c) and re-verified against code on 2026-11-06. Items already shipped at harvest time were dropped (LICENSE is MIT, AGENTS.md exists, summary line already prints resolved/token counts).

## Status legend

| Status           | Meaning                                                     |
| ---------------- | ----------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                   |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                   |
| 🔵 `BLOCKED`     | Cannot proceed, external dependency or decision needed.     |
| 🟢 `DONE`        | Completed. Remove from this list and log in `CHANGELOG.md`. |

## High Impact

| #  | Task                                                                                                    | Status    | Impact | Effort | Evidence                                                                                  |
| -- | ------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------- |
| 2  | `check --json` always emits JSON, incl. validation failures (as findings with a `validation` rule)      | 🔴 `TODO` | High   | 1h     | `cmd/io-surface/main.go:125-131` prints text to stderr, exits 1, empty stdout; report §d7 |
| 3  | Print actual `violations` entries when CUE unification fails (not "incompatible list lengths")          | 🔴 `TODO` | High   | 1-2h   | `internal/manifest/manifest.go:131-135`; report §d4                                       |
| 4  | `audit.commit` = `git rev-parse --short HEAD` (+ `-dirty`), fallback `working-tree`                     | 🔴 `TODO` | Med    | 30min  | `internal/scaffold/scaffold.go:78-79` writes the literal; report §d8                      |
| 5  | `render --check` + BEGIN/END generated blocks (doc ↔ manifest drift gate)                               | 🔴 `TODO` | High   | 3-4h   | `internal/render/render.go` (full-doc renderer only); README "Status" promises it next    |
| 6  | CI: regenerate facts and fail on drift (freshness gate against the committed snapshot)                  | 🔴 `TODO` | Med    | 1h     | `.github/workflows/ci.yml` has self-check/self-render but no facts-regen diff             |
| 7  | Baseline "only shrinks": fail when it grows beyond the committed count unless a flag allows it          | 🔴 `TODO` | Med    | 1-2h   | Rule exists only as comments: `internal/scaffold/schema.cue:90`, `scaffold.go:159`        |
| 8  | Baseline entries gain `owner` + `expires`; an expired entry fails                                       | 🔴 `TODO` | Med    | 1-2h   | `#BaselineEntry` has only id/note/date: `internal/scaffold/schema.cue:92-97`              |
| 9  | Hint when declared sites fall outside `--dirs` (18 silent findings on InboxClean)                       | 🔴 `TODO` | Med    | 30min  | `cmd/io-surface/main.go:83-97` parses dirs; no scope hint in `check.Run`; report §b9      |
| 10 | Golden test for `render.Markdown` (multiple families, claims, scope, empty states)                      | 🔴 `TODO` | Med    | 1h     | No `_test.go` in `internal/render/`; report §d9/P0-10                                     |
| 11 | `cmd/io-surface` tests: exit codes 0/1/2, flag parsing, `--json` shape                                  | 🔴 `TODO` | Med    | 2h     | No tests in `cmd/io-surface/`; report §d9/P0-11                                           |
| 12 | `manifest` package tests: decode edge cases + planted constraint violation                              | 🔴 `TODO` | Med    | 1h     | No tests in `internal/manifest/`; report §d9/P0-12                                        |
| 13 | `render` package tests: claims columns, judgments section, scope OUT                                    | 🔴 `TODO` | Med    | 1h     | No tests in `internal/render/`; report §d9/P0-13                                          |
| 14 | Record the enforcement policy (D4) in the manifest so a repo cannot silently opt out                    | 🔴 `TODO` | Med    | 2h     | Report P0-5; no policy field in `schema.cue`                                              |

## Medium Impact

| #  | Task                                                                                                                                                                  | Status       | Impact | Effort | Evidence                                                                                                      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ------------------------------------------------------------------------------------------------------------- |
| 15 | Gate io-surface itself outside CI (pre-commit hook or `bin/dev`)                                                                                                      | 🔴 `TODO`    | Med    | 30min  | CI gates it, local runs do not; no `.pre-commit-config.yaml`; report P1-16                                    |
| 16 | Pin `gofmt`/`go vet` locally (pre-commit), not only in CI                                                                                                             | 🔴 `TODO`    | Med    | 30min  | `.github/workflows/ci.yml` only; report P0-14                                                                 |
| 17 | SARIF emission via `go-finding` (suppression files + LSP diagnostics for free)                                                                                        | 🔴 `TODO`    | Med    | 2-3h   | Report P1-22; README "Status" names SARIF next                                                                |
| 18 | BuildFlow provider via `go-finding/toolsdk` so `./bin/dev` runs `io-surface check`                                                                                    | 🔴 `TODO`    | Med    | 2-3h   | Report P1-21; no provider code in repo                                                                        |
| 19 | Add missing extraction families: `os.Stdin/Stdout/Stderr`, `syscall.Mmap`, `net.ListenConfig`, `bufio.NewReader/NewWriter`, `http.ServeContent` (`filepath.Walk/WalkDir/Glob` landed 2026-11-06) | 🔴 `TODO`    | Med    | 1h     | Tables lack them: `internal/extract/extract.go:156-202`; report P3-34                                         |
| 20 | Derive seams from the manifest's `via` values instead of the hardcoded `seamIdent` table                                                                              | 🔴 `TODO`    | Med    | 2h     | `internal/extract/extract.go:215-225`; report P3-35                                                           |
| 21 | Cross-package seam support (a `pkg/` helper owning the raw call)                                                                                                      | 🔴 `TODO`    | Med    | 2-3h   | `symbolID` is same-package only: `internal/extract/extract.go:327-336`; P3-36                                 |
| 22 | `io-surface baseline add <id> --note <text> --date <d>` (stop hand-editing CUE)                                                                                       | 🔴 `TODO`    | Med    | 1h     | No such subcommand: `cmd/io-surface/main.go:33-48`; report P4-45                                              |
| 23 | `version: "io-surface/v1"` field inside the manifest; bump on any schema change                                                                                       | 🔴 `TODO`    | Med    | 30min  | No version field: `internal/scaffold/schema.cue`; report P4-46                                                |
| 24 | `--ignore` globs for generated code (`*_templ.go`, `zz_generated*`)                                                                                                   | 🔴 `TODO`    | Med    | 1h     | Walk skips only dot-dirs/vendor/node_modules: `internal/extract/extract.go:233-255`; P4-49                    |
| 25 | Gate InboxClean: commit `docs/architecture/` + a check in its CI (cross-repo)                                                                                         | 🔴 `TODO`    | Med    | 30min  | Report §b4/P1-15 — manifest exists there but is ungated/untracked                                             |
| 26 | Declare or baseline dnsblockd's 5 errors, get it to 0 errors (cross-repo)                                                                                             | 🔴 `TODO`    | Med    | 1h     | Report §a5/P1-18 (LeaseCache.Leases, bodyLimitMiddleware, randomCookieToken, views.CSS, WriteTempFileWithDir) |
| 27 | dnsblockd: reverse mode in `check-io-surface.sh` + Nix check; retire `tools/io-extract` (cross-repo, after session lock)                                              | 🔵 `BLOCKED` | Med    | 2h     | Report §b1/P1-17+20 — repo held by `fleet-exec-20261105` lock                                                 |

## Low Impact

| #  | Task                                                                                                 | Status       | Impact | Effort | Evidence                                                                  |
| -- | ---------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ------------------------------------------------------------------------- |
| 28 | Reconsider bare `.Do`/`.Close` tokens — classify as ambiguous rather than a family                   | 🔴 `TODO`    | Low    | 30min  | `"Do": "outbound"` in `internal/extract/extract.go:206`; P3-37            |
| 29 | `extract --stats`: per-package histogram to spot outlier packages                                    | 🔴 `TODO`    | Low    | 1h     | No stats flag: `cmd/io-surface/main.go:231-249`; P3-38                    |
| 30 | `io-surface version --json` with commit + build date                                                 | 🔴 `TODO`    | Low    | 30min  | `version` prints plain text: `cmd/io-surface/main.go:42-43`; P4-44        |
| 31 | Document the taxonomy freeze rule (a new family = a schema version bump)                             | 🔴 `TODO`    | Low    | 15min  | Not documented anywhere; report P4-47                                     |
| 32 | Document ID migration for package moves (path change = a new ID set)                                 | 🔴 `TODO`    | Low    | 15min  | Not documented anywhere; report P4-48                                     |
| 33 | Register the repo in the fleet inventory (`meta` / stack-selection)                                  | 🔴 `TODO`    | Low    | 30min  | Report P4-43; repo currently invisible to own tooling                     |
| 34 | Run the checker under `-race` and on a very large repo; record results                               | 🔴 `TODO`    | Low    | 1h     | Never measured; report §c-18                                              |
| 35 | Run against crush-daily + ~10 daemon repos; publish a real family distribution                       | 🔴 `TODO`    | Low    | 2h     | Report P4-50                                                              |
| 36 | Tag `v0.1.0` (version const, README Status and CHANGELOG all say 0.1.0; no git tag)                  | 🔴 `TODO`    | Low    | 15min  | `git tag` is empty; `cmd/io-surface/main.go:25`                           |
| 37 | Smoke-test `go install github.com/larsartmann/io-surface/cmd/io-surface@latest` via proxy.golang.org | 🔴 `TODO`    | High   | 10min  | Public install path never executed; session report 2026-11-06 §d2         |
| 38 | Check pkg.go.dev indexes the module once the proxy caches it                                         | 🔴 `TODO`    | Low    | 10min  | Follows from #37                                                          |
| 39 | Handle the open Dependabot PR (actions group bump): review + merge or close                          | 🔴 `TODO`    | Med    | 15min  | Open since publishing; title-read only; session report §c3                |
| 40 | Provision a Forgejo API token on this machine (unlocks #41/#42 + fleet automation)                   | 🔵 `BLOCKED` | Med    | 15min  | No token found (env/netrc/tea/credentials searched); session report §g-Q1 |
| 41 | Configure server-side push mirror Forgejo → GitHub (end manual dual-push)                            | 🔵 `BLOCKED` | Med    | 15min  | `push_mirrors` API → 403 anon; gogenfilter has one; blocked by #40        |
| 42 | Flip Forgejo repo visibility per owner decision (gogenfilter is public; io-surface is private)       | 🔵 `BLOCKED` | Low    | 5min   | Anon API → 404; blocked by #40; session report §g-Q2                      |
| 43 | Wire `docs-gates` into CI as a standing gate, pinned to a rev                                        | 🔴 `TODO`    | Med    | 1h     | One-shot PASS only; session report §b4                                    |
| 44 | Add a CI job that installs the module from the public path (module-path tripwire)                    | 🔴 `TODO`    | Med    | 30min  | Session report §g-Q3                                                      |
| 45 | Add "verified how" discipline to future health reports (name the command per claim)                  | 🔴 `TODO`    | Low    | 15min  | Session report §d2                                                        |
| 46 | Seed `.docs-gates.json` thresholds for this repo                                                     | 🔴 `TODO`    | Low    | 15min  | Defaults used; session report §f-46                                       |
| 47 | Re-derive TODO_LIST evidence line numbers at the next docs audit (they rot)                          | 🔴 `TODO`    | Low    | 15min  | Inherent to the format; session report §e5                                |
| 48 | Consider `go-release` flow for v0.1.1 once P0 items #1-4 land                                        | 🔴 `TODO`    | Low    | 30min  | go-release skill; session report §f-48                                    |

---

<!--
  Guidance for the builder filling this in:  (template guidance retained)
  - Source of truth is the CODE. Verify each item before adding, many
    documented TODOs are already done.
  - DONE items should be REMOVED, not kept. Use CHANGELOG.md for history.
  - Cite evidence (file:line) so the next person can verify without re-deriving.
  - Long-term / unrefined ideas live in ROADMAP.md, not here.
-->
