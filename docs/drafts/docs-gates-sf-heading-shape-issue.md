> [!NOTE]
> This filing was drafted by GLM-5.3-flash (via Crush) from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

TL;DR: a status report whose §f heading is written `## §f — …` instead of the canon `## f)` is invisible to freshness/audit-all/routing — `SectionFRe` requires `f)` directly after the hashes, and the gates then report "has no §f section" for a report that has one. Ask: warn (at minimum) when a heading looks like a §f section that the canon regex rejects.

## Problem

`SectionFRe` is the single shared definition of where §f begins (the comment on it says freshness and routing must agree on one spelling — correct instinct):

`// internal/docsfreshness/reports.go:18 (master c81685b, 2026-11-22)`

```go
SectionFRe = regexp.MustCompile(`(?i)^#{2,}\s*f\)`)
```

A heading written `## §f — Follow-ups` does not match (the `§` sits between the whitespace and the `f`), so `CheckReport` finds no §f items and the gate prints:

`// internal/docsfreshness/audit.go:119`

```
docs-freshness: <report> has no §f section — nothing to check
```

— for a report that carries ~95 open forward items in that section.

## The incident that cost real harvest

Two io-surface status reports (`docs/status/2026-11-15_15-01_*` and `docs/status/2026-11-21_19-55_*`) used `## §f —` for 6 days. The docs-gates battery stayed 5/5 PASS (FRESH) the whole time while the reports' forward-looking items were machine-unharvestable by freshness, audit-all, and routing alike. Found only by hand; renamed to the canon `## f)` on 2026-11-21; the battery then saw the items immediately.

## Ask

Cheapest fix: a warn-level check for near-miss shapes — a heading matching something like `^#{2,}\s*§?\s*f\b` (or a fuzzy "f-section" probe) that the canon `SectionFRe` rejects gets a warning naming the report and the line. Full fix: tolerate `§` in `SectionFRe` itself. The warn-only shape keeps the canon strict (one spelling, per the shared-definition comment) while making the blind spot loud instead of silent.

💘 Generated with Crush
