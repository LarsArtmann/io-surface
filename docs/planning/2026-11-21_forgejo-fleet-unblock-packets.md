# Forgejo fleet-unblock packets (TODO #40 / #41 / #42 / #155) — 2026-11-21

Four packets, one dependency chain: the owner drops the Forgejo API token
(#40), and #41 (push mirror) + #42 (visibility) + #155 (Forgejo CI) execute
server-side the same day. Everything here is VERIFIED against working
precedent — gogenfilter completed this exact migration on 2026-10-25 and its
runner/workflow facts are cited per section. The token itself is an OWNER
action (Forgejo UI → Settings → Applications); nothing in this packet needs
to exist on this machine first.

---

## §1 — #40: Forgejo API token packet (the unblock key)

**Ask the owner for:** one Forgejo API token with scope
`repository: write` + `admin` (admin is needed only for #155's runner
registration; the mirror + visibility calls in §2/§3 work with repository
write — issue both scopes in one token to avoid a second round-trip).

**Where it goes:** `~/.config/forgejo/token` (chmod 600), then either
`export FORGEJO_TOKEN=<token>` per shell or a `tea login add` profile
(server `https://forgejo.home.lan`, token name suggested: `fleet-automation`).

**Why safe:** the token stays on this machine (never in any repo, never in
CI secrets), grants only this instance, and is revocable in the same UI in
one click. The searches that failed on 2026-11-08 (env, netrc, tea config,
credentials store) stay authoritative — the 2026-11-21 session re-checked
nothing new exists.

**Unlocks immediately:** #41 §2 (mirror), #42 §3 (visibility), #155 §4 (CI).

---

## §2 — #41: push-mirror runbook (Forgejo → GitHub, ends manual dual-push)

**Precedent:** gogenfilter's Forgejo repo carries a server-side push mirror
to `git@github.com:LarsArtmann/gogenfilter.git` — the mechanism works on
this instance today.

**API path (owner token, one call):**

```
POST https://forgejo.home.lan/api/v1/repos/lars/io-surface/push_mirrors
Authorization: token <FORGEJO_TOKEN>
{
  "interval": "8m0s",
  "remote_address": "git@github.com:LarsArtmann/io-surface.git",
  "remote_user": "git",
  "ssh_key_id": <mirror-deploy-key-id>
}
```

**Server-UI path (fallback if the API rejects the ssh_key field):**
repo `lars/io-surface` → Settings → Mirror Settings → Push Mirror →
target `git@github.com:LarsArtmann/io-surface.git`, auth = the instance's
mirror SSH deploy key (the one gogenfilter's mirror uses — reuse it, one
GitHub deploy key serves all fleet mirrors), interval 8m.

**Acceptance test:** push a marker commit to Forgejo only, wait ≤ 10 min,
`git fetch github && git rev-parse github/master` matches. Then flip
AGENTS.md's "Remotes & publish" paragraph: `origin` only; the GitHub push
dies with the manual dance.

**Fallback if the mirror cannot be created:** keep dual-push (current
state); the packet changes nothing else.

---

## §3 — #42: visibility decision note (owner decides, one click)

Current: Forgejo `lars/io-surface` = **private**; GitHub mirror = **public**
(this is what `go install github.com/larsartmann/io-surface/cmd/io-surface@latest`
resolves through — #37/#44 verified that chain 2026-11-06).

|                  | Forgejo stays private (status quo)     | Forgejo goes public (gogenfilter pattern)                                        |
| ---------------- | -------------------------------------- | -------------------------------------------------------------------------------- |
| module consumers | unaffected (GitHub mirror serves them) | unaffected (same, plus proxy redundancy)                                         |
| leak surface     | none on the primary                    | the primary IS public: issues/commits become world-readable the moment they land |
| daemon commits   | invisible                              | live (incl. the `chore: auto-commit` noise history)                              |
| CI (post-#155)   | runs on the instance regardless        | identical                                                                        |
| cost of flip     | none                                   | none (reversible in one click)                                                   |

Recommendation: **keep private until #155's burn-in completes, then decide
once for the whole fleet** (gogenfilter went public as part of its cutover;
fleet-consistency beats per-repo exceptions). This note changes no behavior
today.

---

## §4 — #155: Forgejo-CI port plan (io-surface ci.yml → `.forgejo/workflows/ci.yml`)

**Runner facts (verified, gogenfilter migration 2026-10-25 + its
`docs/planning/2026-10-25_17-13_forgejo-ci-migration-pareto-plan.md` V1-V4):**
forgejo-runner v13.2.0, host executor on evo-x2, label **`native`**,
capacity 2, no Docker; unit PATH carries nix 2.34 + go 1.26.8 + node 24;
`GOTOOLCHAIN=local` forbids toolchain downloads (io-surface needs go 1.27 →
**ride the flake's go**, the gogenfilter pattern); RW cache only
`/mnt/buildcache/ci-runner-cache` with warm `GOMODCACHE`/`GOCACHE`;
egress to proxy.golang.org works; `actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2`
resolves fine on this instance.

**Port shape (gogenfilter consolidated-job pattern):** the capacity-2
instance-wide runner favors ONE job over io-surface's three GitHub jobs.
Windows: io-surface's ci.yml already has none (`runs-on: ubuntu-latest`
only) — the fleet "drop windows" policy is satisfied as-is.

Mapping (`.github/workflows/ci.yml` → `.forgejo/workflows/ci.yml`):

| GitHub job/step                                            | Forgejo port                                                                                                                                                              |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `setup-go` (go-version-file+cache)                         | drop — use the flake's go 1.27 (`nix develop -c go test ...` or flake-provided `go` on PATH), caches come from the unit env                                               |
| test job (build/vet/test-race/gofmt)                       | keep step-for-step inside the one job; `scripts/self-gate.sh` already mirrors this locally — consider invoking it directly                                                |
| Self-check + facts freshness + self-render + inventory-doc | unchanged (pure `go run`, no actions)                                                                                                                                     |
| dprint/check action                                        | `nix run nixpkgs#dprint -- check ...` (no GitHub action dependency)                                                                                                       |
| SARIF emit + upload-artifact                               | emit steps unchanged; artifacts: keep `upload-artifact` (works on Forgejo) or `cp` into a served dir — runner-host executor supports the action per gogenfilter probe.yml |
| docs-gates `go run …@v0.3.1`                               | unchanged                                                                                                                                                                 |
| markdown render audit                                      | unchanged (`scripts/check-md-render.sh`)                                                                                                                                  |
| capabilities job (Capslock)                                | second job or sequential block, `continue-on-error: true` (non-blocking parity), capslock via `go install` through the proxy egress (works per V4)                        |
| public-install job                                         | keep as second job — it MUST keep egress to proxy.golang.org (V4 verified)                                                                                                |

**Sequence:** owner lands the token (#40) → commit `.forgejo/workflows/ci.yml`
on Forgejo-primary → watch the first run on `evo-x2` green → burn-in ≥ 1
week of pushes → delete `.github/workflows/ci.yml` in the cutover commit
(gogenfilter W10 pattern) → only then template the rest of the fleet.

**Registry note:** Dependabot is GitHub-only (dead on Forgejo; gogenfilter
AGENTS.md line 25 documents the same) — the `@pin` bumps in ci.yml
(docs-gates v0.3.1, actions SHAs) become hand-swept or Renovate-swept;
TODO #161 already tracks the pin-rot tripwire.

**Acceptance test:** `.forgejo/workflows/ci.yml` green on a real Forgejo
push, `.github/workflows/ci.yml` deleted, GitHub mirror still receives
releases through the §2 mirror (releases/tag-pushes ride the same sync).
