#!/bin/sh
# Ставит Scrawl из распакованного архива релиза: бинарник и .desktop-файл.
# Переменные: PREFIX (по умолчанию ~/.local), BINDIR, APPDIR.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
PREFIX=${PREFIX:-$HOME/.local}
BINDIR=${BINDIR:-$PREFIX/bin}
APPDIR=${APPDIR:-$PREFIX/share/applications}
desktop=io.github.jexme.Scrawl.desktop

install -Dm755 "$here/scrawl" "$BINDIR/scrawl"
install -d "$APPDIR"
# В Exec подставляется абсолютный путь: portal сопоставляет приложение с
# .desktop-файлом, только если Exec существует.
sed "s|^Exec=scrawl|Exec=$BINDIR/scrawl|" "$here/$desktop" > "$APPDIR/$desktop"
chmod 644 "$APPDIR/$desktop"

# Сообщение по языку системы и типу сеанса: в X11 портала нет, мониторы выбирать не нужно.
x11=0
[ -z "${WAYLAND_DISPLAY:-}" ] && [ -n "${DISPLAY:-}" ] && x11=1

# Программы, без которых не работают буфер обмена и уведомления: в Wayland wl-copy (пакет
# wl-clipboard), в X11 xclip, везде notify-send (libnotify-bin). Имена пакетов для Debian и Ubuntu.
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
	if [ "$x11" = 1 ]; then
		echo "X11 session: no portal setup needed. Install xclip (clipboard) and set a hotkey: scrawl -settings, tab \"System\"."
	else
		echo "Next: scrawl -settings, tab \"System\": pick the monitors, grant the permission, set a hotkey."
		echo "(the same with commands: scrawl -setup, then scrawl -grant-screenshot)"
	fi
	;;
esac
