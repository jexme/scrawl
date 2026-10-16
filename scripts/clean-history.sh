#!/bin/sh
# Готовит чистую историю для публикации в локальной ветке clean/main: убирает из
# всех коммитов копию Gio (third_party/gio/) и заменяет личный email автора.
# Исходные ветки и origin не меняются, ничего не пушится. Нужен git-filter-repo.
#
# Использование: scripts/clean-history.sh ИМЯ НОВЫЙ_EMAIL СТАРЫЙ_EMAIL [ВЕТКА]
#   scripts/clean-history.sh jexme 22722008+jexme@users.noreply.github.com old@example.com
# Ветка по умолчанию текущая. Результат: git push НОВЫЙ_РЕПОЗИТОРИЙ clean/main:main
set -eu

[ $# -ge 3 ] || { sed -n '2,11p' "$0"; exit 1; }
name=$1 new=$2 old=$3
branch=${4:-$(git rev-parse --abbrev-ref HEAD)}
command -v git-filter-repo >/dev/null || { echo "нужен git-filter-repo (sudo apt install git-filter-repo)" >&2; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
git clone -q --no-local --branch "$branch" --single-branch . "$tmp/clean"
echo "$name <$new> <$old>" > "$tmp/mailmap"
(cd "$tmp/clean" && git filter-repo --force --path third_party/gio/ --invert-paths --mailmap "$tmp/mailmap" 2>&1 | tail -1)

# Проверки: копии Gio и старого адреса быть не должно.
cd "$tmp/clean"
if git log --name-only --format= | grep -q '^third_party/gio/'; then echo "в истории осталась копия Gio" >&2; exit 1; fi
if git log --format='%ae %ce' | grep -q "$old"; then echo "в истории остался старый email" >&2; exit 1; fi
cd - >/dev/null

git fetch -q "$tmp/clean" "$branch"
git branch -f clean/main FETCH_HEAD
echo "Готово: ветка clean/main ($(git rev-list --count clean/main) коммитов), дерево совпадает с $branch: $([ "$(git rev-parse clean/main^{tree})" = "$(git rev-parse "$branch"^{tree})" ] && echo да || echo НЕТ)"
