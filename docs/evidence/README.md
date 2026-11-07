# Fleet evidence provenance

fleetscan-2026-11-07-v031.json
Produced by: `./scripts/fleetscan.sh docs/evidence/fleetscan-2026-11-07-v031.json`
(from repo root, io-surface @ master 2026-11-07, post-v0.4.0 tree with the
full 10-rule detector ruleset; version const 0.4.0).
Corpus: all 160 pma-discovered Go projects (read-only scan; `lint` face,
report-only, no manifest required).
Headline: 160/160 scanned, 91 with findings.
Per rule: io-in-loop 351, write-non-atomic 340, stat-then-open 50,
client-no-timeout 13, err-dropped 11, ctx-dropped 3, defer-in-loop 1,
tiny-buffer 0.

Policy: every fleet-scanned number cited anywhere in the repo must live here
(or be re-derivable by the named command). /tmp is not evidence.

fleetscan-2026-11-07-t16-refined.json
Produced by: `./scripts/fleetscan.sh docs/evidence/fleetscan-2026-11-07-t16-refined.json`
(post-T16 same-path refinement). io-in-loop 351 -> 248 (-103, the per-item
range-bound class); all other rules byte-identical to v031; 88 repos with
findings.
