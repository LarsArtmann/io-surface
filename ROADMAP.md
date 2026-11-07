# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. Prove the tool where it was born (adoption before features)

The gate has teeth, but a manifest that is not gated rots. Until real repos
block on it, io-surface is a very good documentation linter.

- Wire the gate into the pilot repos (dnsblockd reverse gate + Nix check;
  InboxClean CI) and retire the duplicate extractor there
- One honest pass over dnsblockd's hand-ported manifest (regenerate facts,
  re-port the prose into claims)
- BuildFlow provider registration so `./bin/dev` runs the gate fleet-wide
- Register the repo in the fleet inventory (`meta` / stack-selection)
- Publish a real family distribution from ~10 daemon repos

### 2. Consumer outputs that justify keeping the manifest alive

Detection works; the reason for a second repo to keep the manifest alive does
not exist yet. Cheapest fix first.

- systemd hardening generator: `ReadWritePaths`/`ReadOnlyPaths`/`ProtectSystem`
  derived from `file-write`/`file-read`; `RestrictAddressFamilies` from
  `listener`/`outbound`/`ipc-subprocess`; diff against the live unit (evo-x2,
  read-only)
- Published `io-surface.json` + JSON Schema export, freshness-gated
- `io-surface diff a.cue b.cue` → release-note changelogs
- `io-surface explain <id>`: family, calls, claims, what would change the verdict
- D2 diagram rendering (families → containers, sites → nodes)
- Test-gap map: durable `file-write` sites ↔ crash-recovery tests
- Baseline burn-down section in `render` (count, oldest entry, age)

### 3. Raise the extraction ceiling

Every "complete inventory" claim is bounded by what resolved imports can see
(the token tier is 67% of dnsblockd's sites). Either add the precise tier or
stop calling the inventory complete.

- Opt-in `--precise` tier using `golang.org/x/tools/go/packages` (boundary
  recorded in Theme 5: the v0.3.0 detector layer consumes resolved-tier AST
  captures only — precise stays this theme's own track)
- Seams declared as data (manifest `via`) instead of a hardcoded table;
  cross-package seams
- Extract `--stats` face for outlier packages; confidence distribution visible
  in every report

### 4. Unified I/O insights: sites (hermetic) × reachability (Capslock)

Decision record for the io-surface × Capslock unification (plan:
`docs/planning/2026-11-06_06-02_unified-io-insights-pareto-plan.md`; evidence:
`docs/status/2026-11-06_06-45_capslock-v034-blind-spot-run.md`). io-surface
answers **where** I/O sits — per-call-site, hermetic, error-tier. Capslock
answers **what is reachable** — SSA callgraph through dependencies, transitive,
noisy. The unified project keeps ONE manifest with TWO fact classes: `sites`
(where, declared, gateable) and `capabilities` (pkg → reachable capability set,
advisory). Claims justify both — including I/O that hides behind wrappers, the
hole neither engine fills alone.

Landing sequence: pinned external capslock binary (CI-only + local dev shells,
never imported) → `io-surface absorb` ingests its JSON → manifest gains a
closed-enum `capabilities` section (schema v1.1) → advisory rules
`shadow-capability` (capability with zero declared sites, not baselined) and
`capability-drift` (absorbed set ≠ committed set) → GitHub-only CI job keeps
the facts fresh. Every graph-tier finding starts info/warning and never exits 1;
promotion to gate goes through an explicit `enforcement` policy decision.

Deliberately NOT this theme: embedding capslock's analyzer (unstable internal
API, heavyweight deps would break hermeticity), and a dashboard (render stays
markdown). The first blind-spot run validated the shape: 10/11 real direct
sinks were already declared by the hermetic extractor; the false-positive
classes observed (phantom NETWORK edges from SSA conflation) are exactly why
the capability tier must be advisory.

Note the boundary with Theme 3: `--precise` would add _in-tree type facts_ to
the hermetic extractor; Capslock adds _transitive reachability including
dependencies_ from an external engine. Different economics, different trust
tiers, different tools — neither subsumes the other.

### 5. Detector composition boundary + SARIF policy (ADR, T24)

Decision record for how the io-bug-radar detector layer composes with heavier
analyzers, and when SARIF output reaches a consumer (plan
`docs/planning/2026-11-07_05-02_io-bug-radar-pareto-plan.md` T24; evidence:
`docs/status/2026-11-07_08-08_io-bug-radar-p0-p2-detectors-fleet-proven.md`).

**Considered:**

1. **golangci-lint plugin / focus-filter** — surface detector findings through
   golangci's pipeline. Costs: an interlock dependency on the fleet's
   golangci versions, a plugin build per version matrix, and zero new
   information (the findings already exist as SARIF). Rejected.
2. **Audit module** (go-finding consumer reading io-surface SARIF) — viable
   later, but today there is no audit-module deployment to feed; building the
   bridge before the consumer is YAGNI. Deferred.
3. **Standalone CLI + SARIF as THE integration contract** (chosen, and what
   v0.3.0 shipped): `io-surface lint --format sarif` emits stable rule IDs
   (`io-surface/err-dropped`, …), confidence 1.0, severities from the
   evidence-gated registry. Any consumer (LSP, go-finding pipeline, code
   scanning) integrates without io-surface caring. The Theme 3 `--precise`
   boundary stays untouched: detectors are AST-pattern observations on
   already-resolved call sites; type-fact work remains Theme 3's own track.

**SARIF upload policy (unblocks TODO #61):** upload findings to a
code-scanning-style consumer only for rules promoted ABOVE info (T17 bar:
FP≈0 on the fleet run). Until then, SARIF goes to CI artifacts, not gates —
an info-severity flood in a scanning UI trains people to ignore the UI. The
promotion evidence table in the status report is the gate for flipping any
rule: today, zero rules qualify, so the policy resolves to "artifact-only"
for every current rule. Revisit per rule, not per release.

**Owner decision recorded:** standalone + artifact-only now; audit-module
bridge deferred until a consumer exists; golangci composition rejected.

## Open questions (owner decisions, not tasks)

- **Q1 — which repo gets the first _blocking_ gate?** dnsblockd has the richest
  manifest but was lock-held; InboxClean is free but claim-less (a smoke test,
  not a proof). (Birth report §g-Q1)
- **Q2 — flake now, or stay Go-module-only until a second repo gates on it?**
  A flake buys `nix build` + sandbox-native gating and costs a row in the
  fleet's pin matrix. GitHub CI already satisfies "own CI so it can't rot"
  for the tool itself. (Birth report §g-Q2)
- **Q3 — may the weak (heuristic) tier ever fail a build, or is it advisory
  forever?** Opt-in `--strict-weak` per repo is the compromise candidate; this
  is a false-positive-budget policy call. (Birth report §g-Q3) For the
  DETECTOR tier the answer already shipped (D3/D4, v0.3.0): token-tier sites
  never produce detector findings structurally, and every rule ships at info
  with fleet-evidence-gated promotion (Theme 5's SARIF policy keys off the
  same promotion bar).
- **Q4 — purge the ~67 MB binary blobs from git history, or keep them?** The
  auto-commit daemon committed a compiled `cmd/io-surface/io-surface` twice
  (blobs of 35,180,278 and 35,181,139 bytes, commits `05874f3` / `e15bf62`,
  2026-11-06 05:10/05:13). The forward leak is closed: untracked + gitignored
  (`/cmd/io-surface/io-surface`, `/bin/`), rebuild-proven. Tagged `v0.1.0`
  predates the blobs, so tagged module zips are clean; but `@latest`/`@master`
  resolves to HEAD, and a tracked 35 MB file ships in every proxy zip of
  master. **Purge:** `git filter-repo --invert-paths --path
  cmd/io-surface/io-surface` — rewrites every SHA from `05874f3` onward,
  requires force-push to both remotes and fresh clones everywhere; status
  reports citing those SHAs go stale. **Keep:** ~67 MB one-time clone cost,
  zero SHA breakage, but every zip of master carries the binary forever.
- **Q5 — do Capslock's runtime-power capabilities ever become first-class
  vocabulary, or stay out-of-scope forever?** The blind-spot run showed
  REFLECT / RUNTIME / SYSTEM_CALLS / UNSAFE_POINTER / OPERATING_SYSTEM /
  READ_SYSTEM_STATE / MODIFY_SYSTEM_STATE reach into every package
  transitively — noise as per-site disciplines, but real information as
  package-level reachability. Deciding "new discipline = schema version bump"
  (taxonomy freeze applies) vs "permanently out-of-scope" gates whether
  `absorb` stores or drops them. First-class would also answer MODIFY/READ
  _SYSTEM_STATE for systemd-hardening consumers (Theme 2).

## Non-goals

Things we are deliberately NOT pursuing and why:

- **Benchmark suite / perf gate:** load-sensitive numbers must never share an
  exit code with a deterministic gate. Separate tool, separate economics.
- **Auto-fixer:** disciplines classify; they never rewrite code.
- **Dashboard:** the manifest is data; render it however you like.
- **Multi-language extractors (today):** the taxonomy and manifest contract
  generalize; the extractor is Go-first. Revisit on a real demand signal.
