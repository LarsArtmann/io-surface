# I/O Surface Inventory

> GENERATED from the io-surface manifest by `io-surface render`. Do not edit by hand:
> edit `facts.cue` (regenerate) or `claims.cue` (judgment) and re-render.
>
> Audited 2026-11-06 at `abc1234` — 4 sites, 3 location claims.

## Taxonomy at a glance

| Family       | Sites | Claimed | With rationale |
| ------------ | ----: | ------: | -------------: |
| `file-read`  |     1 |       1 |              1 |
| `file-write` |     1 |       1 |              0 |
| `database`   |     1 |       1 |              0 |
| `outbound`   |     1 |       0 |              0 |

## `file-read` (1)

| Site                        | Location                    | Resource            | Trigger       | Discipline           |
| --------------------------- | --------------------------- | ------------------- | ------------- | -------------------- |
| `internal/store.LoadConfig` | `internal/store/store.go:9` | service config file | process start | read-only, no-secret |

### Judgments

- **`internal/store.LoadConfig`** — config is read once at boot; paths come from a validated flag

## `file-write` (1)

| Site                  | Location                     | Resource   | Trigger               | Discipline |
| --------------------- | ---------------------------- | ---------- | --------------------- | ---------- |
| `internal/store.Save` | `internal/store/store.go:31` | state file | explicit user command | —          |

### Judgments

- **`internal/store.Save`** — exception: atomic write handled by wrapper

## `database` (1)

| Site             | Location             | Resource          | Trigger     | Discipline |
| ---------------- | -------------------- | ----------------- | ----------- | ---------- |
| `cmd/app.openDB` | `cmd/app/main.go:24` | local sqlite file | command run | local-only |

## `outbound` (1)

| Site            | Location             | Resource | Trigger | Discipline |
| --------------- | -------------------- | -------- | ------- | ---------- |
| `cmd/app.fetch` | `cmd/app/main.go:18` | —        | —       | —          |

## Reachability (capabilities)

> Engine-observed by capslock v0.3.4 of `abc1234` (2026-11-06): what each package can REACH, beyond its declared sites.
> `direct` = through the package's own code (high signal); `deep` adds transitive
> dependency paths (stdlib linkage reaches everywhere and carries no per-package signal).
> Advisory: a package whose direct capability has no covered family is a `shadow-capability` info.

| Package          | Direct    | Deep (transitive) | Judgment                                                    |
| ---------------- | --------- | ----------------- | ----------------------------------------------------------- |
| `cmd/app`        | `NETWORK` | `NETWORK`         | —                                                           |
| `internal/ghost` | —         | `EXEC`            | —                                                           |
| `internal/store` | `FILES`   | `EXEC`, `FILES`   | direct EXEC here is the test helper's git call, judged fine |

## Deliberately out of scope

- `testdata/` — fixture I/O never ships
- `vendor/` — third-party code, not ours to declare

