#!/bin/sh
# Собирает подписанный apt-репозиторий из .deb-файлов.
# Использование: scripts/build-apt-repo.sh ПАПКА_С_DEB ВЫХОДНАЯ_ПАПКА
# Подпись: ключ GPG_KEY_ID из текущей связки gpg (в CI его кладёт workflow). Без GPG_KEY_ID
# репозиторий собирается без подписи (для проверки структуры).
# Нужны: dpkg-dev (dpkg-scanpackages), gzip, gpg.
set -eu

in=${1:?папка с .deb}
out=${2:?выходная папка}
suite=stable
comp=main
arch=amd64

mkdir -p "$out"
out=$(cd "$out" && pwd)
pool="pool/$comp"
mkdir -p "$out/$pool" "$out/dists/$suite/$comp/binary-$arch"
cp "$in"/*.deb "$out/$pool/"

(
	cd "$out"
	bin="dists/$suite/$comp/binary-$arch"
	dpkg-scanpackages --multiversion --arch "$arch" "$pool" /dev/null > "$bin/Packages" 2>/dev/null
	gzip -9nk -f "$bin/Packages"

	{
		echo "Origin: Scrawl"
		echo "Label: Scrawl"
		echo "Suite: $suite"
		echo "Codename: $suite"
		echo "Date: $(LC_ALL=C date -Ru)"
		echo "Architectures: $arch"
		echo "Components: $comp"
		echo "Description: Scrawl screenshot tool"
		echo "SHA256:"
		for f in "$bin/Packages" "$bin/Packages.gz"; do
			rel=${f#dists/$suite/}
			printf ' %s %s %s\n' "$(sha256sum "$f" | cut -d' ' -f1)" "$(wc -c < "$f")" "$rel"
		done
	} > "dists/$suite/Release"

	if [ -n "${GPG_KEY_ID:-}" ]; then
		gpg --batch --yes --pinentry-mode loopback ${GPG_PASSPHRASE:+--passphrase "$GPG_PASSPHRASE"} \
			-u "$GPG_KEY_ID" --clearsign -o "dists/$suite/InRelease" "dists/$suite/Release"
		gpg --batch --yes --pinentry-mode loopback ${GPG_PASSPHRASE:+--passphrase "$GPG_PASSPHRASE"} \
			-u "$GPG_KEY_ID" --armor --detach-sign -o "dists/$suite/Release.gpg" "dists/$suite/Release"
		# Открытый ключ для apt: signed-by принимает бинарный (dearmored) файл.
		gpg --batch --yes --export "$GPG_KEY_ID" > scrawl-archive-keyring.gpg
	fi
)

# Страница-заглушка: Pages отдаёт индекс, а apt-клиентам нужны только файлы выше.
cat > "$out/index.html" <<'HTML'
<!doctype html><meta charset="utf-8"><title>Scrawl apt repository</title>
<h1>Scrawl apt repository</h1>
<p>Setup instructions: <a href="https://github.com/jexme/scrawl#install">github.com/jexme/scrawl</a></p>
HTML
echo "$out"
