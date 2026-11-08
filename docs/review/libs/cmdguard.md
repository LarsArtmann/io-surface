# cmdguard — adoption & utilization review

> **Status: USED — direct dependency** (`github.com/larsartmann/cmdguard/v4 v4.2.0`)
> **Verdict: fully leveraged for what this CLI is; one free patch bump pending.**
> **Adoption score: 93/100 (very high)** — 6.5 of 7 applicable capabilities fully leveraged
> Latest version: **v4.2.1** (2026-11-04, module proxy confirms) · io-surface pins **v4.2.0**
> Reviewed: 2026-11-07 · io-surface `docs/review/libs/` series

## What the library offers

cmdguard is the LarsArtmann Cobra-replacement CLI framework: commands and
flags declared as typed structs, cmdguard owns help/error rendering and the
exit path, the consumer owns the exit-code contract. Core v4 surface used
here: `NewApp`, `NewCommand`, `NewParentCommand`, `AddCommands`,
`WithShort/WithLong/WithExample`, `WithCLIVersion`, `WithBatteries`,
`ArgsFromContext`. Beyond what io-surface touches, the family offers:
config-file loading with env precedence and (unreleased) strict keys, `config
explain` provenance, a machine-readable error envelope
(`--output json|yaml|toml` with stable `error_code` fields, unreleased), dynamic
shell completion (`WithCompletion`), `describe` (versioned JSON contract of
commands/flags), `doctor`, and seven satellite modules (`prompts`, `spinner`,
`telemetry`, `lint`, `glamour`, `flightrecorder`, `schema`).

## io-surface today (evidence)

- `cmd/io-surface/main.go:62-161` — `v4.NewApp` + `WithCLIVersion` + `WithBatteries`, 8 commands + 1 parent wired via `AddCommands`; every command carries `WithShort/WithLong/WithExample`.
- Typed flag structs with `flag:`/`help:`/`default:`/`required:` tags for every command (`cmd/io-surface/commands.go:28-100, 496-502`), including a `required:"true"` flag (`absorbFlags.Capslock`, commands.go:96).
- `v4.NewParentCommand` + `WithSubcommands` for `baseline add` (main.go:149-155); positional args via `v4.ArgsFromContext` (commands.go:458).
- Exit-code contract deliberately owned by io-surface: `errFindings` sentinel + `exitCode()` mapping 0/1/2 (main.go:33-51) — cmdguard prints the error once, the returned error is for exit mapping only.
- Custom `version` command overrides cmdguard's auto-registered one and keeps the byte-pinned one-line output plus `--json` provenance (commands.go:48-75).

## Capability inventory

| #  | Capability                                                                                           | Status             | Evidence / reason                                                                                                                                                                                         |
| -- | ---------------------------------------------------------------------------------------------------- | ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | App wiring + typed struct-tag flags                                                                  | 🟢 Fully leveraged | 8 commands, every flag typed and documented                                                                                                                                                               |
| 2  | Per-command Short/Long/Example                                                                       | 🟢 Fully leveraged | all 8 commands, main.go:68-147                                                                                                                                                                            |
| 3  | `WithBatteries` (strict validation, signals, version/doctor, hints)                                  | 🟢 Fully leveraged | main.go:160                                                                                                                                                                                               |
| 4  | Parent commands + positional args                                                                    | 🟢 Fully leveraged | `baseline add`, main.go:149, commands.go:458                                                                                                                                                              |
| 5  | Consumer-owned exit-code contract                                                                    | 🟢 Fully leveraged | 0/1/2 mapping, main.go:39-51, pinned by `main_test.go`                                                                                                                                                    |
| 6  | Custom command overriding auto-registered ones                                                       | 🟢 Fully leveraged | `version --json`, commands.go:48-75                                                                                                                                                                       |
| 7  | Version currency                                                                                     | 🟡 Partially used  | v4.2.0 vs v4.2.1; v4.2.1 ships `887dcae` "Fix ExitCode(0) on errorfamily failures and flag leaks on reused CLIs" — low impact here (single Execute per process) but the bump is free                      |
| 8  | Config files / env precedence / `config explain`                                                     | ⚪ N/A             | io-surface is repo-driven: no config file concept; flags+defaults are the whole surface (deliberate)                                                                                                      |
| 9  | Machine error envelope (`--output json…` + `error_code`)                                             | ⚪ N/A             | io-surface's machine contract is its own `--format json/sarif` stdout, byte-pinned by goldens; usage errors are cmdguard's domain already                                                                 |
| 10 | Dynamic completion (`WithCompletion`)                                                                | ⚪ N/A             | only candidate would be `--format` value completion; no dynamic data exists today                                                                                                                         |
| 11 | Satellite modules (`prompts`, `spinner`, `telemetry`, `lint`, `glamour`, `flightrecorder`, `schema`) | ⚪ N/A             | no interactivity by design, sub-second runs (no spinner), telemetry deliberately absent, `lint` targets cmdguard-API linters, `schema` emits JSON Schema for config-file consumers (no config files here) |

Score derivation (skill rubric): N=11 rows, A=7 applicable, fully=6, partial=1 → round(100 × 6.5/7) = **93**.

## Recommendation

**Keep, bump to v4.2.1.** io-surface uses exactly the core surface the
framework recommends for gate CLIs and nothing it shouldn't. The satellite
modules stay out for honest reasons, not neglect. Action item (hygiene, not
urgency): `go get github.com/larsartmann/cmdguard/v4@v4.2.1 && go mod tidy`,
then run the full gate. Master is 8 commits ahead of v4.2.1 with unreleased
work (strict config keys, `config explain`, stable error codes) — none of it
changes the core API io-surface consumes; no migration needed.

## How to verify

```bash
rg -n 'cmdguard' --glob '*.go' cmd/                 # 4 files mention; 2 import core v4
go list -m github.com/larsartmann/cmdguard/v4        # pinned vs @latest
git -C /home/lars/projects/cmdguard log --oneline v4.2.0..v4.2.1
go run ./cmd/io-surface --help && go run ./cmd/io-surface doctor
```
