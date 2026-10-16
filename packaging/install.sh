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

echo "Установлено в $BINDIR. Дальше: scrawl -setup, затем scrawl -grant-screenshot"
echo "(или те же кнопки в окне настроек: scrawl -settings)"
