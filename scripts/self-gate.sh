#!/usr/bin/env bash
# self-gate: run CI's `test` job locally as ONE command, before pushing.
#
# Mirrors .github/workflows/ci.yml step for step: build → vet → test (-race)
# → gofmt → self-check → facts freshness → self-render → inventory-doc
# freshness → dprint parity → SARIF emits → docs-gates battery. The
# docs-gates pin is read FROM ci.yml at runtime, so this script can never
# drift from what CI actually enforces — bump the pin in ci.yml and the
# local gate follows.
#
# Usage: scripts/self-gate.sh
# Exit codes: 0 every stage green; 1 at the first failing stage with WHAT
# failed, WHY it matters, and the exact fix. SARIF artifacts land in a temp
# dir and are discarded on exit (CI uploads them; locally they prove the
# emission path only).
#
# Note: the facts-freshness stage runs `init --force`, which rewrites ONLY
# facts.cue (claims/baseline/schema are never touched). If real drift is
# found, the refreshed facts.cue stays in the working tree for you to
# review and commit — the script prints the diff before failing.
set -euo pipefail

cd "$(dirname "$0")/.."
ARTIFACTS="$(mktemp -d /tmp/io-surface-selfgate.XXXXXX)"
trap 'rm -rf "$ARTIFACTS"' EXIT

step() { printf '\n==> %s\n' "$*"; }

fail() {
	# $1 what failed, $2 why it matters, $3 the exact fix
	printf '\nFAIL: %s\n  why: %s\n  fix: %s\n' "$1" "$2" "$3" >&2
	exit 1
}

step "build"
go build ./... ||
	fail "go build ./..." "CI's first step; nothing else can run on a broken build." \
		"fix the compile error printed above"

step "vet"
go vet ./... ||
	fail "go vet ./..." "CI runs vet right after build; any finding reddens CI." \
		"fix the vet finding printed above"

step "test (-race)"
go test -race ./... ||
	fail "go test -race ./..." "CI tests with -race; a data race ships to CI otherwise." \
		"go test -race ./<pkg> on the failing package, fix the race"

step "gofmt (whole tree)"
UNFORMATTED="$(gofmt -l .)"
if [[ -n "$UNFORMATTED" ]]; then
	printf '%s\n' "$UNFORMATTED" >&2
	fail "gofmt -l . lists unformatted files (above)" \
		"CI fails on any unformatted file, whole tree including testdata." \
		"regular code: gofmt -w <files>. testdata/ is formatter-frozen: restore from git blobs instead (git cat-file blob <hash> > <path>)"
fi

step "self-check (the tool gates its own repo)"
go run ./cmd/io-surface check --root . ||
	fail "self-check reported findings" \
		"an I/O call was added under cmd/ or internal/ without declaring it — the gate's own contract." \
		"declare the site in claims.cue/facts.cue or add a baseline entry with a note; then re-run this script"

step "facts freshness (init --force + filtered diff)"
go run ./cmd/io-surface init --root . --force
if ! git diff --quiet \
	-I'^[[:space:]]*commit:' \
	-I'^[[:space:]]*date:' \
	-- docs/architecture/io-surface/facts.cue; then
	git diff -I'^[[:space:]]*commit:' -I'^[[:space:]]*date:' -- docs/architecture/io-surface/facts.cue >&2
	fail "facts.cue is stale (diff above)" \
		"the committed manifest no longer matches the tree — code shipped without a facts refresh." \
		"review the diff above, then commit the refreshed facts.cue (init --force touches ONLY that file)"
fi

step "self-render (render must not crash)"
go run ./cmd/io-surface render --root . >/dev/null

step "inventory doc freshness"
go run ./cmd/io-surface render --root . --doc docs/architecture/io-surface/INVENTORY.md --lint --check ||
	fail "INVENTORY.md is stale vs the manifest" \
		"the embedded inventory block drifted from the committed manifest (the block carries the lint appendix; --check must repeat --lint)." \
		"go run ./cmd/io-surface render --root . --doc docs/architecture/io-surface/INVENTORY.md --lint && commit"

step "inventory survives dprint"
if command -v dprint >/dev/null 2>&1; then
	dprint check docs/architecture/io-surface/INVENTORY.md ||
		fail "dprint check failed on INVENTORY.md" \
			"the generated block must survive a dprint pass or the first local fmt reddens CI's --check gate." \
			"dprint fmt docs/architecture/io-surface/INVENTORY.md, re-render, commit"
else
	echo "SKIP: dprint not installed locally (CI's dprint/check step still enforces this)"
fi

step "emit gate SARIF (proves the emission path)"
go run ./cmd/io-surface check --root . --format sarif >"$ARTIFACTS/io-surface.sarif"

step "emit lint SARIF (artifact-only policy, TODO #61)"
go run ./cmd/io-surface lint --root . --format sarif >"$ARTIFACTS/io-surface-lint.sarif"

step "docs pointers (AGENTS/README/CONTRIBUTING references resolve)"
scripts/check-docs-pointers.sh ||
	fail "a docs/ path referenced by the living docs is missing (list above)" \
		"AGENTS.md/README.md direct every session; a broken pointer sends readers to a dead file." \
		"fix the reference in the named living doc (or restore the missing file)"

step "markdown render audit (GitHub-facing corpus)"
scripts/check-md-render.sh ||
	fail "check-md-render found a GitHub-render breaker (fences/tables/badges)" \
		"docs/status reports are read on GitHub; a swallowed fence or dropped table hides evidence." \
		"fix the named file (balance the fences, add the table separator row, or fix the badge path)"

step "docs gates (pin read from ci.yml)"
GATES_REV="$(sed -n 's/.*docs-gates@\([0-9a-zA-Z][0-9a-zA-Z.-]*\).*/\1/p' .github/workflows/ci.yml | head -1)"
if [[ -z "$GATES_REV" ]]; then
	fail "no docs-gates@<rev> pin found in .github/workflows/ci.yml" \
		"this script mirrors CI by reading its pin; ci.yml's Docs gates step changed shape." \
		"update the sed pattern in this script to match the new ci.yml line"
fi
go run "github.com/Artmann-Technologies/docs-gates/cmd/docs-gates@${GATES_REV}" run ||
	fail "docs-gates battery red" \
		"one of the five doc gates (freshness/audit-all/annotate/routing/cadence) failed." \
		"nix run github:Artmann-Technologies/docs-gates -- run . shows the failing gate with details"

printf '\nSELF-GATE GREEN — every ci.yml test-job stage passed locally.\n'
