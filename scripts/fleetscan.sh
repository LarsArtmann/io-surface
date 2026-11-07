#!/usr/bin/env bash
# fleetscan: run `io-surface lint` across every pma-discovered Go project and
# aggregate the detector findings per rule (plan T16). Output: a JSON report
# (default /tmp/fleetscan-report.json) plus a stdout summary. Read-only on the
# scanned repos; the aggregate feeds the evidence-gated severity decisions
# (T17) — numbers without this command are unverified per repo convention.
#
# Usage: scripts/fleetscan.sh [OUT.json] [REPOS.tsv]
#   OUT.json   aggregate destination (default /tmp/fleetscan-report.json)
#   REPOS.tsv  optional name<TAB>path list (default: pma list -o json)
set -euo pipefail

OUT="${1:-/tmp/fleetscan-report.json}"
IOSURFACE_BIN="$(mktemp -u /tmp/io-surface-lint.XXXXXX)"
trap 'rm -f "$IOSURFACE_BIN"' EXIT

cd "$(dirname "$0")/.."
go build -o "$IOSURFACE_BIN" ./cmd/io-surface

REPOS_TSV="${2:-}"
if [[ -z "$REPOS_TSV" ]]; then
  REPOS_TSV="$(mktemp /tmp/fleetscan-repos.XXXXXX.tsv)"
  trap 'rm -f "$REPOS_TSV" "$IOSURFACE_BIN"' EXIT
  pma list -o json 2>/dev/null | sed -n '/^\[/,$p' |
    python3 -c '
import json, sys
for p in json.load(sys.stdin):
    if "go" in p.get("Types", []) and not p.get("Missing"):
        print(p["Name"] + "\t" + p["Path"])' > "$REPOS_TSV"
fi

python3 - "$REPOS_TSV" "$IOSURFACE_BIN" "$OUT" << 'PY'
import json, subprocess, sys

repos_tsv, bin_path, out_path = sys.argv[1], sys.argv[2], sys.argv[3]
repos = []
with open(repos_tsv) as fh:
    for line in fh:
        line = line.rstrip("\n")
        if not line:
            continue
        name, path = line.split("\t", 1)
        repos.append((name, path))

per_rule = {}
per_repo = []
scan_failures = []
for name, path in repos:
    try:
        proc = subprocess.run(
            [bin_path, "lint", "--root", path, "--format", "json"],
            capture_output=True, text=True, timeout=300,
        )
    except subprocess.TimeoutExpired:
        scan_failures.append({"project": name, "path": path, "error": "timeout"})
        continue
    # lint exits 0 by design (report-only); anything else is a scan failure.
    if proc.returncode != 0:
        scan_failures.append({"project": name, "path": path, "error": proc.stderr[-200:]})
        continue
    try:
        rep = json.loads(proc.stdout)
    except json.JSONDecodeError:
        scan_failures.append({"project": name, "path": path, "error": "json decode"})
        continue
    counts = {}
    for f in rep.get("findings", []):
        counts[f["rule"]] = counts.get(f["rule"], 0) + 1
    for rule, n in counts.items():
        per_rule[rule] = per_rule.get(rule, 0) + n
    per_repo.append({
        "project": name, "path": path,
        "sites": rep["stats"]["extracted"], "resolved": rep["stats"]["resolved"],
        "findings": counts,
    })

report = {
    "generatedBy": "scripts/fleetscan.sh (io-surface lint face)",
    "discovered": len(repos),
    "scanned": len(per_repo),
    "scanFailures": scan_failures,
    "reposWithFindings": sum(1 for r in per_repo if r["findings"]),
    "findingsByRule": dict(sorted(per_rule.items(), key=lambda kv: -kv[1])),
    "repos": per_repo,
}
with open(out_path, "w") as fh:
    json.dump(report, fh, indent=1)

print(f"fleetscan: {len(per_repo)}/{len(repos)} repos scanned, "
      f"{report['reposWithFindings']} with findings")
for rule, n in report["findingsByRule"].items():
    print(f"  {n:6d}  {rule}")
if scan_failures:
    print(f"  scan failures: {len(scan_failures)}")
PY
