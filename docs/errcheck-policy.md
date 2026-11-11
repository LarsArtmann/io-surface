# errcheck policy

Relocated from AGENTS.md 2026-11-11 (agents-md-size gate: 220-line cap).
Decided 2026-11-08, tracked as TODO #123.

stdout/stderr report writes are deliberately unchecked and carry `_, _ =`
at the site:

- The exit-code contract (0 clean / 1 gate findings / 2 usage-internal)
  reflects the GATE result, never a printer hiccup.
- A piped reader closing early kills the process via SIGPIPE per Go runtime
  convention — the correct CLI behavior.
- In-memory `strings.Builder` assembly (render/scaffold/manifest message and
  document building) is provably non-failing and shares the same `_, _ =`
  convention (extended 2026-11-08: the application sweep found 20+ bare
  sites golangci's max-same-issues cap of 3 had been revealing three at a
  time).

No `.golangci.yml` exists by design and CI does not run golangci-lint; do
not "fix" the `_, _ =` sites or add a config without reopening the decision.
Everything else (real error returns, including test helpers like
`fmt.Sscanf` parses and `os.WriteFile` fixture writes) is checked.
