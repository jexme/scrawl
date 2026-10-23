#!/bin/sh
# Builds a .deb for Debian and Ubuntu from a ready bin/scrawl.
# Usage: packaging/deb.sh VERSION ARCH OUTDIR   (VERSION as in the tag: v0.1.0-rc12)
set -eu

version=${1:?version, for example v0.1.0-rc12}
arch=${2:?architecture, for example amd64}
out=${3:?directory for the .deb}
root=$(cd "$(dirname "$0")/.." && pwd)
desktop=io.github.jexme.Scrawl.desktop

# Debian version: v0.1.0-rc12 -> 0.1.0~rc12 (a pre-release sorts below the final),
# v0.1.0-rc12-1-gabc-dirty -> 0.1.0~rc12+1.gabc.dirty.
deb_version=$(printf '%s' "$version" | sed -e 's/^v//' -e 's/-/~/' -e 's/-/+/' -e 's/-/./g')

pkg=$(mktemp -d)
trap 'rm -rf "$pkg"' EXIT

install -Dm755 "$root/bin/scrawl" "$pkg/usr/bin/scrawl"
install -d "$pkg/usr/share/applications"
# Absolute Exec: the portal matches an application to a .desktop file only if Exec exists.
sed 's|^Exec=scrawl|Exec=/usr/bin/scrawl|' "$root/packaging/$desktop" > "$pkg/usr/share/applications/$desktop"
chmod 644 "$pkg/usr/share/applications/$desktop"

doc="$pkg/usr/share/doc/scrawl"
install -d "$doc"
install -m644 "$root/README.md" "$root/THIRD_PARTY_NOTICES.md" "$doc/"
{
	echo "Scrawl is released under the MIT license."
	echo "Third-party components and their licenses: THIRD_PARTY_NOTICES.md in this directory."
	echo
	cat "$root/LICENSE"
} > "$doc/copyright"
chmod 644 "$doc/copyright"

size=$(du -sk "$pkg/usr" | cut -f1)
install -d "$pkg/DEBIAN"
cat > "$pkg/DEBIAN/control" <<CONTROL
Package: scrawl
Version: $deb_version
Architecture: $arch
Maintainer: jexme <22722008+jexme@users.noreply.github.com>
Section: graphics
Priority: optional
Installed-Size: $size
Homepage: https://github.com/jexme/scrawl
Depends: libc6 (>= 2.34), libpipewire-0.3-0t64 | libpipewire-0.3-0, libwayland-client0, libwayland-egl1, libwayland-cursor0, libxkbcommon0, libxkbcommon-x11-0, libegl1, libx11-6, libx11-xcb1, libxcursor1, libxfixes3
Recommends: wl-clipboard, xclip, libnotify-bin
Description: Fast screenshots with annotations for GNOME (Wayland and X11)
 Select an area, draw arrows, boxes, text and blur, copy to the clipboard
 or save to a file. Before/after collages, 9 interface languages.
 .
 On Wayland the clipboard needs wl-clipboard, on X11 xclip; notifications
 need libnotify-bin. After installing, run "scrawl -settings" to pick the
 monitors, grant the permission and assign a hotkey.
CONTROL

# Package directories must be 755 regardless of the builder's umask.
find "$pkg" -type d -exec chmod 755 {} +

mkdir -p "$out"
file="$out/scrawl_${deb_version}_${arch}.deb"
dpkg-deb --root-owner-group -Zxz --build "$pkg" "$file" >/dev/null
echo "$file"
