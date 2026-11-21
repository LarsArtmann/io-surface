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
  [libX11.so.6]=xorg.libX11
  [libxcb.so.1]=xorg.libxcb
  [libXcomposite.so.1]=xorg.libXcomposite
  [libXdamage.so.1]=xorg.libXdamage
  [libXext.so.6]=xorg.libXext
  [libXfixes.so.3]=xorg.libXfixes
  [libxkbcommon.so.0]=libxkbcommon
  [libXrandr.so.2]=xorg.libXrandr
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
  store_path=$(nix eval "nixpkgs#$attr.out.outPath" --raw)
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
