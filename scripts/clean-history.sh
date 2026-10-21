#!/bin/sh
# Prepares a clean history for publication in the local branch clean/main: removes the copy of
# Gio (third_party/gio/) from all commits, replaces the author's personal email and
# erases that same email from file contents (replacing it with old@example.com).
# The source branches and origin are not changed, nothing is pushed. Needs git-filter-repo.
#
# Usage: scripts/clean-history.sh NAME NEW_EMAIL OLD_EMAIL [BRANCH]
#   scripts/clean-history.sh jexme 12345+jexme@users.noreply.github.com old@example.com
# The default branch is the current one. Result: git push NEW_REPOSITORY clean/main:main
set -eu

[ $# -ge 3 ] || { sed -n '2,11p' "$0"; exit 1; }
name=$1 new=$2 old=$3
branch=${4:-$(git rev-parse --abbrev-ref HEAD)}
command -v git-filter-repo >/dev/null || { echo "нужен git-filter-repo (sudo apt install git-filter-repo)" >&2; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
git clone -q --no-local --branch "$branch" --single-branch . "$tmp/clean"
echo "$name <$new> <$old>" > "$tmp/mailmap"
echo "$old==>old@example.com" > "$tmp/replace"
(cd "$tmp/clean" && git filter-repo --force --path third_party/gio/ --invert-paths --mailmap "$tmp/mailmap" --replace-text "$tmp/replace" 2>&1 | tail -1)

# Checks: there must be neither a copy of Gio nor the old address.
cd "$tmp/clean"
if git log --name-only --format= | grep -q '^third_party/gio/'; then echo "в истории осталась копия Gio" >&2; exit 1; fi
if git log --format='%ae %ce' | grep -q "$old"; then echo "в истории остался старый email" >&2; exit 1; fi
if [ -n "$(git log --all -S"$old" --format=%h | head -n 1)" ]; then echo "старый email остался в тексте файлов" >&2; exit 1; fi
cd - >/dev/null

git fetch -q "$tmp/clean" "$branch"
git branch -f clean/main FETCH_HEAD
echo "Готово: ветка clean/main ($(git rev-list --count clean/main) коммитов), дерево совпадает с $branch: $([ "$(git rev-parse clean/main^{tree})" = "$(git rev-parse "$branch"^{tree})" ] && echo да || echo НЕТ)"
