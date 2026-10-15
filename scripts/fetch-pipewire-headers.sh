#!/bin/sh
# Скачивает заголовки libpipewire (пакеты libpipewire-0.3-dev и libspa-0.2-dev) и
# распаковывает в .deps/pipewire-dev без sudo и без установки в систему. Нужен,
# если нет прав на `sudo apt install libpipewire-0.3-dev`. Makefile подхватывает
# эту папку сам. Работает на Debian и Ubuntu (apt-get download и dpkg -x).
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
debs="$root/.deps/debs"
dst="$root/.deps/pipewire-dev"

mkdir -p "$debs"
cd "$debs"
# Версия dev-пакетов должна совпадать с установленной библиотекой.
/usr/bin/apt-get download libpipewire-0.3-dev libspa-0.2-dev

rm -rf "$dst"
mkdir -p "$dst"
for d in "$debs"/*.deb; do
	dpkg -x "$d" "$dst"
done

# В dev-пакете ссылка libpipewire-0.3.so указывает на файл runtime-пакета,
# которого в распакованной папке нет: перенаправляем на системную библиотеку.
lib=$(ls /usr/lib/*/libpipewire-0.3.so.0 | head -n 1)
ln -sfn "$lib" "$dst/usr/lib/$(basename "$(dirname "$lib")")/libpipewire-0.3.so"
echo "Заголовки PipeWire распакованы в $dst"
