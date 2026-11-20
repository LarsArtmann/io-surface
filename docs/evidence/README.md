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

Privacy: the `fleetscan-*.json` artifacts themselves are gitignored. They
carry absolute `/home/lars/projects/...` paths and 160+ local project names,
and the GitHub remote is a public mirror. The committed record is this file
(provenance + aggregate numbers), not the JSON; regenerate locally with the
command named per entry.

fleetscan-2026-11-07-t16-refined.json
Produced by: `./scripts/fleetscan.sh docs/evidence/fleetscan-2026-11-07-t16-refined.json`
(post-T16 same-path refinement). io-in-loop 351 -> 248 (-103, the per-item
range-bound class); all other rules byte-identical to v031; 88 repos with
findings.

cv-manual-verification-2026-11-17 (the CV repo's hand-adjudicated sample)
Source: the 2026-11-17 io-surface review waves in the CV repo (lint over all
11 Go modules, 284 sites), every hit hand-read and adjudicated; the full
triage lives in CV's docs/architecture/io-surface/claims.cue and
docs/status/2026-11-17_16-45_io-surface-review-wave-status.md. This is a
MANUAL verification sample (policy §2 vocabulary), not a fleetscan run.
Per rule (hits / FP-by-policy-definition / verified-fixed):

- io-in-loop 18 / 18 / 0 — every hit correct as written: distinct-target
  per-item loops 13 (retention sweep, per-document validation, fixture
  materializer, import/export loops, env overrides, parent-walk root
  finders, tmp sweep, asset paths), deliberate retry loops 5 (HTTP retry
  ×4, connect-with-backoff 1). Confirms the card's known-FP classes at
  100% of the sample; the one genuinely hoistable site (a per-iteration
  MkdirAll) had already been fixed pre-sample in wave 1.
- write-non-atomic 9 / 9 / 0 — every hit a DECIDED semantic, not a torn-
  state bug: O_EXCL refuse-overwrite 3 (lease claim = mutual exclusion,
  session/shortlist exports = never-clobber), O_APPEND journal 1,
  ephemeral scratch 4 (debug dumps ×2, empty-file perm seeding, mock
  fixture), dev generator 1 (perm-preserving stat-then-write). Strengthens
  the carve-out evidence for the pending classes below.
- stat-then-open 1 / 1 / 0 — the boundary-table generator's stat preserves
  the input file's permission bits before writing; removing the stat is
  the WRONG fix there (perm must be read before write).
  Caveat: N per rule is below the policy's W4 floor (30) — this sample
  strengthens the FP-class taxonomy, it does not qualify any promotion.
