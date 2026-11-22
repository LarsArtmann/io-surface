#!/usr/bin/env bash
# fp-sample: seed a promotion-policy §7 verification sample for one detector
# rule out of a fleetscan aggregate. The fleetscan JSON carries per-repo
# counts only, so this re-runs `io-surface lint --format json` on the repos
# that carry the rule until the sample floor is met (policy W4: N ≥ 30),
# then writes a worksheet for hand adjudication (policy §2: verified vs FP
# per site). The worksheet lands in docs/evidence/ — never /tmp — and the
# adjudicated numbers go into the §9 ledger by the session that runs it.
#
# Usage: scripts/fp-sample.sh RULE FLEETSCAN.json [N] [OUT.json]
#   RULE           detector rule (io-surface/write-non-atomic, ...)
#   FLEETSCAN.json aggregate, e.g. docs/evidence/fleetscan-2026-11-15.json
#   N              sample floor (default 30)
#   OUT.json       worksheet destination
#                  (default docs/evidence/fp-sample-<rule>-<date>.json)
set -euo pipefail

RULE="${1:?usage: fp-sample.sh RULE FLEETSCAN.json [N] [OUT.json]}"
FLEET="${2:?usage: fp-sample.sh RULE FLEETSCAN.json [N] [OUT.json]}"
N="${3:-30}"
DATE="$(date +%Y-%m-%d)"
OUT="${4:-docs/evidence/fp-sample-${RULE#io-surface/}-${DATE}.json}"

IOSURFACE_BIN="$(mktemp -u /tmp/io-surface-fpsample.XXXXXX)"
trap 'rm -f "$IOSURFACE_BIN"' EXIT

cd "$(dirname "$0")/.."
mkdir -p docs/evidence
go build -o "$IOSURFACE_BIN" ./cmd/io-surface

python3 - "$RULE" "$FLEET" "$N" "$OUT" "$IOSURFACE_BIN" "$DATE" <<'PY'
import json, os, subprocess, sys
from datetime import datetime, timezone

rule, fleet_path, floor, out_path, bin_path, today = sys.argv[1:7]
floor = int(floor)

with open(fleet_path) as fh:
    fleet = json.load(fh)

candidates = [
    r for r in fleet.get("repos", [])
    if r.get("findings", {}).get(rule, 0) > 0
]
if not candidates:
    sys.exit(f"fp-sample: no repo in {fleet_path} carries {rule}")

entries = []
scanned = 0
for repo in candidates:
    if len(entries) >= floor:
        break
    path = repo["path"]
    dirs = repo.get("dirs", "default")
    args = [bin_path, "lint", "--root", path, "--format", "json"]
    if dirs != "default":
        args += ["--dirs", dirs]
    proc = subprocess.run(args, capture_output=True, text=True, timeout=300)
    if proc.returncode != 0:
        print(f"fp-sample: skip {repo['project']}: lint exit {proc.returncode}",
              file=sys.stderr)
        continue
    rep = json.loads(proc.stdout)
    scanned += 1
    for f in rep.get("findings") or []:
        if f.get("rule") != rule:
            continue
        entries.append({
            "project": repo["project"],
            "location": f["location"],
            "id": f.get("id", ""),
            "message": f.get("message", ""),
            "adjudication": "UNADJUDICATED",
            "note": "",
        })
        if len(entries) >= floor:
            break

worksheet = {
    "generatedBy": "scripts/fp-sample.sh (promotion-policy §7 sampler)",
    "rule": rule,
    "fleetSource": os.path.relpath(fleet_path),
    "sampledAt": datetime.now(timezone.utc).isoformat(timespec="seconds"),
    "floor": floor,
    "sampled": len(entries),
    "reposScanned": scanned,
    "entries": entries,
}
with open(out_path, "w") as fh:
    json.dump(worksheet, fh, indent=1)

print(f"fp-sample: {rule} — {len(entries)}/{floor} sites sampled "
      f"from {scanned} repos -> {out_path}")
print("Adjudicate every entry (verified = rule is right that the flagged")
print("code exhibits the bug class; FP = correct as written), then set the")
print("per-entry adjudication and update the §9 ledger in")
print("docs/planning/2026-11-07_promotion-policy.md.")
PY
