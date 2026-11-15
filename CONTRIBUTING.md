# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

One command runs the whole quality bar (what CI's `test` job enforces):

```bash
scripts/self-gate.sh
```

The individual steps, if you want them one at a time:

```bash
go build ./...                          # build
go vet ./...                            # vet
go test -race ./...                     # tests
gofmt -l .                              # must print nothing
go run ./cmd/io-surface check --root .  # the tool gates its own repo
scripts/check-docs-pointers.sh          # living-docs references resolve
```

Quality gates beyond the CI job (formatter parity, go-fix tooling) run via
BuildFlow: `buildflow -s io-surface`. No `golangci-lint` and no
`.golangci.yml` exist here by design — the errcheck waiver and its
rationale live in `docs/errcheck-policy.md`.

Note: this repo is gated by its own tool — if your change adds an I/O call
under `cmd/` or `internal/`, the self-check above will fail until the site is
declared in the manifest (`docs/architecture/io-surface/`). Run
`go run ./cmd/io-surface init --root . --force` to regenerate `facts.cue`;
your `claims.cue`/`baseline.cue` are never touched by a refresh.

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
