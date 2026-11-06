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

- Opt-in `--precise` tier using `golang.org/x/tools/go/packages`
- Seams declared as data (manifest `via`) instead of a hardcoded table;
  cross-package seams
- Extract `--stats` face for outlier packages; confidence distribution visible
  in every report

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
  is a false-positive-budget policy call. (Birth report §g-Q3)
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

## Non-goals

Things we are deliberately NOT pursuing and why:

- **Benchmark suite / perf gate:** load-sensitive numbers must never share an
  exit code with a deterministic gate. Separate tool, separate economics.
- **Auto-fixer:** disciplines classify; they never rewrite code.
- **Dashboard:** the manifest is data; render it however you like.
- **Multi-language extractors (today):** the taxonomy and manifest contract
  generalize; the extractor is Go-first. Revisit on a real demand signal.
