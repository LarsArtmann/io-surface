#!/usr/bin/env bash
# Mobile-render audit, measurement half: report per-element horizontal
# overflow of a page rendered at 320 px — the numeric counterpart to
# 320px-shot.sh (whose PNG scrollbar arrows are the visual detector).
#
# CDP-less probe: a wrapper page iframes the target at 320 px, walks
# contentDocument for scrollWidth > 320, and writes the results into its
# own DOM; `chrome --dump-dom` prints that DOM, and the script parses the
# numbers out. No DevTools protocol, no /tmp recipe (the 2026-11-21 audit's
# harness evaporated with /tmp — this file is its committed recovery).
#
# Usage: scripts/320px-probe.sh <page.html> [threshold]
# Exit 0 when nothing exceeds the threshold at document level, 3 when it
# does (advisory: nothing in CI consumes this yet; the exit code is for
# quick shell loops).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHROME="${CHROME_BIN:-$HOME/.cache/hyperframes/chrome/chrome-headless-shell/linux-152.0.7977.30/chrome-headless-shell-linux64/chrome-headless-shell}"
WIDTH=320
HEIGHT=2400
THRESHOLD="${2:-320}"

[ -x "$CHROME" ] || {
  echo "320px-probe: chrome binary not found at $CHROME (override with CHROME_BIN)" >&2
  exit 1
}
[ "${1:-}" ] || {
  echo "usage: $0 <page.html> [threshold]" >&2
  exit 2
}
IN=$(readlink -f "$1")

# The library provisioning block mirrors 320px-shot.sh verbatim (kept
# standalone on purpose: each script must run alone; extend SONAME_PKG in
# BOTH when a new soname appears).
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
    echo "320px-probe: unmapped soname '$so' — extend SONAME_PKG in $0" >&2
    exit 1
  fi
  [ -n "${resolved[$attr]:-}" ] && continue
  resolved[$attr]=1
  store_path=""
  for form in "$attr.out.outPath" "$attr.outPath"; do
    if candidate=$(nix eval "nixpkgs#$form" --raw 2>/dev/null) && [ -e "$candidate/lib/$so" ]; then
      store_path=$candidate
      break
    fi
  done
  if [ -z "$store_path" ]; then
    echo "320px-probe: '$so' not found under nixpkgs#$attr" >&2
    exit 1
  fi
  ldpath="$ldpath$store_path/lib:"
done
export LD_LIBRARY_PATH="${ldpath}${LD_LIBRARY_PATH:-}"

WRAPPER=$(mktemp -d /tmp/320px-probe-wrapper.XXXXXX)
trap 'rm -rf "$WRAPPER"' EXIT
cat > "$WRAPPER/probe.html" <<HTML
<!doctype html>
<meta charset="utf-8">
<body style="margin:0">
<iframe id="f" src="file://$IN" style="width:${WIDTH}px;height:${HEIGHT}px;border:0"></iframe>
<pre id="out">PENDING</pre>
<script>
const f = document.getElementById('f');
f.onload = () => {
  setTimeout(() => {
    const doc = f.contentDocument;
    if (!doc) { document.getElementById('out').textContent = 'ERROR no contentDocument'; return; }
    const rows = [];
    rows.push('document.scrollWidth=' + doc.scrollingElement.scrollWidth);
    let overflowing = 0;
    for (const el of doc.querySelectorAll('*')) {
      if (el.scrollWidth > $THRESHOLD) {
        overflowing++;
        if (overflowing <= 25) {
          rows.push(el.tagName + ' class=' + (el.className || '-') + ' scrollWidth=' + el.scrollWidth);
        }
      }
    }
    rows.push('elements exceeding ${THRESHOLD}px: ' + overflowing);
    document.getElementById('out').textContent = rows.join('\\n');
  }, 700);
};
</script>
HTML

"$CHROME" --headless --disable-gpu --no-sandbox \
  --window-size=$WIDTH,$HEIGHT \
  --dump-dom "file://$WRAPPER/probe.html" 2>/dev/null > "$WRAPPER/dom.html"

python3 - "$WRAPPER/dom.html" "$WRAPPER/result.txt" << 'PY'
import html, re, sys
dom = open(sys.argv[1], encoding="utf-8", errors="replace").read()
m = re.search(r'<pre id="out">(.*?)</pre>', dom, re.S)
if not m or "PENDING" in m.group(1):
    sys.exit(1)
open(sys.argv[2], "w").write(html.unescape(m.group(1)).strip())
PY

if ! grep -q "document.scrollWidth=" "$WRAPPER/result.txt"; then
  echo "320px-probe: probe produced no measurements (iframe load failed?)" >&2
  exit 1
fi

cat "$WRAPPER/result.txt"
doc=$(grep -o "document.scrollWidth=[0-9]*" "$WRAPPER/result.txt" | head -1 | cut -d= -f2)
if [ "${doc:-999999}" -gt "$THRESHOLD" ]; then
  echo "320px-probe: OVERFLOW — document scrollWidth $doc > $THRESHOLD" >&2
  exit 3
fi
echo "320px-probe: OK — document scrollWidth $doc <= $THRESHOLD"
