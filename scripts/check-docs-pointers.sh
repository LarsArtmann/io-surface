#!/usr/bin/env bash
# check-docs-pointers: every docs/ path named in the living docs (AGENTS.md,
# README.md, CONTRIBUTING.md) must exist on disk. A moved, renamed or deleted
# doc referenced by the living docs is a broken pointer the reader only
# discovers mid-task — AGENTS.md and README direct every session, so their
# pointers are load-bearing.
#
# Deliberately literal: glob references (docs/.../2026-11-07_08-08_*.md) and
# bare directory references (docs/review/libs/) are skipped — the check owns
# the exact-path .md/.cue/.json pointer class, which is the one that rots.
#
# Usage: scripts/check-docs-pointers.sh   (exit 1 lists every missing target)
set -euo pipefail

cd "$(dirname "$0")/.."

fail=0
for src in AGENTS.md README.md CONTRIBUTING.md; do
	# Pointers live in prose; fenced code blocks carry consumer-repo EXAMPLE
	# paths (docs/architecture/IO.md in the doc-embed walkthrough) that must
	# not be checked against THIS repo. Strip the fences before grepping.
	while read -r ref; do
		if [[ ! -e "$ref" ]]; then
			printf 'MISSING: %s (referenced in %s)\n' "$ref" "$src" >&2
			fail=1
		fi
	done < <(awk '/^```/{fence=!fence; next} !fence' "$src" |
		grep -oE '\bdocs/[A-Za-z0-9_./-]+\.(md|cue|json)' | sort -u)
done

if [[ "$fail" -eq 0 ]]; then
	echo "docs pointers: all referenced docs/ paths exist"
fi
exit "$fail"
