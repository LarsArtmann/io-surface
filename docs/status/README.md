# Status reports

Point-in-time snapshots of this repo's work: what shipped, what broke, what
stayed open. Newest first. A report is history — it is never updated to stay
"current"; corrections go in a NEW report that names its predecessor (the
`§errata` convention when the divergence is from a plan).

| Report                                                                              | One-line summary                                                                               |
| ----------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `2026-11-06_21-30_unified-gate-execution-whole-todo-run-and-self-review.md`         | Whole-TODO run: v0.2.0 shipped, provider fleet-live, self-review with tag-audit finding        |
| `2026-11-06_14-25_fleet-io-surface-distribution.md`                                 | First fleet run through the BuildFlow provider; 7/8 Go repos clean, InboxClean 3 real findings |
| `2026-11-06_08-21_unified-capabilities-shipped-t20-inflight.md`                     | Capabilities end-to-end shipped and dogfooded; T20 (extractor families) in flight              |
| `2026-11-06_07-59_gate-failure-triage-and-ci-bar-verification.md`                   | Gate failure triage and what the CI bar actually verifies                                      |
| `2026-11-06_06-45_capslock-v034-blind-spot-run.md`                                  | Capslock v0.3.4 evidence run: blind spots, phantom edges, why the tier is advisory             |
| `2026-11-06_06-33_pareto-follow-through-cross-repo-gates-and-manifest-hardening.md` | Parallel-session lane: id-form rule, baseline lifecycle, manifest test suite                   |
| `2026-11-06_05-42_cmdguard-adoption-and-sentinel-contract-sprint.md`                | cmdguard v4 CLI adoption; sentinel error contract; handlers moved to commands.go               |
| `2026-11-06_05-36_capslock-compare-and-binary-in-history.md`                        | First capslock compare run; the committed-binary-in-history finding                            |
| `2026-11-06_04-27_io-surface-pareto-execution-install-to-sarif-and-pilot-gates.md`  | Pareto P0 execution: install path, SARIF emission, first pilot gates                           |
| `2026-11-06_02-33_self-dogfood-and-force-safety.md`                                 | Self-dogfood audit; `init --force` scoped to facts.cue only                                    |
| `2026-11-06_01-20_io-surface-publish-mirror-and-docs-audit.md`                      | Publishing (Forgejo + GitHub mirror) and the docs audit                                        |
| `2026-11-06_00-14_io-surface-birth-bench-matrix-and-first-pilot.md`                 | The birth report: extractor, checker, renderer, bench matrix, first pilot                      |
