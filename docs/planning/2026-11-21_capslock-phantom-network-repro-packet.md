# Capslock phantom-NETWORK repro packet (TODO #50) — VERIFIED 2026-11-21

Status: the verify-before-filing gate that FAILED on 2026-11-06 now PASSES.
A faithful minimal repro exists (23 lines, deterministic, discriminated on 4
variants). The draft below is owner-gated: DO NOT run `gh issue create`
without an explicit publish order.

## What changed since the 2026-11-06 failure

That attempt reproduced with `strings.FieldsSeq` alone and got zero edges —
the trigger was misidentified. Today's run on the full io-surface tree
(capslock v0.3.4, go1.27.3) still shows **63 NETWORK entries** on a CLI whose
own code never imports `net/http`, and minimizing FROM the real graph found
the actual trigger pair:

1. **a range-over-func over a stdlib Seq** (`strings.SplitSeq` — closure
   `strings.splitSeq$1`), and
2. **`net/http` anywhere in the analyzed package universe** (even a dead
   `_ = http.ParseCookie` reference) — this pulls in `ParseCookie$1`,
   net/http's own range-over-func closure (`net/http/cookie.go:99`: `for s := range strings.SplitSeq(line, ";")`).

A generic instantiation is NOT required (plain functions reproduce).

## Minimal reproducer (verified)

`go.mod`: `module phantomB-nogen` + `go 1.27`; no dependencies.

```go
package main

import (
	"fmt"
	"net/http"
	"strings"
)

type File struct{ Fields []string }

func parseConfig(cfg File) {
	_ = cfg
	for f := range strings.SplitSeq("a,b", ",") {
		fmt.Println(f)
	}
}

func main() {
	_ = http.ParseCookie
	parseConfig(File{})
}
```

Command: `capslock -output=json -packages=./...`
Result: 2 NETWORK entries, chains
`main -> parseConfig -> strings.splitSeq$1 -> net/http.ParseCookie$1`,
site `iter.go:41:14`.

## Discriminator matrix (all runs capslock v0.3.4, go1.27.3 linux/amd64)

| variant                                                                     | NETWORK entries |
| --------------------------------------------------------------------------- | --------------- |
| plain func + `strings.SplitSeq` range + `_ = http.ParseCookie`              | **2**           |
| same with generic instantiation `parseConfig[T any]`                        | **2**           |
| same shape but a custom local `iter.Seq` instead of `strings.SplitSeq`      | 0               |
| plain func + `strings.SplitSeq`, no `net/http` anywhere (the 11-06 attempt) | 0               |
| `json.NewEncoder(os.Stdout).Encode(...)` standalone (class A minimal)       | 0               |

`GOEXPERIMENT=jsonv2` is set in this environment; clearing it does NOT
change the result (2 entries either way).

## The misattribution proof (strongest single fact for upstream)

The reported closure is named `net/http.ParseCookie$1` at
`filename: iter.go, line: 41` — but:

- `$(go env GOROOT)/src/net/http/iter.go` does not exist;
- `strings/iter.go:41` is `_, size := utf8.DecodeRuneInString(s)` — the body
  of `strings.splitSeq`'s returned closure.

So closure **identity** (a NETWORK-relevant net/http symbol) and closure
**position** (strings' own seq machinery) disagree, and the NETWORK
capability rides on the identity.

## Full-tree observation (context, not the repro)

On io-surface @ master (17 sites, 12 packages): 63 NETWORK entries rooted in
its own packages. Two edge families:

- class B (this repro's shape): `cmd/io-surface.init → internal/manifest.init
  → cuelang.org/go/cue.ParsePath → … → strings.splitSeq$1 →
  net/http.ParseCookie$1 @ iter.go:41` (11+ entries, one per manifest-touching
  entry point);
- class A (same family, different pair):
  `(*encoding/json.Encoder).Encode → (*net/http.response).Write @
  encoding/json/v2 v2_stream.go:156` (observed only on the full tree; the
  standalone json-encoder repro yields 0 — minimization not yet achieved).

## Upstream duplicate check (Gate 5, 2026-11-21)

No open/closed capslock issue mentions ParseCookie/SplitSeq/phantom-NETWORK.
Same-family precedent: google/capslock#318 (Go 1.27 sync.Pool →
ARBITRARY_EXECUTION misclassification) was accepted by collaborator jcd2 and
fixed in v0.3.4 — the report style below follows that precedent
(source-level analysis + version deltas).

## FILING DRAFT (owner gate: do not post without explicit order)

Proposed title:
`NETWORK capability misattributed across stdlib range-over-func closures (strings.splitSeq$1 -> net/http.ParseCookie$1)`

````markdown
> [!NOTE]
> This filing was drafted by GLM-5.3-flash (via Crush) from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

TL;DR: ranging over a stdlib Seq iterator (`strings.SplitSeq`) in one package while any analyzed package references `net/http` reports a phantom NETWORK path `strings.splitSeq$1 -> net/http.ParseCookie$1`, and the closure's name and its position disagree. Ask: scope seq-iterator closure bodies per package so the capability reflects real dials.

## Problem

A zero-network program reports NETWORK. Reproducer (module needs only `go 1.27`, no dependencies):

```go
package main

import (
	"fmt"
	"net/http"
	"strings"
)

type File struct{ Fields []string }

func parseConfig(cfg File) {
	_ = cfg
	for f := range strings.SplitSeq("a,b", ",") {
		fmt.Println(f)
	}
}

func main() {
	_ = http.ParseCookie
	parseConfig(File{})
}
```
````

`capslock -output=json -packages=./...` yields 2 NETWORK entries:

```json
{
  "packageName": "main",
  "capabilityName": "NETWORK",
  "path": [
    { "name": "main" },
    { "name": "parseConfig" },
    { "name": "strings.splitSeq$1", "package": "strings" },
    {
      "name": "net/http.ParseCookie$1",
      "package": "net/http",
      "site": { "filename": "iter.go", "line": "41", "column": "14" }
    }
  ]
}
```

Two things are wrong in that last hop:

1. The position misattributes: `iter.go:41` is `$(go env GOROOT)/src/strings/iter.go:41`, the body of `strings.splitSeq`'s closure (`_, size := utf8.DecodeRuneInString(s)`). There is no `net/http/iter.go`; `net/http.ParseCookie$1` is the range-over-func closure in `net/http/cookie.go:99` (`for s := range strings.SplitSeq(line, ";")`).
2. The edge is phantom: nothing in the program dials a network; `ParseCookie` only parses a string.

## Discriminators

| variant                                                            | NETWORK entries |
| ------------------------------------------------------------------ | --------------- |
| plain func + `strings.SplitSeq` range + `_ = http.ParseCookie`     | 2               |
| same with generic instantiation `parseConfig[T any]`               | 2               |
| plain func + custom local `iter.Seq` instead of `strings.SplitSeq` | 0               |
| plain func + `strings.SplitSeq`, no `net/http` anywhere            | 0               |

The trigger needs both a range over a stdlib Seq (closure `strings.splitSeq$1`) and `net/http` inside the analyzed universe (closure `ParseCookie$1`); the two then appear conflated into one chain.

## Observed at fleet scale

The same edges show up on a real 12-package CLI (io-surface) whose own code never imports `net/http`: 63 NETWORK entries rooted in its packages, e.g. `cmd/io-surface.init -> internal/manifest.init -> cuelang.org/go/cue.ParsePath -> ... -> strings.splitSeq$1 -> net/http.ParseCookie$1`, plus a second family `(*encoding/json.Encoder).Encode -> (*net/http.response).Write @ encoding/json/v2 v2_stream.go:156`.

## Versions

- capslock v0.3.4 (compiled with Go go1.27.3, includes Go tools v0.51.0)
- Go 1.27.3 linux/amd64; `GOEXPERIMENT=jsonv2` set in the environment, clearing it does not change the result
- stdlib per Go 1.27.3: `net/http/cookie.go:99`, `strings/iter.go:37`

Related: #318 was the same disease family (stdlib internals surfacing as a wrong capability) — this one survived that fix.

💘 Generated with Crush

```
## Line-number freshness (per verify-before-filing §5)

Re-verify `strings/iter.go:41`, `net/http/cookie.go:99`, and the capslock
version output at the EXACT Go toolchain/capslock build used when posting,
as the last step before `gh issue create`.
```
