# dnsblockd v1 → v1.2 migration — decision packet

**Prepared:** 2026-11-08 (remainder plan N4 / TODO #130) · **Status:** packet
only; nothing in dnsblockd has been touched by this session. The session is
owner-scheduled (2026-11-08); this page is the artifact the session runs on.

## Decision this packet asks for

One call, made at session start:

> **Per-site judgment WITH the owner** (default, recommended), or
> **mechanical auto-migrate + owner reviews the diff**?

Everything else below is scope and risk, not choice: the runbook
(`/home/lars/projects/dnsblockd/docs/architecture/io-surface/MIGRATION-v1.2.md`,
T19) already fixes the mechanics.

## Scope (what the session does)

| Item                                                                | Count       | Judgment needed          |
| ------------------------------------------------------------------- | ----------- | ------------------------ |
| Undeclared sites failing the gate (`writeUnbound`, `cssHandler`)    | 2 errors    | declare vs baseline each |
| Extraction-blind declared symbols                                   | ~10 infos   | `extractable: false` + rationale, or leave |
| v1 claims ported to v1.2 shape: per-site `discipline` tags + `exception` prose replacing the global `disciplines.cue` constraints | 134 claims  | per-site "why this write is safe" — the reason the owner is in the room |
| New v1.2 vocabulary: `criticality` + `durability` per claim         | 134 claims  | what breaks / what a lost write costs |
| `version: "io-surface/v1.2"` in the manifest                        | 1 line      | none (mechanical)        |
| Nix wiring: retire/replace `check-io-surface.sh` with the CLI gate (`io-surface check` as the Nix check) | 1 flake edit | keep-or-retire call for the old script |

Effort estimate: **2–3 h** with the owner (134 sites at ~45–60 s of judgment
each is the bulk); **~30 min** if mechanical-then-review is chosen.

## Risks

1. **Judgment fatigue** — 134 sequential micro-decisions degrade quality
   after ~an hour. Mitigation: process per-family batches (file-write first —
   that's where `durability` bites), break after each family.
2. **The atomic-write question** (runbook §2.3): `writeUnbound`'s
   `bufio.NewWriter` around a durable file write forces the
   `atomic-write` discipline call early — take it first, it calibrates the rest.
3. **In-flight work**: dnsblockd's tree had uncommitted
   `internal/journal/retention.go` changes at packet time (2026-11-08 06:50,
   verified how: `git status --short` in dnsblockd). The session must start
   from a clean or explicitly-rebased tree.
4. **Schema-copy drift**: dnsblockd carries its own `schema.cue` copy; the
   migration must ship the byte-identical v1.2 schema (copy from
   `internal/scaffold/schema.cue` at the release tag), or `check` refuses.

## Unblock path for #27 (reverse mode + Nix check + retire `tools/io-extract`)

#27 stays BLOCKED on the migration, not on the lock: reverse mode in
`check-io-surface.sh` only matters while that script is authoritative. Once
the CLI gate replaces it (scope row 6), #27's script-side work collapses to
"retire the script" and `tools/io-extract` retirement becomes a plain
deletion review. Order: migrate (this session) → flip the Nix check → then
#27 is a 30-min cleanup, no longer 2 h.

## Schedule ask

The session is scheduled "2026-11-08" without a window. The packet needs a
2–3 h block (or 30 min if mechanical). Owner answers in the session kick-off;
the TODO row #130 records the outcome.
