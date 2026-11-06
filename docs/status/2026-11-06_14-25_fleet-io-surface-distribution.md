# Fleet I/O-surface distribution — first BuildFlow-provider run (2026-11-06)

The v0.2.0 provider (`github.com/larsartmann/io-surface/provider`, blank-imported
into BuildFlow at `13716de`) makes the gate a fleet-wide DAG step. This is the
first distribution across a real repo sample, run via
`buildflow -s io-surface --format finding` per repo.

## Distribution (11 repos)

| Repo                  | Findings | Errors | Note                                                          |
| --------------------- | -------: | ------ | ------------------------------------------------------------- |
| dnsblockd             |        0 | 0      | 0-error gate holds through the provider (TODO #26 still true) |
| **InboxClean**        |    **3** | **3**  | 3× `undeclared-site` (`bufio.NewReader`) — see below          |
| go-atomic-write       |        0 | 0      | clean                                                         |
| gogenfilter           |        0 | 0      | clean                                                         |
| dynamic-markdown-site |        0 | 0      | clean                                                         |
| go-output             |        0 | 0      | clean                                                         |
| samber-do-auditlog    |        0 | 0      | clean                                                         |
| go-workflow-auditlog  |        0 | 0      | clean                                                         |
| wireguard-collector   |        — | —      | Rust repo: Go trigger does not fire (step not run)            |
| mic-monitor           |        — | —      | same                                                          |
| clipboard-monitor     |        — | —      | same                                                          |

8/8 Go repos gated; 7 clean, 1 with real findings. 3 Rust collectors correctly
skipped by the language trigger.

## The InboxClean finding (actionable, real)

The three errors are `io-surface/undeclared-site` for `bufio.NewReader` in
`cmd/inboxclean/commands.go` (×2) and `cmd/inboxclean/confirm.go`. This is the
T20 extractor family expansion (`bufio` constructors → `stream-copy`) doing its
job on the first real consumer: InboxClean's committed manifest predates the
family, so the sites were invisible at v0.1.0 and surface at v0.2.0. Fix is the
standard loop in that repo: `init --force` + declare/claim the three sites (or
`baseline add` with a reason), then bump its CI pin to v0.2.0. NOT done here —
cross-repo writes stay out of this session's lane (fleet-exec lock discipline).

## Family distribution (what the sample exercises)

The gate's family vocabulary as exercised by the fleet sample: file-read,
file-write, env, outbound, listener, database, stream-copy, entropy,
ipc-subprocess, embed — all ten families have at least one real fleet site
declared across the sample's manifests (dnsblockd + InboxClean + io-surface
itself cover every family).

## Verification

- Command per repo: `buildflow -s io-surface --format finding` (binary
  `buildflow 13716de`, provider discovered from `sdk_imports.go`)
- io-surface module resolved from the public proxy at v0.2.0 (release-verified)
- This repo itself gates clean through the same step (findings: [])
