# Dogfooding Status — io-surface eating its own food

**Scope.** Verified live 2026-11-07 19:52 with: `go run ./cmd/io-surface check --root .`
(exit 0), `go run ./cmd/io-surface lint --root .` (exit 0, 11 infos),
`init --force` + CI-filtered `git diff` (quiet → CI-WOULD-PASS),
`render --doc … --check` (exit 0), `gofmt -l .` (empty), `gh run list` on
io-surface/gogenfilter/InboxClean, `git rev-list --count` vs both remotes,
and per-repo `git status`/`git diff` in the three pilots. NOT verified:
Forgejo-side execution (runs no workflows), CI on the current unpushed HEAD,
InboxClean billing state, authorship of dnsblockd's in-flight edits (left
untouched — concurrent-session rule). This report covers DOGFOODING only;
the full repo snapshot lives in the 09:09 retrospective of today.

## a) Fully done

| Item                                                                                                                                                                          | Verified how                                                                                 |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Self-gate green: 16 declared / 16 extracted (16 resolved, 0 token), 2 baselined, 0 err/0 warn/0 info                                                                          | `check --root .` → exit 0                                                                    |
| Real CI green ON THE LAST PUSHED commit (5b4432b): all 9 gate steps incl. facts-freshness, doc-fresh+dprint, self-SARIF, docs-gates, capslock absorb, public-install tripwire | `gh run list -R LarsArtmann/io-surface` → run 42353726369, success, 2026-11-07T08:02Z, 1m21s |
| Facts manifest fresh: refresh drifts only the tolerated `commit:` audit line                                                                                                  | `init --force`; `git diff -I'^commit:' -I'^date:' …` quiet                                   |
| Inventory doc freshness + gofmt clean                                                                                                                                         | `render --doc … --check` exit 0; `gofmt -l .` empty                                          |
| T10 dogfood bug is genuinely fixed: `symbolVisible` N+1 → per-unique-path read cache                                                                                          | `internal/check/location.go:20-36`; the retro's "FIXED" claim matches the code               |
| gogenfilter pilot landed: gate step + manifest committed by daemon                                                                                                            | `git -C gogenfilter show a6e78aa --stat` (ci.yml + docs/architecture/); clean tree           |
| Fleet-scale self-use: 182-repo fleetscan, 5,802 production sites                                                                                                              | `scripts/fleetscan.sh`; evidence report 2026-11-07_08-08                                     |

## b) Partially done

| Item                         | Gap                                                                                                                                                                                                                                                                                    |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| InboxClean pilot             | ALL FOUR pilot files sit **uncommitted** in its working tree (ci.yml pin v0.1.0→v0.3.0, claims.cue, schema.cue, pdf.go fd-leak fix). "Done" is true only on disk. Verified: `git -C InboxClean status --porcelain`.                                                                    |
| Self-lint dogfood            | 11 infos on our own tree, exit 0 (info-only by design): 5 write-non-atomic, 5 io-in-loop, 1 stat-then-open. None tracked as TODO rows. One is a post-fix residual the detector cannot see through (`location.go:28` flags the loop despite the read cache — loop-shape, not dataflow). |
| lint in CI                   | ABSENT. The detector layer never runs on io-surface in CI; T10 was a one-off session act. Retro §f2 names it; no TODO row exists yet. The SARIF-artifact-only policy (TODO #61, T24 ADR) already decided the shape — one step to wire.                                                 |
| gogenfilter gate observation | Wired but never seen running: GitHub CI's last workflow run is 2026-11-04; the pilot commit is unpushed (Forgejo runs no workflows). Verified: `gh run list -R LarsArtmann/gogenfilter`.                                                                                               |

## c) Not started

- dnsblockd v1→v1.2 judgment migration (owner call pending; AGENTS.md documents
  the decision point). NB: `nix/checks/default.nix` + `scripts/check-io-reverse.sh`
  are modified in its tree right now — looks like another session's #27 work;
  not mine to touch.
- Self-lint CI step (no TODO row either).
- Next pilot cohort (go-taskqueue 37 findings, retro §f28).

## d) Totally fucked up

**The push ritual is dead.** `master` is **3 commits ahead on BOTH remotes**
(`git rev-list --count origin/master..master` = 3, same for github). The
CI-green run proves 5b4432b, but HEAD — today's two status reports and the
verification commits — is invisible to CI. The daemon commits; nobody has
pushed since 5b4432b. InboxClean's uncommitted pilot diff is the same failure
mode one level deeper.

## e) What we should improve

1. Make "push both remotes" an end-of-session ritual (or wire the Forgejo
   server-side mirror, TODO #41, and push once).
2. Commit pilots in their own repos the moment they verify green — a pilot
   claim without a commit is a rumor.
3. Track the 11 self-lint infos (TODO row or `unused-baseline`-style ledger);
   note the `location.go:28` residual as a known loop-shape limitation.
4. Wire `lint --format sarif` into our CI as artifact-only per the #61 policy.

## f) Top #10 next (dogfooding-focused; full backlog in the 09:09 retro §f)

1. `git push origin master && git push github master` (3 commits, today's proof).
2. Commit InboxClean's 4 pilot files in InboxClean (owner-policy question §g).
3. Add CI step: `go run ./cmd/io-surface lint --root . --format sarif > lint.sarif` + artifact upload.
4. Push gogenfilter, then watch its CI run green (retro §f8).
5. TODO row for the 11 self-lint infos + the location.go:28 residual note.
6. Watch InboxClean CI once billing allows; record the green link (retro §f10).
7. dnsblockd v1→v1.2 migration — needs the owner call (retro §g-Q1).
8. gogenfilter claims pass for its 5 sites (retro §f24).
9. Onboard go-taskqueue as the next fleet pilot (retro §f28).
10. Re-check facts freshness after the daemon absorbs my `init --force` refresh (expected: only `commit:` line).

## g) Top #1 question I cannot answer myself

**Pilot commit + push authority (re-ask of retro §g-Q3, still unanswered):**
may I commit and push in the PILOT repos (InboxClean's 4-file diff, gogenfilter's
branch), or do you want eyes on those diffs first? Today's evidence says the
daemon does not save us: uncommitted pilot work and an unpushed HEAD mean
every "done" claim is currently unverifiable by CI. A yes/no unblocks items 1–4.
