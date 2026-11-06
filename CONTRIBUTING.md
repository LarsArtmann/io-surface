# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

The whole quality bar (what CI enforces):

```bash
go build ./...                          # build
go vet ./...                            # vet
go test ./...                           # tests
gofmt -l .                              # must print nothing
go run ./cmd/io-surface check --root .  # the tool gates its own repo
```

Note: this repo is gated by its own tool — if your change adds an I/O call
under `cmd/` or `internal/`, the self-check above will fail until the site is
declared in the manifest (`docs/architecture/io-surface/`). Run
`go run ./cmd/io-surface init --root . --force` to regenerate `facts.cue`;
your `claims.cue`/`baseline.cue` are never touched by a refresh.

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
