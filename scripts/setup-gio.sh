#!/bin/sh
# Готовит third_party/gio: исходники Gio той версии, что в go.mod, плюс наш патч
# third_party/gio.patch (app.FullscreenAt: fullscreen на нужном мониторе Wayland).
# Повторный запуск ничего не делает, пока версия и патч не менялись; -f пересоздаёт.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
dst="$root/third_party/gio"
patchfile="$root/third_party/gio.patch"
marker="$dst/.screen-editor-patched"

ver=$(grep -E '^[[:space:]]*gioui\.org v' "$root/go.mod" | awk '{print $2}' | head -n 1)
[ -n "$ver" ] || { echo "не нашёл версию gioui.org в go.mod" >&2; exit 1; }
want="$ver $(sha256sum "$patchfile" | cut -d' ' -f1)"

if [ -f "$marker" ] && [ "$(cat "$marker")" = "$want" ] && [ "${1:-}" != "-f" ]; then
	echo "Gio $ver уже подготовлен"
	exit 0
fi

# Скачиваем из временного каталога: в корне проекта go.mod ссылается на third_party/gio,
# которого может ещё не быть.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
src=$(cd "$tmp" && go mod download -json "gioui.org@$ver" | sed -n 's/.*"Dir": "\(.*\)",\{0,1\}$/\1/p' | sed 's/",$//; s/"$//' | head -n 1)
[ -d "$src" ] || { echo "не удалось скачать gioui.org@$ver" >&2; exit 1; }

rm -rf "$dst"
cp -R "$src" "$dst"
chmod -R u+w "$dst"
patch -p1 -s -d "$dst" < "$patchfile"
echo "$want" > "$marker"
echo "Gio $ver подготовлен с патчем"
