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

case "${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}" in
ru*)
	echo "Установлено в $BINDIR."
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
