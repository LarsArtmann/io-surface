# N19 decisions landed: v0.5.1 history purge + retraction, flake, strict-weak, InboxClean first gate, go-taskqueue onboarded

**Date:** 2026-11-08 ~16:50 CET · **Session:** continuation of the remainder
plan (N19 batch + owner Q1–Q4 implementations + go-taskqueue pilot) ·
**Predecessor:** `2026-11-08_08-48_remainder-n1-n21-v050-shipped-dnsblockd-ported-one-line-from-green.md`

## a) Headline

The owner answered the N19 Q1–Q4 batch (one question call). All four
decisions are now EXECUTED, not just recorded: io-surface's git history is
blob-free (v0.5.1 released with retractions), a fleet-shape flake.nix
builds and gates, `--strict-weak` ships, InboxClean runs the fleet's first
blocking gate at the clean pin, and go-taskqueue carries a green root
manifest. dnsblockd's "one open wound" was already closed by the concurrent
SUPERPLAN-VII session — verified, pushed, no action needed.

## b) Q4 — the purge (the big one)

- Owner chose PURGE with explicit force-push approval. Procedure: mirror
  backup at `/tmp/io-surface-backup-pre-purge.git` (54 MB) →
  `git filter-repo --invert-paths --path cmd/io-surface/io-surface --force`
  (one blob-only commit pruned, 213→212 commits, all 5 tags rewritten) →
  force-push master + tags to BOTH remotes → fresh-clone verification:
  4.0 MB, zero history references to the binary path.
- go-release skill compliance: tagged proxy zips are immutable, so the fix
  is NEVER a re-tag — v0.5.1 cut from the clean tip with
  `retract` directives for v0.2.0–v0.5.0 (rationale comments in go.mod).
  Verified: `go list -m github.com/larsartmann/io-surface@v0.5.1` serves
  from the proxy; pkg.go.dev HTTP 200; GitHub prerelease published; CI
  green on the release commit (`8d25580`, run 42493225143). `@latest` now
  resolves to the clean tag; explicit old pins keep resolving (retraction
  warning only) — InboxClean's pin was bumped anyway.

## c) Q1/Q2/Q3 — the quick strikes

- **Q1 (InboxClean first blocking gate):** their manifest already declared
  `enforcement: "gate"`, so the ratification landed as the pin upgrade to
  v0.5.1 (InboxClean commit `7147b75c`). Local gate green (109 declared /
  130 extracted, 0 errors).
- **Q2 (flake now):** fleet `go-standard` module (go_1_27, linux-only,
  proxyVendor, `subPackages = ["cmd/io-surface"]`), vendorHash fixed via
  `buildflow -s nix-hash-fix --fix` (never by hand). Verified:
  `nix build .#io-surface` + `nix run .# -- check --root .` → 0 errors.
  Commit `6cf531a`. AGENTS.md un-withholds the flake decision.
- **Q3 (`--strict-weak`):** `Options.StrictWeak` — one severity flip in the
  pairing pass; default stays advisory forever (the structural guarantee is
  opt-out, not gone). Pinned by `TestRun_StrictWeakEscalates`. Commit
  `db14508`.

## d) CI lessons (two real gate catches, both fixed)

- `db14508` failed CI's facts-freshness (the strict-weak line shifts landed
  after that commit; the refreshed facts rode the next commit) and
  `6cf531a` failed inventory-doc freshness — root cause: I refreshed
  `facts.cue` but never re-rendered `INVENTORY.md`. Fixed by
  `render --root . --doc docs/architecture/io-surface/INVENTORY.md`;
  **tip `1edcb79` is CI green** (run 42496345329). The freshness gates
  caught exactly what they exist to catch.

## e) go-taskqueue pilot onboarded (owner: "onboard now")

Per `docs/planning/2026-11-08_go-taskqueue-pilot-onboarding.md` (execution
record appended there): root manifest scaffolded (230 sites, 10 families),
gate green by construction; CI gate step added at `@v0.5.1`; the one
timeout-less client fixed WITHOUT the detector's literal advice —
`unixSocketClient` also serves the `/v1/watch` SSE stream, so a client
`Timeout` would have killed every watch stream. The fix is
`ResponseHeaderTimeout: 10s` + a documented `//io-surface:ignore
client-no-timeout …` directive (lint 53 → 52). The plan's multi-module risk
item resolved: BuildFlow's gate face pairs cleanly with a root-only
manifest (0 error findings); leaf manifests not needed. **Committed locally
(`5d223a42`), NOT pushed** — the repo is push-owner-gated and carried
another session's in-flight edits.

## f) Open items (harvested into TODO_LIST rows #139–#142)

None unharvested — see §g for the verifying commands.

## g) Notes for the owner

1. **GitHub BILLING blocks InboxClean CI** (and possibly other repos): the
   pin-bump run's jobs never started — "recent account payments have failed
   or your spending limit needs to be increased". The pin bump itself is
   locally verified (v0.5.1 gate green); the CI confirm is owed after
   billing is fixed.
2. **dnsblockd diff review** (the mechanical port's judgment debt): 92
   merged claims, 42 dropped keys, 21 UNREVIEWED fills, and now 3
   `claim-conflict` warnings the gate surfaces every run
   (`loadTrialBlocklist` atomic-write-on-outbound, `processor.Run` /
   `processor.writeMapping` buffered-write-on-stream-copy) — deck in
   MIGRATION-v1.2.md §5.
3. **go-taskqueue judgment hour:** 230 sites with empty claims blocks
   (InboxClean-class, not dnsblockd-class); the 20 write-non-atomic radar
   findings are the real-bug-potential batch.
4. **Stale-tooling trap closed:** the system `buildflow`/`docs-gates`
   binaries were stale and produced FALSE gate reds (an old provider
   invoked a docs-gates verb the current binary dropped, blocking a commit
   while the real battery was green). Fresh shims installed at
   `~/.local/bin` (PATH-wins); the trap and the refresh commands are in
   AGENTS.md sharp edges.
5. **Verified how (the session's claims):** every release claim above names
   its command inline; the reports' own numbers come from the cited runs.
