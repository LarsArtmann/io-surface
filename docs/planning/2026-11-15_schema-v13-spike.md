# Schema v1.3 spike — design record (implementation unscheduled)

**Date:** 2026-11-15 · **Scope:** DESIGN ONLY (owner-ratified vocabularies; TODO
rows #135/#136 own the implementation session) · **Feeds:** the one combined
v1.3 train (ROADMAP Q5 + Q6 decisions, both recorded 2026-11-08)

## 1. Version gate

- `schema.cue`: `version?: "io-surface/v1" | "io-surface/v1.1" | "io-surface/v1.2" | "io-surface/v1.3"`.
- `manifest.checkVersion` accepts v1.3; `TestLoad_VersionRefusal` gains a
  v1.3-accepts case next to the shipped-version cases.
- Scaffold stamps `io-surface/v1.3` for NEW manifests; `init --force` never
  touches the version of an existing manifest (current behavior, unchanged).
- v1/v1.1/v1.2 manifests load unchanged: both additions are optional fields
  with additive defaults (absent = current behavior).

## 2. `claims.ownership` (Q6)

```cue
// after durability in #Claims
ownership: "self" | "caller" | "helper" | *"self"
// names the owning helper when ownership == "helper"; required in that
// case, forbidden otherwise (CUE: ownership: "helper" implies helper: string)
helper?: string
```

- Default-on-absent is `self` = today's behavior; no site-shape change, no
  family change (D7: one optional enum field + one conditional string).
- `closer-missing` semantics:
  - `self` — current behavior (this function must close it).
  - `caller` — drop the finding when the site's handle is returned or stored
    (makes the existing mechanical transfer suppression a DECLARED policy that
    also covers shapes the AST cannot see: registries, pools, long-lived
    sessions).
  - `helper` — drop findings on sites whose handle reaches the named helper
    call (e.g. `ownership: "helper"`, `helper: "util.Drain"`); the helper name
    is matched against the function's call expressions by receiver-insensitive
    base name (same canonicalization as `internal/match`).
- Enforcement: the claim is POLICY (advisory modulation like criticality), not
  a suppression directive; the directive stays the fine-grained tool.
- Render: NO report column in v1.3 (goldens stay byte-stable); ownership
  appears in SARIF result metadata only when it changed a verdict.
- FP mapping (fleet triage): cache stores → `caller`; pool hand-off →
  `helper`; long-lived session handles → `caller`.

## 3. Runtime-power capabilities (Q5)

- `#CapabilityName` grows from the mapped three to the closed ten:
  `FILES | NETWORK | EXEC | REFLECT | RUNTIME | SYSTEM_CALLS | UNSAFE_POINTER | OPERATING_SYSTEM | READ_SYSTEM_STATE | MODIFY_SYSTEM_STATE`.
  (Exact engine names from the 2026-11-06 blind-spot run; `MODIFY_SYSTEM_STATE/*`
  sub-variants normalize to the base name at absorb.)
- Package-level ONLY: `capabilities.cue` entries may carry them in
  `direct`/`deep` sets; the SITE contract never grows runtime-power
  disciplines (the transitive-everywhere noise class stays out).
- `internal/capability` grows the same enum in the same commit (single source;
  the existing sync test extends to the runtime-power set — enum + mapping
  must move together or the package does not compile).
- `absorb`: stores known runtime-power names as package-level facts; unknown
  engine names keep the M34 loud-tier (reported, not stored).
- Advisory shadow tier: `shadow-capability` semantics UNCHANGED for mapped
  caps. NEW advisory info finding `io-surface/runtime-power-direct` (package
  whose DIRECT set contains a runtime-power cap) — artifact-only until a D4
  promotion (same evidence bar as every detector rule; expected fleet volume
  is large, so it ships at info and stays there absent FP≈0 evidence).
- `capability-drift` semantics unchanged (absorbed set ≠ committed set).

## 4. Taxonomy freeze accounting

Families (10) and disciplines (7) are UNTOUCHED — no family/schema version
beyond the bump itself. The capability enum grows under its own freeze
process (Q5 decision): enum + mapping + test in one commit.

## 5. Session test plan (for #136)

1. Version: v1.3 accepts; v1.2 manifest with ownership field → refuses (field
   unknown to v1.2 closed schema) — pins the "bump required to use it" rule.
2. Ownership: closer-missing drops on `caller` (handle stored) and `helper`
   (named helper reached); keeps on `self`; `helper` without `helper` name is
   a validation error.
3. Capabilities: runtime-power set round-trips through absorb fixture;
   `runtime-power-direct` fires advisory on a synthetic package; sync test
   proves enum/mapping unity.
4. Goldens: no lint/gate golden changes EXCEPT the new advisory rule's fixture
   rows; `-update` re-pin reviewed word-by-word.
5. Round-trip: scaffold fixture → load → check → render stays green with the
   v1.3 stamp (render output unchanged).

## 6. Non-goals (explicit)

- No per-site runtime-power disciplines.
- No render column for ownership.
- No change to detector severities, pairing order, or the baseline contract.
- No dnsblockd coupling: its migration can adopt v1.3 later independently
  (see `2026-11-15_dnsblockd-manifest-first-regen-packet.md` — that blocker is
  a WRITER issue, orthogonal to schema v1.3).
