#!/bin/sh
# Builds a signed apt repository from .deb files.
# Usage: scripts/build-apt-repo.sh DEB_DIR OUTPUT_DIR
# Signing: the key GPG_KEY_ID from the current gpg keyring (in CI the workflow puts it
# there). Without GPG_KEY_ID the repository is built unsigned (to check the structure).
# Besides the repository itself it writes the files users need to set it up:
#   scrawl-archive-keyring.gpg  the public key, binary (for [signed-by=...] in one-line sources)
#   scrawl-archive-keyring.asc  the same key, ASCII-armored
#   scrawl.sources              a ready deb822 source file with the armored key inline
# APT_REPO_URL overrides the public address written into scrawl.sources.
# Requires: dpkg-dev (dpkg-scanpackages), gzip, gpg.
set -eu

in=${1:?directory with the .deb files}
out=${2:?output directory}
suite=stable
comp=main
url=${APT_REPO_URL:-https://jexme.github.io/scrawl}

mkdir -p "$out"
out=$(cd "$out" && pwd)
pool="pool/$comp"
mkdir -p "$out/$pool"
cp "$in"/*.deb "$out/$pool/"

# The architectures present in the packages (amd64 always, arm64 when that build exists).
archs=$(for f in "$out/$pool"/*.deb; do dpkg-deb -f "$f" Architecture; done | sort -u | tr '\n' ' ' | sed 's/ $//')
[ -n "$archs" ] || { echo "no .deb files in $in" >&2; exit 1; }

(
	cd "$out"
	sums=""
	for arch in $archs; do
		bin="dists/$suite/$comp/binary-$arch"
		mkdir -p "$bin"
		dpkg-scanpackages --multiversion --arch "$arch" "$pool" /dev/null > "$bin/Packages" 2>/dev/null
		gzip -9nk -f "$bin/Packages"
		for f in "$bin/Packages" "$bin/Packages.gz"; do
			rel=${f#dists/$suite/}
			sums="$sums$(printf ' %s %s %s\n' "$(sha256sum "$f" | cut -d' ' -f1)" "$(wc -c < "$f")" "$rel")
"
		done
	done

	{
		echo "Origin: Scrawl"
		echo "Label: Scrawl"
		echo "Suite: $suite"
		echo "Codename: $suite"
		echo "Date: $(LC_ALL=C date -Ru)"
		echo "Architectures: $archs"
		echo "Components: $comp"
		echo "Description: Scrawl screenshot tool"
		echo "SHA256:"
		printf '%s' "$sums"
	} > "dists/$suite/Release"

	if [ -n "${GPG_KEY_ID:-}" ]; then
		# The passphrase goes to gpg through a pipe (--passphrase-fd 0), never through argv,
		# where every local user could read it in the process list.
		sign() {
			if [ -n "${GPG_PASSPHRASE:-}" ]; then
				printf '%s' "$GPG_PASSPHRASE" | gpg --batch --yes --pinentry-mode loopback --passphrase-fd 0 -u "$GPG_KEY_ID" "$@"
			else
				gpg --batch --yes --pinentry-mode loopback -u "$GPG_KEY_ID" "$@"
			fi
		}
		sign --clearsign -o "dists/$suite/InRelease" "dists/$suite/Release"
		sign --armor --detach-sign -o "dists/$suite/Release.gpg" "dists/$suite/Release"
		# The public key for apt: signed-by accepts a binary (dearmored) file, and the armored
		# one can be embedded into a deb822 .sources file.
		gpg --batch --yes --export "$GPG_KEY_ID" > scrawl-archive-keyring.gpg
		gpg --batch --yes --armor --export "$GPG_KEY_ID" > scrawl-archive-keyring.asc
		{
			echo "Types: deb"
			echo "URIs: $url"
			echo "Suites: $suite"
			echo "Components: $comp"
			echo "Architectures: $archs"
			echo "Signed-By:"
			# deb822 multi-line value: every line starts with a space, an empty line is " .".
			sed -e 's/^$/./' -e 's/^/ /' scrawl-archive-keyring.asc
		} > scrawl.sources
	fi
)

# A placeholder page: Pages serves an index, while apt clients need only the files above.
cat > "$out/index.html" <<'HTML'
<!doctype html><meta charset="utf-8"><title>Scrawl apt repository</title>
<h1>Scrawl apt repository</h1>
<p>Setup instructions: <a href="https://github.com/jexme/scrawl#install">github.com/jexme/scrawl</a></p>
HTML
echo "$out"
