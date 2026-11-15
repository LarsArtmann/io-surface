#!/usr/bin/env bash
# check-md-render: structural render audit for the docs/status report corpus
# (GitHub-facing markdown). Three GitHub-render breakers, checked mechanically:
#   1. unbalanced code fences (a stray ``` swallows the rest of the file as code)
#   2. table blocks without a separator row (GitHub drops the whole table)
#   3. image/badge references whose local path does not exist (remote http(s)
#      URLs are recorded, not fetched)
# Usage: scripts/check-md-render.sh [DIR] (default: docs/status)
set -euo pipefail

DIR="${1:-docs/status}"
fail=0
files=$(find "$DIR" -name '*.md' | sort)
count=0
for f in $files; do
	count=$((count + 1))
	# 1. fence balance: count fence-start lines; odd means a file swallowed whole
	fences=$(grep -c '^```' "$f" || true)
	if ((fences % 2 != 0)); then
		echo "FENCE-UNBALANCED: $f ($fences fence lines)"
		fail=1
	fi
	# 2. tables: a run of pipe lines must contain a separator row (---)
	table_rc=0
	python3 - "$f" << 'PYEOF' || table_rc=$?
import re, sys
path = sys.argv[1]
lines = open(path).read().split('\n')
i = 0
while i < len(lines):
    if lines[i].lstrip().startswith('|') and i + 1 < len(lines) and lines[i+1].lstrip().startswith('|'):
        j = i
        while j < len(lines) and lines[j].lstrip().startswith('|'):
            j += 1
        block = lines[i:j]
        if not any(re.match(r'^\s*\|[\s:|-]+\|\s*$', b) for b in block):
            print(f'TABLE-NO-SEPARATOR: {path} lines {i+1}-{j}')
            sys.exit(1)
        i = j
    i += 1
PYEOF
	if ((table_rc != 0)); then fail=1; fi
	# 3. badges/images: markdown image refs must resolve when local
	while IFS=: read -r ln ref; do
		path_part=$(printf '%s' "$ref" | sed -E 's/^!\[[^]]*\]\(([^)]+)\).*/\1/')
		case "$path_part" in
			http://*|https://*|#*) : ;;  # remote or anchor: recorded, not fetched
			*)
				if [[ ! -e "$(dirname "$f")/$path_part" && ! -e "$path_part" ]]; then
					echo "BADGE-UNRESOLVED: $f:$ln $path_part"
					fail=1
				fi
				;;
		esac
	done < <(grep -nE '!\[[^]]*\]\([^)]+\)' "$f" || true)
done
echo "check-md-render: $count file(s) audited under $DIR"
exit $fail
