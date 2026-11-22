# erraudit policy

Decided 2026-11-22 clearing the first `erraudit lint ./... --max` sweep
(16 findings; verified: `erraudit lint ./... --max --disable generic_return`
exits 0).

erraudit is the repo's error-handling auditor. The canonical strict audit is:

```bash
erraudit lint ./... --max --disable generic_return
```

## generic_return: accepted as a recorded posture

`--max` enables `--enforce-generic-return`: every function returning the bare
`error` interface whose body creates an error is flagged, and `--no-suppress`
bypasses every `//nolint` comment. Returning `error` is idiomatic Go, and the
exit-code contract depends on it (`cmd/io-surface/main.go` `exitCode`
classifies via `errors.Is`), so the check is disabled as the standing
decision, not per run. New occurrences of the actual problem class still
belong in code review; concrete pointer error types (`*MyError`) are a
typed-nil landmine and stay banned.

## Coded errors: stdlib sentinels, not go-error-family

erraudit's fix template suggests `errorfamily.NewRejection`/`WrapRejection`.
Deliberately NOT adopted: go-error-family is an indirect dependency and
promoting it would break the four-direct-dependencies architecture
(AGENTS.md). The same contract, stdlib-first (template:
`internal/manifest/baseline_write.go`):

- sentinels live in per-package `errors.go` (`ErrBaseline`,
  `ErrNotInitialized`, `ErrCapabilityType`, `ErrDocBlock`, ...),
- failure paths wrap: `fmt.Errorf("%w: specifics", ErrXxx)`,
- callers (and tests) classify with `errors.Is`.

2026-11-22 sweep outcome: the five anonymous-failure functions gained
sentinels (`errNoSites`, `errUsage`, `ErrCapabilityType`, `ErrDocBlock`);
`capability.Families` was deleted as production-dead instead of coded; the
remaining generic_return advisories are the accepted class (sentinel or `%w`
already present, per erraudit's own verdict message).
