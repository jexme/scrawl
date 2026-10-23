#!/bin/sh
# Downloads the libpipewire headers (packages libpipewire-0.3-dev and libspa-0.2-dev) and
# unpacks them into .deps/pipewire-dev without sudo and without installing anything into
# the system. It is needed if you have no rights for `sudo apt install
# libpipewire-0.3-dev`. The Makefile picks this folder up by itself. Works on Debian and
# Ubuntu (apt-get download and dpkg -x).
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
debs="$root/.deps/debs"
dst="$root/.deps/pipewire-dev"

mkdir -p "$debs"
cd "$debs"
# The version of the dev packages must match the installed library.
/usr/bin/apt-get download libpipewire-0.3-dev libspa-0.2-dev

rm -rf "$dst"
mkdir -p "$dst"
for d in "$debs"/*.deb; do
	dpkg -x "$d" "$dst"
done

# In the dev package the libpipewire-0.3.so link points to a file of the runtime package,
# which is absent from the unpacked folder: we redirect it to the system library.
lib=$(ls /usr/lib/*/libpipewire-0.3.so.0 | head -n 1)
ln -sfn "$lib" "$dst/usr/lib/$(basename "$(dirname "$lib")")/libpipewire-0.3.so"
echo "PipeWire headers unpacked into $dst"
