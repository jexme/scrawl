#!/bin/sh
# Prepares third_party/gio: the Gio sources of the version in go.mod, plus our patch
# third_party/gio.patch (app.FullscreenAt: fullscreen on the right Wayland monitor).
# A repeated run does nothing while the version and the patch are unchanged; -f recreates.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
dst="$root/third_party/gio"
patchfile="$root/third_party/gio.patch"
marker="$dst/.scrawl-patched"

ver=$(grep -E '^[[:space:]]*gioui\.org v' "$root/go.mod" | awk '{print $2}' | head -n 1)
[ -n "$ver" ] || { echo "could not find the gioui.org version in go.mod" >&2; exit 1; }
want="$ver $(sha256sum "$patchfile" | cut -d' ' -f1)"

if [ -f "$marker" ] && [ "$(cat "$marker")" = "$want" ] && [ "${1:-}" != "-f" ]; then
	echo "Gio $ver is already prepared"
	exit 0
fi

# We download from a temporary directory: in the project root go.mod points to
# third_party/gio, which may not exist yet.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
(cd "$tmp" && go mod download "gioui.org@$ver") || true
src=$(cd "$tmp" && go list -m -f '{{.Dir}}' "gioui.org@$ver" 2>/dev/null | head -n 1)
# Under Git Bash on Windows Go returns a path like C:\Users\...: cp and patch expect the
# Unix form.
if [ -n "$src" ] && command -v cygpath >/dev/null 2>&1; then src=$(cygpath -u "$src"); fi
[ -d "$src" ] || { echo "could not download gioui.org@$ver" >&2; exit 1; }

rm -rf "$dst"
cp -R "$src" "$dst"
chmod -R u+w "$dst"
patch -p1 -s -d "$dst" < "$patchfile"
echo "$want" > "$marker"
echo "Gio $ver prepared with the patch"
