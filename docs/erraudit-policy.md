# erraudit policy

Decided 2026-11-22 clearing the first `erraudit lint ./... --max` sweep
(16 findings); go-error-family adopted the same day by owner directive
("go-error-family is a code lib I use and ALL LarsArtmann/* SDKs are totally
to be used if useful"). Every claim below names its verifying command; last
full battery 2026-11-22 15:3x CET.

The canonical strict audit is:

```bash
erraudit lint ./... --max --disable generic_return
```

Exits 0 (verified). `erraudit lint ./... --max --enforce-coded-errors
--disable generic_return` also exits 0: every go-error-family constructor
carries a non-empty code, so the What/Why/Fix template bar is met.

## generic_return: accepted as a recorded posture

`--max` enables `--enforce-generic-return`: every function returning the bare
`error` interface whose body creates an error is flagged, and `--no-suppress`
bypasses every `//nolint` comment. Returning `error` is idiomatic Go, and the
exit-code contract depends on it (`cmd/io-surface/main.go` `exitCode`
classifies via `errors.Is`), so the check is disabled as the standing
decision, not per run. Concrete pointer error types (`*MyError`) are a
typed-nil landmine and stay banned. 11 functions remain flagged at the
accepted class (sentinel or `%w` already present, per erraudit's own
verdict); functions using go-error-family constructors drop out of plain
`--max` entirely — coded constructors are the pattern the linter wants.

## Coded errors: go-error-family IS the standard

The house library (`github.com/larsartmann/go-error-family`, direct dep since
2026-11-22) owns error classification: failure surfaces carry a behavioral
Family + a machine-readable code in fleet style (`gate.findings`, `cli.usage`,
`init.no-sites`, `capability.type`, `render.doc-block`) and render as
`[family:code] message`. Sentinels stay in per-package `errors.go` — the home
convention is unchanged, the mechanism changed. Sites construct rejections
with the shared code constant, so `errors.Is` matches by code+family and
tests pin the machine contract via `errorfamily.Code`.

Deliberately NOT adopted from the library (for now, re-decide when CLI UX
work warrants it):

- Family-default exit codes (BSD sysexits): this CLI's 0/1/2 contract
  (0 clean / 1 gate findings / 2 usage-internal) stays local in `exitCode`
  (main.go) — one fact, one home. Gate findings have no errorfamily family.
- What/Why/Fix templates (`handle.go` layer): codes come first; the template
  layer is a follow-up, not a blocker.

## Migration backlog (measured)

`erraudit lint ./... --max --enforce-go-error-family` flags 11 functions
(2026-11-22, erraudit ee8a415) — the remaining stdlib-constructor error
surfaces (manifest, scaffold, extract, check, lint, emit, and the
accepted-class cmd handlers). They are the queue for the full coded-errors
end state; see the 2026-11-22 status report §f for the routing.
