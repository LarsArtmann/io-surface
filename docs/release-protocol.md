# Release protocol (io-surface)

Relocated from AGENTS.md 2026-11-11 (agents-md-size gate: 220-line cap).
The version const lives in `cmd/io-surface/main.go`; the CHANGELOG follows
Keep a Changelog. Releases are 0.x SemVer (features = MINOR): v0.2.0
2026-11-06, v0.4.0 (provider two-face) 2026-11-07, v0.5.1 (history purge +
retraction of v0.2.0–v0.5.0 — proxy zips are immutable, so blob-carrying
versions stay cached; `@latest` moved to the clean tag) 2026-11-08.

## The protocol that worked

1. Version const + CHANGELOG cut + README status line.
2. Full gates (`go build/vet/test`, `gofmt -l .`, self-check, self-render).
3. Push BOTH remotes (`origin` Forgejo, `github` public mirror).
4. CI green on THAT commit.
5. Annotated tag to BOTH remotes.
6. Proxy check: `go list -m github.com/larsartmann/io-surface@vX`.
7. GitHub Release.

pkg.go.dev indexing lags (404s are crawler lag, not breakage — the proxy is
the hard guarantee).

## Provider release coupling

`provider/` (repo root, NOT `internal/` — BuildFlow must import it)
self-registers as a BuildFlow DAG tool via `go-finding/toolsdk`. Since
v0.4.0 Detect runs BOTH faces: the manifest gate (`check.Run`) for adopted
modules AND the io-bug-radar sweep (`lint.Run`) for every Go module —
manifestless modules get radar-only at info severity, never a step error,
and audit demotion applies to both faces (one recorded posture governs).

Both halves of that loop ship together: the package here AND the blank
import in BuildFlow's `tools/providers/sdk_imports.go` (its go.mod requires
the published tag, so a provider change means a new release here before
BuildFlow can pick it up).

## Upstream issue protocol

Upstream Capslock phantom-edge issue (#50) is BLOCKED, not fileable: the
verify-before-filing gate failed — a minimal zero-network FieldsSeq repro
yields zero NETWORK edges under v0.3.4, so the conflation needs the fuller
io-surface call graph. Don't re-attempt without a faithful repro.

## BuildFlow pre-commit hook

The hook lives at `.githooks/pre-commit` (`core.hooksPath=.githooks`),
fleet-template generated. The 2026-11-06 template had a bash syntax bug
(missing outer `done` + `$( )` close in the staged-module loop) — fixed
here at birth; the upstream template
(`BuildFlow internal/cli/precommit_types.go`) has since been fixed too
(verified structurally 2026-11-07), so the local hook and the template are
functionally identical (comment wording + `| sort -u` placement differ —
not worth regenerating). The hook's file-size gate reads `max_file_size`
from `.buildflow.yml` — this repo pins 650 with rationale (split tracked
as TODO #64).
