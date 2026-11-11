# Check semantics worth remembering

Relocated from AGENTS.md 2026-11-11 (agents-md-size gate: 220-line cap).
The gate itself is `internal/check`; rule constants there are the stable
contract.

- Only **resolved**-confidence extraction produces error findings; the token
  tier is advisory by construction (and must stay that way — birth-status
  report §g-Q3).
- `location-drift` is info, never a failure (line numbers are derived data).
- Exact-ID pairing happens BEFORE canonical pairing: two same-named methods
  on different types canonicalize identically (`AllowlistManager.Load` vs
  `PermanentAllowlist.Load`) — deliberate, documented in
  `internal/match/collision_test.go`.
- `_test.go` files are never manifest sites; any path containing
  `/testutil/` is flagged `testinfra`. The walk skips dot-dirs, `vendor`,
  `node_modules`. Default scan dirs: `cmd`, `internal` (`--dirs a,b`
  overrides; a narrowing that excludes declared sites adds a `scope-hint`
  info naming the flag as the cause).
- The CI facts-freshness gate ignores the audit `commit:`/`date:` lines via
  `git diff -I` (`-I'^[[:space:]]*commit:' -I'^[[:space:]]*date:'`) — those
  two lines legitimately drift because HEAD always moves between the
  manifest-refresh commit and the CI run. Do not "simplify" them away —
  without them the gate is structurally red.
- Facts-freshness cheat-sheet: the gate compares the CURRENT TREE against
  the COMMITTED manifest (regenerate from tree, diff vs HEAD's facts.cue) —
  so it fails whenever working-tree Go edits moved or added I/O sites before
  `init --force` ran, and it stays green for pure comment/doc edits; a
  failure pasted from an earlier run may already be fixed on disk, so
  re-run `go run ./cmd/io-surface check --root .` before acting.
- Baseline lifecycle is enforced: an entry suppressing nothing today warns
  (`unused-baseline` — only-shrinks), an expired `expires:` date fails
  (`baseline-expired`, error). The canonical-key hiding of two
  same-base-name sites behind one entry remains — the warning is the
  guardrail, not a fix.
