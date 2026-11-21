#!/usr/bin/env bash
# Mobile-render audit: screenshot a report page at 320 px with
# chrome-headless-shell (the hyperframes-provisioned binary).
#
# Chrome's shared libraries are provisioned on demand: every soname the
# dynamic loader cannot find is mapped to a nixpkgs attribute (SONAME_PKG
# below) and resolved with `nix eval`, so the recipe survives reboots and
# store-path churn — no hand-maintained LD_LIBRARY_PATH (the /tmp recipe
# used on 2026-11-15 evaporated with /tmp).
#
# Usage: scripts/320px-shot.sh <page.html> [output.png]
# Defaults output to docs/evidence/320px-audit/<page>-320.png.
# The horizontal scrollbar visible in the PNG is the overflow detector:
# scroll arrows at the bottom edge mean content exceeds 320 px.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHROME="${CHROME_BIN:-$HOME/.cache/hyperframes/chrome/chrome-headless-shell/linux-152.0.7977.30/chrome-headless-shell-linux64/chrome-headless-shell}"
WIDTH=320
HEIGHT=2400

[ -x "$CHROME" ] || {
  echo "320px-shot: chrome binary not found at $CHROME (override with CHROME_BIN)" >&2
  exit 1
}
[ "${1:-}" ] || {
  echo "usage: $0 <page.html> [output.png]" >&2
  exit 2
}
IN=$(readlink -f "$1")
OUT=${2:-"$REPO_ROOT/docs/evidence/320px-audit/$(basename "${IN%.*}")-320.png"}

missing_sonames() {
  ldd "$CHROME" 2>&1 | awk '/not found/ {print $1}' | sort -u
}

declare -A SONAME_PKG=(
  [libasound.so.2]=alsa-lib
  [libatk-1.0.so.0]=at-spi2-core
  [libatk-bridge-2.0.so.0]=at-spi2-core
  [libatspi.so.0]=at-spi2-core
  [libdbus-1.so.3]=dbus.lib
  [libexpat.so.1]=expat
  [libgbm.so.1]=libgbm
  [libgio-2.0.so.0]=glib
  [libglib-2.0.so.0]=glib
  [libgobject-2.0.so.0]=glib
  [libnspr4.so]=nspr
  [libnss3.so]=nss
  [libnssutil3.so]=nss
  [libudev.so.1]=udev
  [libX11.so.6]=libx11
  [libxcb.so.1]=libxcb
  [libXcomposite.so.1]=libxcomposite
  [libXdamage.so.1]=libxdamage
  [libXext.so.6]=libxext
  [libXfixes.so.3]=libxfixes
  [libxkbcommon.so.0]=libxkbcommon
  [libXrandr.so.2]=libxrandr
)

ldpath=""
declare -A resolved=()
for so in $(missing_sonames); do
  attr=${SONAME_PKG[$so]:-}
  if [ -z "$attr" ]; then
    echo "320px-shot: unmapped soname '$so' — extend SONAME_PKG in $0" >&2
    exit 1
  fi
  [ -n "${resolved[$attr]:-}" ] && continue
  resolved[$attr]=1

  # Multi-output derivations resolve differently: `.out.outPath` names the
  # `out` output (glib's libs), while output attrs like `dbus.lib` need
  # `.outPath` (their `.out` would silently point back at the out output).
  # Verify the soname actually exists in whichever store path we pick.
  store_path=""
  for form in "$attr.out.outPath" "$attr.outPath"; do
    if candidate=$(nix eval "nixpkgs#$form" --raw 2>/dev/null) && [ -e "$candidate/lib/$so" ]; then
      store_path=$candidate
      break
    fi
  done
  if [ -z "$store_path" ]; then
    echo "320px-shot: '$so' not found under nixpkgs#$attr (tried .out.outPath and .outPath)" >&2
    exit 1
  fi
  ldpath="$ldpath$store_path/lib:"
done
export LD_LIBRARY_PATH="${ldpath}${LD_LIBRARY_PATH:-}"

still_missing=$(missing_sonames)
if [ -n "$still_missing" ]; then
  echo "320px-shot: libraries still missing after provisioning: $still_missing" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"
"$CHROME" --headless --disable-gpu --no-sandbox \
  --screenshot="$OUT" --window-size=$WIDTH,$HEIGHT \
  "file://$IN" >/dev/null 2>&1

echo "320px-shot: $OUT"
