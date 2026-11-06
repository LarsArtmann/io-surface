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

## Commands

| Command                                                                                                | What it does                                                                                             |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| `io-surface init [--root DIR] [--force]`                                                               | Extract the tree and write the CUE package (green by construction; `--force` refreshes `facts.cue` only) |
| `io-surface check [--root] [--manifest] [--json] [--format text\|json\|sarif] [--advisory] [--strict]` | The bidirectional gate (exit 1 on error findings); `--format sarif` emits SARIF 2.1.0                    |
| `io-surface render [--root] [--out FILE]`                                                              | Render the inventory markdown from the manifest (claims-aware)                                           |
| `io-surface render --doc FILE [--check]`                                                               | Rewrite the generated block inside FILE (prose preserved); `--check` exits 1 on drift                    |
| `io-surface extract [--root] [--include-tests]`                                                        | Print the raw extraction as JSON (debugging, counts)                                                     |

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

## Findings

| Rule               | Severity  | Meaning                                                                                                            |
| ------------------ | --------- | ------------------------------------------------------------------------------------------------------------------ |
| `undeclared-site`  | **error** | A high-confidence I/O site is neither declared nor baselined                                                       |
| `stale-claim`      | warning   | Declared site's symbol is gone from its file — the claim now lies                                                  |
| `id-drift`         | warning   | Same symbol, different ID form (function became a method); update the ID, it is not a new site                     |
| `missing-location` | warning   | Site is declared extractable but carries no location                                                               |
| `location-drift`   | info      | Line numbers moved (derived data — regenerate `facts.cue`)                                                         |
| `extraction-blind` | info      | Symbol exists but no watched I/O call extracts from it — mark `extractable: false` or teach the extractor the seam |
| `ambiguous-match`  | info      | Two symbols in one package share a base name; declare the exact ID form                                            |
| `weak-site`        | info      | Heuristic-tier finding (method-name token like `.Do`, `.Serve`); advisory only                                     |

Only `error` fails the gate. That is deliberate: line numbers are output, not input,
and a heuristics tier that could block a merge would be switched off within a week.

## The manifest

```text
docs/architecture/io-surface/
  schema.cue     the standard (families, disciplines, facts/claims split) — do not edit
  facts.cue      EXTRACTED: location, family, call tokens. Regenerate freely
  claims.cue     HAND-WRITTEN: resource, trigger, discipline tags, exceptions, rationale
  baseline.cue   grandfathered undeclared sites: {id, note, date}. Only shrinks
```

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

`v0.1.0` — extractor, bidirectional gate, scaffolder and renderer, with the CUE
contract. Proven against two real repos: a hand-ported 131-site manifest (dnsblockd)
and a from-scratch 126-site pilot (InboxClean), plus self-hosting.

Next, in order: wire it into the pilot repos as a Nix/BuildFlow check, then
SARIF emission and the systemd hardening generator (the highest-value consumer:
`ReadWritePaths` derived from the `file-write` family).

Shipped since the tag: `--json` holds for every gate outcome (validation
failures included), `init --force` can no longer clobber `claims.cue`/
`baseline.cue`, and the renderer embeds into hand-written docs via BEGIN/END
markers with a `--check` drift gate — this repository's own inventory doc is
gated by it in CI.

## Development

```bash
go build ./...
go test ./...
go vet ./...
io-surface check --root .   # the tool checks itself
```

## License

MIT
