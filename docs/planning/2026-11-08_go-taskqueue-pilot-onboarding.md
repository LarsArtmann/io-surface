# go-taskqueue — io-surface pilot onboarding plan

**Prepared:** 2026-11-08 (remainder plan N20 / fresh numbers replace the
"37 findings" figure quoted from the 2026-11-07 fleet scan — the final
ruleset counts **53** advisory findings, verified how:
`go run github.com/larsartmann/io-surface/cmd/io-surface@v0.5.0 lint --root
~/projects/go-taskqueue` on 2026-11-08).

## Why this pilot next

go-taskqueue is the largest unaudited first-party consumer in the fleet
(multi-module: 12+ leaf `go.mod`s, unusual layout — every internal package is
its own module). Both BuildFlow integration questions are already answered
for it by the N3 evidence: the provider's module fan-out reaches the leaf
modules and the radar fires end-to-end (53 findings through
`buildflow -s io-surface`, identical count through the pinned CLI). Onboarding
exercises nothing new — it repeats the proven InboxClean/gogenfilter loop on
the hardest layout.

## Findings class table (v0.5.0 ruleset, all advisory info)

| Rule                        | Count | Class summary                                                                        |
| --------------------------- | ----- | ------------------------------------------------------------------------------------ |
| `io-surface/io-in-loop`     | 32    | N+1 shapes: per-item `filepath.Glob`/`os.Stat`/HTTP GET inside loops                  |
| `io-surface/write-non-atomic` | 20  | direct final-path writes without temp+rename (state files, caches)                    |
| `io-surface/client-no-timeout` | 1  | one `http.Client` literal without a `Timeout`                                          |

All 53 are info severity: nothing fails, everything is a candidate for the
per-site `//io-surface:ignore <rule> <reason>` directive or a real fix.

## Onboarding steps (one session, ~60–90 min)

1. **`init`** — `io-surface init --root ~/projects/go-taskqueue` scaffolds
   `docs/architecture/io-surface/{facts,claims,baseline,schema}.cue` from the
   extracted sites. The multi-module layout is handled: extraction walks the
   union of `cmd/` + `internal/` from the repo root (per-leaf-module scans
   are a BuildFlow fan-out detail, not a CLI constraint).
2. **claims pass (the judgment hour)** — for each extracted site: resource,
   trigger, discipline; ~53 sites + the write-family sites get
   `durability` (`ephemeral` caches vs `durable` state) — the same
   per-site conversation as dnsblockd, smaller.
3. **triage the radar findings** — fix the 1 timeout-less client (trivial),
   batch-decide the 20 write-non-atomic sites (queue state writes almost
   certainly want atomic-write semantics — real bug potential), and treat the
   32 io-in-loop infos per-site (many will be legit batching loops that
   deserve a documented ignore).
4. **CI pin** — add the gate step at the then-current release tag
   (`@v0.5.0` or later; the InboxClean ci.yml lines are the template).
5. **gate green** — `check` exits 0 with the manifest committed; the
   advisory lint stays artifact-only per the fleet policy until D4 promotion
   evidence exists.

## Risks

- **Multi-module facts.cue:** extraction is root-scoped, but BuildFlow fans
  per module — the manifest lives at the ROOT's `docs/architecture/io-surface/`
  and the provider's manifest lookup happens per leaf module. The N3 run
  proves findings flow, but the GATE face (manifest pairing) on a workspace
  where only the root has a manifest needs one verification during
  onboarding: adopt at root first, confirm `buildflow -s io-surface` pairs
  cleanly, and only then decide whether leaf modules need their own manifests
  (they should not — record the outcome here if they do).
- **Judgment volume:** 53 sites is InboxClean-class, not dnsblockd-class
  (134). One session suffices.

## Owner review

Executing this plan is an owner call (pilot queue order + session window);
tracked as the last question of the 2026-11-08 decision batch.
