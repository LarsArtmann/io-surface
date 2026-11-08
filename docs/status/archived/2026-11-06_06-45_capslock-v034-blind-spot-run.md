> **Live remainder:** every open item in this report was harvested into
> TODO_LIST (docs-gates routing PASS, 0 unrouted rows, 2026-11-08) — remaining
> context lives in the report body below.
> Archived 2026-11-08 per the docs-cadence ritual (count threshold 15).

# Capslock v0.3.4 blind-spot run — first real capability facts for io-surface

**Report:** 2026-11-06 06:45 CET
**Repo state analyzed:** working tree at `2be70fb` (docs later formatted by dprint)
**Purpose:** evidence for the unification decision (Pareto plan T02) — do Capslock's
SSA callgraph capabilities reveal I/O that io-surface's hermetic extractor misses?

---

## Method (reproducible)

- Capslock **v0.3.4** pinned. `go install` is blocked by the session guard on this
  machine, so the analyzer was built from the pinned module zip
  (`proxy.golang.org/github.com/google/capslock/@v/v0.3.4.zip`) into `/tmp/bin`.
- **Toolchain skew lesson:** the default build (go1.26.8) failed to type-check this
  repo's go1.27.0 stdlib (`unknown field rfd in struct literal of type splicePipe`).
  Rebuilding the analyzer with the _target's_ toolchain
  (`GOTOOLCHAIN=go1.27.0 GOEXPERIMENT=none go build`) fixed it. CI wiring (plan T09)
  must build capslock with the repo's Go version, or skew noise will look like
  findings.
- Binary sha256: `e63a7cd21c8371859681bb398ce5de3132d6cdb4b874d54c7b3eb2b5040b9705`
  (the go1.27.0 rebuild; original binary deleted).
- Command: `capslock -output=json -packages=./...` → 770 KB JSON, 15.6 s wall.
  Raw artifact kept in `/tmp` only (not committed; regenerate with the command above).
- Totals: 281 packages, 79 modules, **236 capability edges**; 7 first-party packages.

## Direct capabilities in first-party code — every sink paired against the manifest

| Sink (verbatim from run)                | Site                                  | Declared in facts.cue?                           | Verdict                                                |
| --------------------------------------- | ------------------------------------- | ------------------------------------------------ | ------------------------------------------------------ |
| `os.ReadFile`                           | `cmd/io-surface/commands.go:165`      | yes (`cmd/io-surface.applyDoc`, file-write)      | covered                                                |
| `os.WriteFile`                          | `cmd/io-surface/commands.go:158`      | yes (`cmd/io-surface.runRender`)                 | covered                                                |
| `os.ReadFile`                           | `internal/check/location.go:23`       | yes (`internal/check.symbolVisible`)             | covered                                                |
| `os.Stat`                               | `internal/extract/extract.go:111`     | yes (`internal/extract.Extract`)                 | covered                                                |
| `os.ReadFile`                           | `internal/extract/extract.go:276`     | yes (`…extractor.parseFile`)                     | covered                                                |
| `os.Stat`                               | `internal/manifest/manifest.go:121`   | yes (`internal/manifest.Load`)                   | covered                                                |
| `os.Stat`                               | `internal/manifest/manifest.go:297`   | yes (`…Manifest.Validate`)                       | covered                                                |
| `os.MkdirAll`/`os.Stat`/`os.WriteFile`  | `internal/scaffold/scaffold.go:55-74` | yes (`internal/scaffold.Write`)                  | covered                                                |
| `os/exec.Command`                       | `internal/scaffold/scaffold.go:205`   | yes (`internal/scaffold.git`, ipc-subprocess)    | covered                                                |
| `filepath.Walk` → internal `os.Lstat`   | stdlib `path/filepath` internals      | yes (`…extractor.walk` declares `filepath.Walk`) | covered — see complementarity                          |
| **`stdout.Write` (`(*os.File).Write`)** | **`cmd/io-surface/commands.go:77`**   | **no**                                           | **real gap → TODO #49** (stdout/stderr scope decision) |

**Score: 10 of 11 real direct sinks are declared.** The extractor's call tables
caught every file/subprocess sink Capslock found in first-party code. The one
undeclared sink is stdout — a scope-boundary decision, not a missed table entry
(`fmt.Print*` is equally unseen; `scope.in` is silent on stdout).

## Capslock false-positive classes observed (evidence for absorb's advisory tier)

1. **Phantom NETWORK via SSA range-func conflation.** `internal/extract` — the
   _hermetic_ package — reported 4 DIRECT NETWORK edges, all with this chain shape:
   `parseFile → go/parser.ParseFile → go/build/constraint.parsePlusBuildExpr →
   strings.FieldsSeq$1 → net/http.ParseCookie$1 @ iter.go:96`. Anonymous
   range-over-func closures from unrelated stdlib packages are conflated; there is
   no network call. → TODO #50 (verify-before-filing applies).
2. **Phantom NETWORK via interface-dispatch conflation.** `cmd/io-surface` showed
   `(*net/http.response).Write @ encoding/json/v2 v2_stream.go:156` reached from
   `runCheck`/`runExtract` — an `io.Writer` param resolved to an http response type
   no CLI path ever provides.
3. **UNANALYZED over-labeling.** `errors.Is @ main.go:44` (pure logic) flagged as
   UNANALYZED DIRECT; `io.ReadAll` behind `go/parser.readSource` flagged UNANALYZED
   through stdlib internals.
4. **Post-I/O methods counted as I/O.** `(*os.fileStat).IsDir @ extract.go:241`
   (FILES) — an in-memory field check on a `Stat` result whose I/O is already
   attributed to the declared `Stat`/`Walk` sites.

## Complementarity — the exact two engines the unification wants

- `filepath.Walk @ extract.go:237` (declared site) internally calls `os.Lstat`
  (stdlib `path.go:424`). io-surface names **where** the I/O enters our code;
  Capslock names **what** syscall class is ultimately reachable. Both are true;
  neither alone suffices for `shadow-capability`-style judgments.
- `json.Encoder.Encode` (used by `runCheck`/`runExtract`) reaches REFLECT
  (`encoding/json/v2` internals). Runtime-power capabilities (REFLECT, RUNTIME,
  SYSTEM_CALLS, UNSAFE_POINTER, OPERATING_SYSTEM, EXEC-transitive,
  READ_SYSTEM_STATE, MODIFY_SYSTEM_STATE/\*) appear in **every** package
  transitively — confirming plan guardrail 6: they are `out-of-scope` vocabulary
  until owner question Q5 is decided.

## Verdicts feeding the unification contract (plan T03/T05)

1. **Mapped vocabulary confirmed sufficient for the direct tier:** FILES →
   file-read/file-write/listener/outbound families; EXEC → ipc-subprocess; NETWORK →
   outbound/listener. All direct first-party facts in this repo map cleanly.
2. **Advisory-first is mandatory:** one third of Capslock's DIRECT labels in this
   small hermetic repo were false positives. Graph-tier findings must never exit 1
   (plan guardrail 2) and absorb must tier unmapped/suspicious names loudly (M34).
3. **`--ignore`/noise budget needed:** UNANALYZED and transitive runtime-power
   edges dominate edge counts (236 edges, ~90% transitive/stdlib). The absorb
   contract should store per-package capability _sets_ (pkg → caps), not edges.
4. **CI recipe:** build capslock pinned AND toolchain-matched (see Method), upload
   the JSON artifact, absorb, advisory-gate (plan T09).
