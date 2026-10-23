#!/bin/sh
# Removes what install.sh (or `make install`) put in place: the binary, the .desktop file, the
# GNOME shortcut named "scrawl" and the autostart entry that points at this binary.
# Usage: uninstall.sh [--purge]
#   --purge   also delete the settings (~/.config/scrawl) and the cache (~/.cache/scrawl).
#             Saved screenshots are never touched.
# Variables: PREFIX (default ~/.local), BINDIR, APPDIR: the same as for install.sh;
# DESTDIR for staged installs (then the shortcut, autostart and --purge are skipped).
# A copy installed by apt is removed with `sudo apt remove scrawl` (see the README).
set -eu

here=$(cd "$(dirname "$0")" && pwd)
PREFIX=${PREFIX:-$HOME/.local}
BINDIR=${BINDIR:-$PREFIX/bin}
APPDIR=${APPDIR:-$PREFIX/share/applications}
DESTDIR=${DESTDIR:-}
desktop=io.github.jexme.Scrawl.desktop

purge=0
for arg in "$@"; do
	case "$arg" in
	--purge) purge=1 ;;
	-h | --help)
		sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		echo "uninstall.sh: unknown option $arg (only --purge)" >&2
		exit 2
		;;
	esac
done

removed=0
remove() {
	if [ -e "$1" ] || [ -L "$1" ]; then
		rm -f "$1"
		echo "Removed $1"
		removed=1
	fi
}

remove "$DESTDIR$BINDIR/scrawl"
remove "$DESTDIR$APPDIR/$desktop"

if [ -z "$DESTDIR" ]; then
	# The autostart entry written by Settings, if it starts this very binary.
	autostart=${XDG_CONFIG_HOME:-$HOME/.config}/autostart/$desktop
	if [ -f "$autostart" ] && grep -qF "$BINDIR/scrawl" "$autostart"; then
		remove "$autostart"
	fi

	# The hotkey. Another copy of scrawl (for example from apt) may still need it.
	hk=""
	for f in "$here/gnome-hotkey.sh" "$here/../scripts/gnome-hotkey.sh"; do
		[ -f "$f" ] && { hk=$f; break; }
	done
	if [ -n "$hk" ]; then
		if command -v scrawl >/dev/null 2>&1; then
			other=$(command -v scrawl)
			echo "Another scrawl is still installed ($other): the GNOME shortcut is kept."
			echo "If it still points at the removed copy, repoint it: sh $hk Print '$other'"
		else
			code=0
			sh "$hk" --remove --auto || code=$?
			case $code in
			0) echo "Removed the GNOME shortcut \"scrawl\"" ;;
			13) echo "Could not remove the GNOME shortcut: delete it in Settings > Keyboard > Custom Shortcuts." ;;
			esac
		fi
	fi

	if [ "$purge" = 1 ]; then
		for d in "${XDG_CONFIG_HOME:-$HOME/.config}/scrawl" "${XDG_CACHE_HOME:-$HOME/.cache}/scrawl"; do
			if [ -d "$d" ]; then
				rm -rf "$d"
				echo "Removed $d"
				removed=1
			fi
		done
	fi
fi

if [ "$removed" = 0 ]; then
	echo "Nothing to remove in $BINDIR and $APPDIR."
else
	echo "Scrawl is uninstalled."
fi
[ "$purge" = 1 ] || [ -n "$DESTDIR" ] || echo "Settings (~/.config/scrawl) and cache (~/.cache/scrawl) were kept; add --purge to delete them."
