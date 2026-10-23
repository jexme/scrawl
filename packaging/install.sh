#!/bin/sh
# Installs Scrawl from an unpacked release archive: the binary and the .desktop file.
# Variables: PREFIX (default ~/.local), BINDIR, APPDIR.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
PREFIX=${PREFIX:-$HOME/.local}
BINDIR=${BINDIR:-$PREFIX/bin}
APPDIR=${APPDIR:-$PREFIX/share/applications}
desktop=io.github.jexme.Scrawl.desktop

install -Dm755 "$here/scrawl" "$BINDIR/scrawl"
install -d "$APPDIR"
# An absolute path is put into Exec: the portal matches the application with the .desktop
# file only if the Exec target exists.
sed "s|^Exec=scrawl|Exec=$BINDIR/scrawl|" "$here/$desktop" > "$APPDIR/$desktop"
chmod 644 "$APPDIR/$desktop"

# The Print key: on GNOME it is assigned automatically unless it is taken by someone
# else's shortcut. The exit code of the script (see scripts/gnome-hotkey.sh) selects the
# message below.
hk=11
if [ -f "$here/gnome-hotkey.sh" ]; then
	hk=0
	sh "$here/gnome-hotkey.sh" Print "$BINDIR/scrawl" --auto || hk=$?
fi
# The built-in GNOME screenshot also sits on Print and intercepts it.
gnome_print=0
if [ "$hk" = 0 ] && gsettings get org.gnome.shell.keybindings show-screenshot-ui 2>/dev/null | grep -q "Print"; then
	gnome_print=1
fi

# The message depends on the system language and the session type: X11 has no portal, so
# there is no need to choose monitors.
x11=0
[ -z "${WAYLAND_DISPLAY:-}" ] && [ -n "${DISPLAY:-}" ] && x11=1

# Programs without which the clipboard and notifications do not work: wl-copy on Wayland
# (package wl-clipboard), xclip on X11, notify-send everywhere (libnotify-bin). Package
# names for Debian and Ubuntu.
missing=""
need() { command -v "$1" >/dev/null 2>&1 || missing="$missing $2"; }
if [ "$x11" = 1 ]; then need xclip xclip; else need wl-copy wl-clipboard; fi
need notify-send libnotify-bin

# Is $BINDIR on PATH? Without it the command "scrawl" is not found.
on_path=0
case ":$PATH:" in *":$BINDIR:"*) on_path=1 ;; esac

# Shared libraries the binary needs and the system lacks (a minimal install, or another
# distribution than the archive was built on). ldd, or readelf plus the ldconfig cache if
# there is no ldd. Only the loader's own wording "not found" is parsed, so the language is
# forced to C. The names are mapped to packages below.
missing_libs=""
if command -v ldd >/dev/null 2>&1; then
	missing_libs=$(LC_ALL=C ldd "$BINDIR/scrawl" 2>/dev/null | sed -n 's/^[[:space:]]*\([^[:space:]]*\) => not found.*/\1/p')
elif command -v readelf >/dev/null 2>&1; then
	ldc=$(command -v ldconfig 2>/dev/null || true)
	[ -n "$ldc" ] || { [ -x /sbin/ldconfig ] && ldc=/sbin/ldconfig; } || true
	if [ -n "$ldc" ]; then
		cache=$("$ldc" -p 2>/dev/null || true)
		for lib in $(LC_ALL=C readelf -d "$BINDIR/scrawl" 2>/dev/null | sed -n 's/.*(NEEDED).*\[\(.*\)\].*/\1/p'); do
			case $cache in *"$lib ("*) ;; *) missing_libs="$missing_libs $lib" ;; esac
		done
	fi
fi

# lib_pkg LIB prints "apt-package dnf-package pacman-package" for a library we know, nothing
# for others. Debian and Ubuntu names; Fedora and Arch names where they are certain.
lib_pkg() {
	case $1 in
	libEGL.so.*) echo "libegl1 libglvnd-egl libglvnd" ;;
	libwayland-egl.so.*) echo "libwayland-egl1 libwayland-egl wayland" ;;
	libwayland-cursor.so.*) echo "libwayland-cursor0 libwayland-cursor wayland" ;;
	libwayland-client.so.*) echo "libwayland-client0 libwayland-client wayland" ;;
	libxkbcommon-x11.so.*) echo "libxkbcommon-x11-0 libxkbcommon-x11 libxkbcommon-x11" ;;
	libxkbcommon.so.*) echo "libxkbcommon0 libxkbcommon libxkbcommon" ;;
	libX11-xcb.so.*) echo "libx11-xcb1 libX11-xcb libx11" ;;
	libX11.so.*) echo "libx11-6 libX11 libx11" ;;
	libXcursor.so.*) echo "libxcursor1 libXcursor libxcursor" ;;
	libXfixes.so.*) echo "libxfixes3 libXfixes libxfixes" ;;
	libpipewire-0.3.so.*)
		# Debian 13 and Ubuntu 24.04 renamed the package (64-bit time_t transition).
		pw=libpipewire-0.3-0
		if command -v apt-cache >/dev/null 2>&1 && apt-cache show libpipewire-0.3-0t64 >/dev/null 2>&1; then pw=libpipewire-0.3-0t64; fi
		echo "$pw pipewire-libs libpipewire"
		;;
	esac
}
add_pkg() { case " $1 " in *" $2 "*) printf '%s' "$1" ;; *) printf '%s' "${1:+$1 }$2" ;; esac; }
lib_apt=""
lib_dnf=""
lib_pac=""
lib_other=""
for lib in $missing_libs; do
	set -- $(lib_pkg "$lib")
	if [ $# = 3 ]; then
		lib_apt=$(add_pkg "$lib_apt" "$1")
		lib_dnf=$(add_pkg "$lib_dnf" "$2")
		lib_pac=$(add_pkg "$lib_pac" "$3")
	else
		lib_other="$lib_other $lib"
	fi
done
lib_all=$(printf '%s ' $missing_libs)

case "${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}" in
ru*)
	echo "Установлено в $BINDIR."
	if [ "$on_path" = 0 ]; then
		echo "Каталога $BINDIR нет в PATH, поэтому команда scrawl не находится. Добавьте в ~/.profile строку"
		echo "  export PATH=\"$BINDIR:\$PATH\""
		echo "и войдите в сеанс заново, либо запускайте программу по полному пути."
	fi
	if [ -n "$missing_libs" ]; then
		echo "Не хватает библиотек: $lib_all"
		[ -z "$lib_apt" ] || echo "  Debian, Ubuntu: sudo apt install $lib_apt"
		[ -z "$lib_dnf" ] || echo "  Fedora: sudo dnf install $lib_dnf"
		[ -z "$lib_pac" ] || echo "  Arch: sudo pacman -S $lib_pac"
		[ -z "$lib_other" ] || echo "  Для остальных найдите пакет, содержащий:$lib_other"
	fi
	if [ -n "$missing" ]; then
		echo "Не хватает программ для буфера обмена и уведомлений. Поставьте:"
		echo "  sudo apt install$missing"
	fi
	case $hk in
	0) echo "Клавиша Print назначена на Scrawl." ;;
	10) echo "Клавиша Print уже назначена на Scrawl." ;;
	12) echo "Клавиша Print занята другим ярлыком: назначьте свою в scrawl -settings, вкладка «Система»." ;;
	13) echo "Не удалось назначить клавишу: сделайте это в scrawl -settings, вкладка «Система»." ;;
	esac
	[ "$gnome_print" = 1 ] && echo "Если Print открывает снимок GNOME, отключите его: Параметры → Клавиатура → Просмотр и настройка сочетаний → Снимки экрана."
	if [ "$x11" = 1 ]; then
		echo "Сеанс X11: портал не нужен. Поставьте xclip (буфер обмена) и назначьте клавишу: scrawl -settings, вкладка «Система»."
	else
		echo "Дальше: scrawl -settings, вкладка «Система»: выбрать мониторы, выдать разрешение, назначить клавишу."
		echo "(то же командами: scrawl -setup, затем scrawl -grant-screenshot)"
	fi
	;;
*)
	echo "Installed to $BINDIR."
	if [ "$on_path" = 0 ]; then
		echo "$BINDIR is not in your PATH, so the command scrawl is not found. Add this line to ~/.profile:"
		echo "  export PATH=\"$BINDIR:\$PATH\""
		echo "and log in again, or run the program by its full path."
	fi
	if [ -n "$missing_libs" ]; then
		echo "Shared libraries are missing: $lib_all"
		[ -z "$lib_apt" ] || echo "  Debian, Ubuntu: sudo apt install $lib_apt"
		[ -z "$lib_dnf" ] || echo "  Fedora: sudo dnf install $lib_dnf"
		[ -z "$lib_pac" ] || echo "  Arch: sudo pacman -S $lib_pac"
		[ -z "$lib_other" ] || echo "  For the rest, find the package that provides:$lib_other"
	fi
	if [ -n "$missing" ]; then
		echo "Programs for the clipboard and notifications are missing. Install them:"
		echo "  sudo apt install$missing"
	fi
	case $hk in
	0) echo "The Print key is now assigned to Scrawl." ;;
	10) echo "The Print key is already assigned to Scrawl." ;;
	12) echo "The Print key is taken by another shortcut: set your own in scrawl -settings, tab \"System\"." ;;
	13) echo "Could not assign the key: do it in scrawl -settings, tab \"System\"." ;;
	esac
	[ "$gnome_print" = 1 ] && echo "If Print opens GNOME's screenshot tool, turn that off: Settings > Keyboard > View and Customize Shortcuts > Screenshots."
	if [ "$x11" = 1 ]; then
		echo "X11 session: no portal setup needed. Install xclip (clipboard) and set a hotkey: scrawl -settings, tab \"System\"."
	else
		echo "Next: scrawl -settings, tab \"System\": pick the monitors, grant the permission, set a hotkey."
		echo "(the same with commands: scrawl -setup, then scrawl -grant-screenshot)"
	fi
	;;
esac
