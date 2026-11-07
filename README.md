# io-surface

**The I/O surface manifest gate.** Declare every place your Go program touches
files, databases, sockets, the network, entropy, subprocesses and env — then let a
gate fail the build the moment code touches I/O nobody documented.

```console
$ io-surface check
error   io-surface/undeclared-site  internal/health.LoadSecret  internal/health/probe.go:6
        undeclared I/O site (os.ReadFile); declare it in facts/claims or add it to baseline with a note
io-surface: 126 declared, 127 extracted (42 resolved, 85 token), 0 baselined — 1 errors, 0 warnings, 0 infos
```

Most documentation gates verify that a doc does not lie. They cannot see a doc that
is **silent**: a new `os.ReadFile` in a handler passes every one of them. This gate
checks both directions — every declared site still exists, and every extracted I/O
site is declared.

---

## Why

An I/O inventory is three useful things wearing one coat:

1. **A completeness gate.** `os.CreateTemp`, `net.Listen`, `http.Get`, `sql.Open` —
   if it is not in the manifest, the build fails with the site, the family and the
   line.
2. **An I/O BOM.** A machine-readable answer to "what files does this write, what
   does it listen on, where does it dial, where does it read entropy" — threat
   modelling as a query instead of grep archaeology, and a direct input to
   `ReadWritePaths=` / `ProtectSystem=strict` / `RestrictAddressFamilies=`.
3. **A judgment record.** Every decided exception ("this cache write skips fsync
   because…") lives in `claims.cue` as data, so it is queryable and survives
   refactors instead of rotting in a doc.

## Install

```bash
go install github.com/larsartmann/io-surface/cmd/io-surface@latest
```

No `cue` binary is needed at runtime: the manifest is loaded in-process with
`cuelang.org/go`. `cue` is only for hand-editing CUE (the `cue fmt` habit).
The CLI itself is built with [cmdguard](https://github.com/LarsArtmann/cmdguard)
(typed flags, styled help and errors, hint-rendered failures); the AST walk
that powers the gate stays stdlib-only and hermetic.

## 60-second adoption

```bash
cd your-repo
io-surface init      # writes docs/architecture/io-surface/{schema,facts,claims,baseline}.cue
io-surface check     # green by construction: the snapshot IS your tree
io-surface render --out docs/architecture/io-surface.md
```

From then on, `check` fails only on I/O added **after** the snapshot. Declare it,
or grandfather it in `baseline.cue` with a note and a date.

`init` scans `cmd/` and `internal/` (override with `--dirs` upstream if your layout
differs). Test files are never manifest sites; `internal/testutil/` sites are marked
`testinfra`.

### No manifest? Lint anyway

The gate wants a manifest; the bugs don't. `lint` runs the same extraction plus
detector rules over any Go tree — no onboarding:

```bash
io-surface lint --root .
```

Rules are evidence-gated: they ship at `info` and never fail a build by default;
`--enforce` exits 1 on error-severity findings (severities are promoted only after
fleet false-positive measurement, so a new version can never go red by surprise).
Findings reuse the gate's rule-ID contract (`io-surface/err-dropped`, …), so
`--format sarif` and `--format json` work unchanged. Detected today: discarded error
returns (`x, _ := os.ReadFile(p)` flags; `_ =` deliberate ignores never do), with
more I/O bug and inefficiency rules landing across the v0.3 line.

### Suppressing a finding: `//io-surface:ignore`

When a finding is correct as written (a deliberate fire-and-forget cleanup, a
force-kill path), silence it at the site — one rule at a time, with a reason
that lives in the review diff:

```go
//io-surface:ignore err-dropped best-effort cache cleanup; nothing to do on failure
os.Remove(cachePath)
```

The directive governs the call on its own line or the line directly below it.
The rule name must be exact and the reason mandatory: a malformed directive
(unknown rule, missing reason) suppresses nothing and instead reports its own
`io-surface/ignore-invalid` finding, so a typo can never quietly eat a real
bug. File-level ignores do not exist by design — a suppression that blunt is a
baseline entry, not a comment.

### BuildFlow

The `provider/` package self-registers io-surface as a first-class BuildFlow
tool (go-finding toolsdk): `buildflow -s io-surface` runs the manifest gate
for adopted repos AND the radar sweep for every Go module — no manifest, no
adoption, still real findings at info severity. One recorded enforcement
policy governs both faces: a manifest's `enforcement: "audit"` demotes errors
to warnings, so a repo's own recorded posture decides whether BuildFlow fails.

## Commands

| Command                                                                                                            | What it does                                                                                                                             |
| ------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `io-surface init [--root DIR] [--force]`                                                                           | Extract the tree and write the CUE package (green by construction; `--force` refreshes `facts.cue` only)                                 |
| `io-surface check [--root] [--manifest] [--json] [--format text\|json\|sarif] [--advisory] [--strict] [--enforce]` | The bidirectional gate (exit 1 on error findings); `--format sarif` emits SARIF 2.1.0; `--enforce` overrides an audit enforcement policy |
| `io-surface render [--root] [--out FILE]`                                                                          | Render the inventory markdown from the manifest (claims-aware)                                                                           |
| `io-surface render --doc FILE [--check]`                                                                           | Rewrite the generated block inside FILE (prose preserved); `--check` exits 1 on drift                                                    |
| `io-surface extract [--root] [--include-tests]`                                                                    | Print the raw extraction as JSON (debugging, counts)                                                                                     |
| `io-surface lint [--root] [--format text\|json\|sarif] [--enforce] [--include-tests]`                              | Detector findings without a manifest (exit 0 unless `--enforce` and error-severity findings); any Go tree lints                          |
| `io-surface absorb --capslock REPORT [--check] [--module] [--engine-version]`                                      | Fold a pinned Capslock JSON report into `capabilities.cue` (that file only); `--check` reports advisory drift, never exits 1             |
| `io-surface baseline add <id> --note WHY [--date YYYY-MM-DD]`                                                      | Grandfather one undeclared site in `baseline.cue` (surgical write, comments preserved); refuses duplicates                               |
| `io-surface … --ignore 'zz_generated*,*_templ.go'`                                                                 | Skip generated files everywhere (init/check/extract/render/baseline): globs match repo-relative paths and file names                     |
| `io-surface version` / `doctor` / `describe`                                                                       | cmdguard-provided: version line, DI/config health, hidden machine-readable CLI contract (`describe --output json`)                       |

Exit codes: `0` clean, `1` gate findings, `2` usage or internal error.

With `--json`, every gate outcome — clean, findings, or a manifest that fails
validation (rule `io-surface/manifest-validation`) — is a machine-parseable
report on stdout: `{"stats": {...}, "findings": [...]}`. Usage/internal
errors (exit 2) are not gate results and print to stderr.

### Embedding the inventory in a hand-written doc

Wrap any markdown in the `io-surface` markers and the renderer owns only the
generated section — your prose is never touched:

```markdown
# Our I/O policy

Hand-written context, review notes, exceptions.

<!-- io-surface:begin -->
<!-- io-surface:end -->
```

```bash
io-surface render --doc docs/architecture/IO.md         # rewrite the block
io-surface render --doc docs/architecture/IO.md --check # exit 1 when stale
```

Put the `--check` form in CI and the published inventory can never silently
go stale; regenerate with the plain form after manifest edits.

### Reachability facts (optional, advisory): `absorb`

Sites answer **where** I/O sits; capabilities answer **what a package can
reach**. The second half comes from Google's
[Capslock](https://github.com/google/capslock) (SSA callgraph, transitive
through dependencies), run externally and pinned — io-surface never imports
it, so the gate stays hermetic:

```bash
go install github.com/google/capslock/cmd/capslock@v0.3.4 # pin deliberately; build with the repo's own Go toolchain
capslock -output=json -packages=./... > capslock.json
io-surface absorb --capslock capslock.json --engine-version v0.3.4
io-surface absorb --capslock capslock.json --check # advisory drift report, never exits 1
```

`absorb` writes `capabilities.cue` — that file ONLY (`claims.cue`,
`baseline.cue` and `facts.cue` are never machine-written). The mapped
vocabulary (`FILES`, `NETWORK`, `EXEC`) lands per package as `direct` (the
package's own code — high signal) and `deep` (including transitive dependency
paths — linkage, not behavior). The gate rule `io-surface/shadow-capability`
(info, never fails) names a package whose **direct** capability has no covered
family: hidden I/O becomes a named finding, and a package-keyed baseline entry
is the deliberate answer. This repository's CI runs the same steps as a
non-blocking job and uploads the raw report as an artifact.

## Findings

| Rule                | Severity  | Meaning                                                                                                             |
| ------------------- | --------- | ------------------------------------------------------------------------------------------------------------------- |
| `undeclared-site`   | **error** | A high-confidence I/O site is neither declared nor baselined                                                        |
| `stale-claim`       | warning   | Declared site's symbol is gone from its file — the claim now lies                                                   |
| `id-drift`          | warning   | Same symbol, different ID form (function became a method); update the ID, it is not a new site                      |
| `missing-location`  | warning   | Site is declared extractable but carries no location                                                                |
| `location-drift`    | info      | Line numbers moved (derived data — regenerate `facts.cue`)                                                          |
| `extraction-blind`  | info      | Symbol exists but no watched I/O call extracts from it — mark `extractable: false` or teach the extractor the seam  |
| `ambiguous-match`   | info      | Two symbols in one package share a base name; declare the exact ID form                                             |
| `weak-site`         | info      | Heuristic-tier finding (method-name token like `.Do`, `.Serve`); advisory only                                      |
| `shadow-capability` | info      | Extracted call family not covered by the package's declared capability set — declare it or drop the call (advisory) |
| `capability-drift`  | info      | Committed `capabilities.cue` disagrees with a fresh engine run; from `absorb --check`, never the hermetic gate      |

Only `error` fails the gate. That is deliberate: line numbers are output, not input,
and a heuristics tier that could block a merge would be switched off within a week.

### Detector rule cards (`lint`)

Every detector ships at `info` until fleet false-positive evidence promotes it
(`docs/planning/2026-11-07_promotion-policy.md`). One card per rule: what it
detects, the known FP classes (so you can judge your own hits), and an
example. A finding that is correct as written is silenced at the site with
`//io-surface:ignore <rule> <reason>` (see above) — file-level ignores do not
exist by design.

| Rule                | Detects                                                                | Known FP classes (accept or suppress)                                                                                                                                                                                                                                    | Positive → fixed                                                  |
| ------------------- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| `err-dropped`       | A watched call's error return discarded (`x, _ := f()`, bare `f()`)    | `io.ReadAll` with capped limit; `filepath.Glob` on literal patterns; `crypto/rand.Read` — all carved out                                                                                                                                                                 | `data, _ := os.ReadFile(p)` → `if data, err := …; err != nil {…}` |
| `closer-missing`    | A closable constructor's result never closed or handed off             | Ownership intentionally transferred (cache store, long-lived handle)                                                                                                                                                                                                     | `f, _ := os.Open(p)` → `defer f.Close()`                          |
| `closer-missing`    | `resp` from an http call with `Body` never closed                      | Body handed to another func (silence with an ignore + reason)                                                                                                                                                                                                            | `resp, _ := http.Get(u)` → `defer resp.Body.Close()`              |
| `ctx-dropped`       | Context-blind call variant while `context.Context` is in scope         | Force-kill/cleanup commands where ctx-abort is wrong (TASKKILL-style)                                                                                                                                                                                                    | `exec.Command("git", …)` → `exec.CommandContext(ctx, …)`          |
| `defer-in-loop`     | `defer` inside a loop body (resources pile up until function exit)     | Tiny bounded loops where the deferred call is cheap and bounded                                                                                                                                                                                                          | Move the work into a helper func and defer there                  |
| `client-no-timeout` | Default http client helpers or `http.Client{}` literal without Timeout | One-shot CLI where a hang is acceptable; internal-only trust domain                                                                                                                                                                                                      | `http.Client{Timeout: 10 * time.Second}`                          |
| `stat-then-open`    | Identical path expression stat'd then opened (TOCTOU + double syscall) | Cache-probe pattern (stat as existence check, open as lazy create)                                                                                                                                                                                                       | Drop the stat; open and handle the error                          |
| `io-in-loop`        | Watched I/O inside a loop body (N+1 shape)                             | Retry loops; genuinely per-item I/O; same-path repeats (refinement landing)                                                                                                                                                                                              | Hoist the call; batch reads; open once outside                    |
| `write-non-atomic`  | Direct final-path write without temp+rename (crash tears the file)     | Decided semantics (2026-11-07 fleet evidence): detection stays universal; a harmless write declares `durability: "ephemeral"` and the finding drops. Mechanical classes pending carve-out: `O_APPEND` journals, lockfile creates, writability probes, test-golden writes | Write temp + `os.Rename` (or adopt go-atomic-write)               |
| `tiny-buffer`       | A sub-page buffer feeding a read/write helper (syscall amplification)  | Decided semantics (2026-11-07): only the watched-consumer table (`io.ReadFull`, `io.CopyBuffer`, `bufio.{NewReader,Writer}Size`), in-loop only — custom wrappers stay unresolved by hermetic design. Fleet hits: 0; preventive rule (mutation-recall 100%)               | `make([]byte, 4096)` or `bufio.Scanner`                           |
| `ignore-invalid`    | A malformed `//io-surface:ignore` directive (unknown rule, no reason)  | None — this IS the guardrail; fix the directive                                                                                                                                                                                                                          | Correct the rule name or add the reason                           |

## The manifest

```text
docs/architecture/io-surface/
  schema.cue        the standard (families, disciplines, facts/claims split) — do not edit
  facts.cue         EXTRACTED: location, family, call tokens. Regenerate freely
  claims.cue        HAND-WRITTEN: resource, trigger, discipline tags, exceptions, rationale
  baseline.cue      grandfathered undeclared sites: {id, note, date}. Only shrinks
  capabilities.cue  ENGINE-DERIVED (optional): per-package reachability, written only by `absorb`
```

`facts.cue` records the repo's enforcement policy (written by `init`):
`enforcement: "gate"` — error findings fail the build — or `"audit"`, which
reports findings but exits 0 (the adoption posture). Because the policy lives
in the manifest, weakening the gate is a committed, reviewable diff, never a
silent CI edit; `check --enforce` overrides it for a single run.

Two contract rules the manifest standard enforces on itself:

- **Taxonomy freeze:** the families, disciplines, and capability names are
  closed enums. Adding one is a schema version bump (`io-surface/v1` →
  `v1.1`), never a casual edit — every consumer's manifest unifies against
  the standard, so silent vocabulary growth would lie to older tools.
- **ID migration:** a site ID embeds its package path, so moving a package
  retires the old IDs and creates new ones. That is expected drift, not
  data loss: regenerate facts, re-attach the claims prose to the new IDs,
  and let `check` confirm the pairing. The baseline only shrinks — never
  rename a baseline entry to "carry" it across a move; re-baseline
  deliberately if the new site still needs grandfathering.

Both files contribute to the same `sites` value (CUE unification), so a claims
entry only carries what a human knows:

```cue
sites: "internal/server.AllowlistManager.Save": claims: {
	resource:   "temp_allowlist_path JSON"
	trigger:    "every grant/revoke/expiry sweep/shutdown"
	discipline: ["atomic-write"]
}

sites: "internal/dns.timedCacheWrite": claims: {
	exception: "no-fsync"
	rationale: "the URL cache is re-fetched unconditionally after a crash, so a torn file is never parsed"
}
```

The ten families: `file-read`, `file-write`, `embed`, `database`, `listener`,
`outbound`, `stream-copy`, `entropy`, `ipc-subprocess`, `env`. A site has exactly
one (its dominant role); secondary roles go in `claims.rationale`.

### Severity modulation: `criticality` and `durability` (v1.2)

Two optional claims fields turn detector findings into weighted signals:

```cue
sites: "internal/queue.companion.reads": claims: {
	criticality: "critical"   // low | normal | critical — default normal
	durability:  "durable"    // ephemeral | recoverable | durable — default recoverable
}
```

When a manifest is present, `lint` escalates findings one severity step on
`critical` sites (the message says `[escalated: claims.criticality=critical]`);
no manifest means no modulation. Claims that cannot hold on their family are
rejected by the gate itself: rule `io-surface/claim-conflict` (warning) flags
`atomic-write` on a `file-read` site — a lying claim, not a judgment.
`criticality`/`durability` are schema v1.2; v1.1 manifests keep unifying
unchanged (the fields default to `normal`/`recoverable`).

### ID conventions

```text
pkg.Func            internal/dns.estimatedLines
pkg.Type.Method     internal/server.AllowlistManager.Save
pkg.Var             internal/server.openAPISpec        (go:embed)
pkg.Func#aspect     internal/config.Load#file          (one symbol, two families)
area.topic          logging.sink-topology              (no single symbol)
```

Matching is receiver-insensitive (`internal/server.Server.Save` and
`internal/server.Save` are the same site), but exact IDs are paired first, so two
different types with a same-named method are never silently merged.

## The extraction tier

`internal/extract` is a stdlib-only AST walker (`go/parser` + `ast.Inspect`,
no `go/packages`, no module downloads), so it runs in a Nix sandbox with
`GOPROXY=off`. It classifies:

- **resolved** — package-qualified calls through resolved imports (`os.*`, `io.*`,
  `net.*`, `tls.*`, `quic.*`, `http3.*`, `http.*`, `crypto/rand`, `database/sql`,
  `os/exec`, `syscall`, otel exporters, oidc, cqrs engines). These can fail the gate.
- **token** — method-name heuristics on unresolvable receivers
  (`ListenAndServe`, `DialContext`, `RoundTrip`, `ExecContext`, …). Advisory.
- **seams** — same-package wrappers (`atomicWriteFile`, `EnqueueOp`,
  `AppendBatch`, …) so the raw call is attributed to the wrapper, not to its callers.

Type facts (interface dispatch, cross-package method sets) are out of scope by
design: sites the AST cannot see are declared as `facts.extractable: false` instead
of being guessed at.

A discipline for keeping these docs honest: every status report claim names the
command that verified it (`check --root .` → `0 errors`, `go test -race ./...`,
`gh run view …`), so the next reader can re-run instead of trusting.

## What this is not

- **Not a benchmark suite.** The median/budget perf gate is a separate concern
  (different economics: load-sensitive numbers must never share an exit code with a
  deterministic gate). See the decision matrix in the originating dnsblockd
  brainstorm.
- **Not an auto-fixer.** Disciplines classify; they never rewrite your code.
- **Not multi-language yet.** The taxonomy and the manifest contract generalize; the
  extractor is Go-first.
- **Not a dashboard.** The manifest is data; render it however you like.

## Status

`v0.4.0` — the integration line: the BuildFlow provider runs BOTH faces
(`buildflow -s io-surface`): adopted repos get the manifest gate, every Go
module in a BuildFlow sweep gets the advisory radar at info severity — the
182-repo fleet scan became continuous evidence for severity promotion. CI
now emits the tool's own lint SARIF as an artifact (artifact-only, per the
recorded upload policy).

`v0.3.0` — the detector line: `io-surface lint` finds dropped errors, leaked
closers, context-blind calls, timeout-less clients, loop-carried I/O, and
double syscalls in ANY Go tree — no manifest needed. Fleet-proven on 160 repos
(one command: `scripts/fleetscan.sh`); rules ship at `info` with evidence-gated
severity promotion. Everything from `v0.2.0` stands: extractor, bidirectional
gate, scaffolder, renderer, claims-aware docs embedding, capabilities via
`absorb` (advisory).

Next, in order: dnsblockd v1→v1.2 judgment migration (owner call), the next
pilot cohort (go-taskqueue, 37 findings), severity promotion once fleet
evidence meets the FP≈0 bar.

Shipped in the 0.2 line: `--json` holds for every gate outcome (validation
failures included), `init --force` can no longer clobber `claims.cue`/
`baseline.cue`, and the renderer embeds into hand-written docs via BEGIN/END
markers with a `--check` drift gate — this repository's own inventory doc is
gated by it in CI.

### Adoption safety: the refresh ritual

`init --force` regenerates ONLY `facts.cue` — `claims.cue`, `baseline.cue` and
`schema.cue` are never overwritten once they exist (a planted-file test pins
this). The safe loop after any I/O-adding change:

```bash
go run ./cmd/io-surface init --root . --force   # facts only
git diff docs/architecture/io-surface/facts.cue # review the mechanical delta
go run ./cmd/io-surface check --root .          # gate must stay green
```

If a refresh surprises you, `git restore docs/architecture/io-surface/facts.cue`
puts it back — the mechanical file has no judgment to lose. NEVER hand-edit
`facts.cue` locations; never expect tooling to rewrite `claims.cue`/
`baseline.cue` (the one sanctioned machine write is `baseline add`, which
appends a note+date entry surgically).

## Development

```bash
go build ./...
go test ./...
go vet ./...
io-surface check --root .   # the tool checks itself
```

## License

MIT
