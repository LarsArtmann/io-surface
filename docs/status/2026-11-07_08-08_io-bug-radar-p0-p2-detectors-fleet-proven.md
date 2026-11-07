# 2026-11-07 08-08 — io-bug-radar P0–P2: detector suite shipped, fleet-proven, severity stayed info

**Status:** DONE — ALL phases (P0/P1/P2/P3, tasks T1–T25 / TODO_LIST #65–#89). v0.3.0
released, tagged and published mid-series (commit `336c279`); P3 landed on top:
claims v1.2 (`criticality`/`durability`), severity modulation + `claim-conflict`,
`write-non-atomic` + `tiny-buffer`, pilot onboarding (InboxClean full migration incl.
the pdf.go fd-leak fix + CI pin bump; gogenfilter fresh scaffold + CI wiring),
composition/SARIF ADR (ROADMAP Theme 5, TODO #61 decided), docs sync.
**Verifies:** `docs/planning/2026-11-07_05-02_io-bug-radar-pareto-plan.md` §4 layer table.

## What shipped

- **Observation channel (D1/D3):** `internal/extract/observations.go` — `Observation`,
  `ExtractWithObservations`; plain `Extract` byte-identical (pinned by
  `TestExtractWithObservations_SitesIdentical`); token-tier sites structurally never reach
  a detector (only the resolved scanFunc branch attaches a `sel`).
  Verified how: `go test ./internal/extract/`.
- **Capture contexts:** parent/loop stack via `ast.Walk` pre/post discipline; shapes
  bare/assign/returned/deferred/expression with positional blank slots; `ctxParam` via
  signature scan. Verified how: fixture shape table tests.
- **Nine detector rules** (`internal/extract/detectors.go`, tables in `tables.go`):
  err-dropped, closer-missing (+http body variant), ctx-dropped, defer-in-loop,
  client-no-timeout (bare helper + literal), stat-then-open, io-in-loop, tiny-buffer and
  write-non-atomic pending P3. Registry↔metadata parity pinned by test (D7).
- **`lint` face (D5):** `internal/lint` + `io-surface lint` command; manifestless, exit 0
  unless `--enforce` and error findings exist; text/json/sarif reuse check.Report+emit.
  Verified how: `TestExecute_LintFace` (runApp e2e), `TestGolden_LintOutput` byte-pins
  text + SARIF (`cmd/io-surface/testdata/lint-*.golden`).
- **Fixtures:** `testdata/detectors/` second miniature tree (bug-shaped code OUT of the
  manifest contract); scaffold round-trip now also asserts severity-clean lint
  (`scaffold_test.go`: zero errors/warnings on a fresh scaffold).
- **Self-dogfood (T10):** `io-surface lint --root .` → 6 info hits, 0 errors. One REAL
  fix: `internal/check/location.go` `symbolVisible` re-read the same file per declared
  location → per-call read memoization. 5 accepted hits (dirs-loop N=2, per-file write
  loop, deliberate stat-guard) recorded below. Verified how: re-run lint post-fix.
- **Fleet harness (T16):** `scripts/fleetscan.sh` (pma discovery → per-repo lint JSON →
  aggregate). One command reproduces the run:
  `./scripts/fleetscan.sh /tmp/fleetscan-final.json`.

## Fleet evidence (2026-11-07, final numbers after carve-outs)

Verified how: `./scripts/fleetscan.sh /tmp/fleetscan-final.json` → 160/160 repos scanned,
72 with findings.

| Rule                | Fleet | Pilot-verified | FP classes found (evidence-acted)                                              |
| ------------------- | ----- | -------------- | ------------------------------------------------------------------------------ |
| `io-in-loop`        | 349   | 3 (0 true)     | retry loops, per-item collection I/O — intended shapes; info forever candidate |
| `stat-then-open`    | 50    | 1 (0 true)     | stat-as-cache-validity-probe then read — intended pattern                      |
| `client-no-timeout` | 13    | 4 (3 true)     | ctx-bounded caller (client sans Timeout but every call ctx-capped)             |
| `err-dropped`       | 11    | 9 (0 true)     | → carved out: `io.ReadAll` capped-LimitReader (9/9), `filepath.Glob` (14/27),  |
|                     |       |                | `rand.Read` (stdlib never-errors). Remaining: os.Remove cleanup class, WalkDir |
|                     |       |                | partial-walk (real-ish), os.ReadFile blank (real)                              |
| `ctx-dropped`       | 3     | 3 (FP class)   | force-kill/cleanup commands (TASKKILL) — ctx-blind is arguably correct         |
| `defer-in-loop`     | 1     | 1 (1 TRUE)     | — learnings `internal/github/client.go:96`: pagination loop defers Body.Close  |

**Confirmed real bugs found by the suite (pilot repos, not touched from here):**

1. InboxClean `internal/papersync/pdf.go:92` — `outFile` (os.CreateTemp) is removed in a
   defer but **never closed**: one leaked fd per decrypt. Fix belongs to InboxClean.
2. io-surface itself: the symbolVisible N+1 (fixed in T10, this repo).

## T17 severity verdict (evidence-gated, D4)

| Rule              | Verdict     | Why                                                           |
| ----------------- | ----------- | ------------------------------------------------------------- |
| err-dropped       | keep `info` | post-carve FP profile healthy but sample small; promote later |
| closer-missing    | keep `info` | 1 real bug, 1 sample — not FP≈0 proof yet                     |
| client-no-timeout | keep `info` | 3/4 true = 25% FP — promotion bar is FP≈0, not met            |
| ctx-dropped       | keep `info` | force-kill FP class documented                                |
| defer-in-loop     | keep `info` | n=1 fleet-wide                                                |
| stat-then-open    | keep `info` | 50 hits, sampled hit was intended pattern                     |
| io-in-loop        | keep `info` | 0/3 sampled actionable; same-path refinement is a v0.4 idea   |

**No rule promoted in v0.3.0.** The self-gate stays green trivially because nothing
ships above info (D4 held).

## Known gaps / next tasks

- dnsblockd manifest still v1-era (old schema + disciplines.cue): its 131 hand-written
  judgment claims need a deliberate v1→v1.2 migration (owner call — the constraint
  failures are real and the claims context is human judgment; NOT mass-migrated here).
- io-in-loop same-path refinement (loop body × printer-keyed expr) — v0.4 candidate.
- err-dropped os.Remove-cleanup class: either a `//nolint`-style suppression story or
  acceptance; decide with more fleet data.
- InboxClean's 21 new token-tier advisory sites (post-refresh) are undeclared-by-design
  (weak-site infos); declaring or baselining them is InboxClean's own follow-up.
