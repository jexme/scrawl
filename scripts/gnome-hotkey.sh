#!/bin/sh
# Назначает горячую клавишу GNOME на scrawl (настройка пользовательских
# комбинаций через gsettings). Использование:
#   scripts/gnome-hotkey.sh [КОМБИНАЦИЯ] [ПУТЬ_К_БИНАРНИКУ] [--dry-run]
# По умолчанию: Print и ~/.local/bin/scrawl. Если комбинация уже занята
# другим пользовательским ярлыком, скрипт ничего не меняет и говорит, кем занята.
set -eu

accel="Print"
bin="$HOME/.local/bin/scrawl"
dry=0
i=0
for arg in "$@"; do
	case "$arg" in
	--dry-run) dry=1 ;;
	*)
		i=$((i + 1))
		[ "$i" = 1 ] && accel="$arg"
		[ "$i" = 2 ] && bin="$arg"
		;;
	esac
done

schema=org.gnome.settings-daemon.plugins.media-keys
base=/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings
list=$(gsettings get "$schema" custom-keybindings)

# Занята ли комбинация другим пользовательским ярлыком.
for path in $(echo "$list" | tr -d "[]'," ); do
	cur=$(gsettings get "$schema.custom-keybinding:$path" binding)
	if [ "$cur" = "'$accel'" ]; then
		name=$(gsettings get "$schema.custom-keybinding:$path" name)
		echo "Комбинация $accel уже занята ярлыком $name ($path). Освободите её в настройках клавиатуры." >&2
		exit 1
	fi
done

# Первый свободный номер ярлыка.
n=0
while echo "$list" | grep -q "custom-keybindings/custom$n/"; do n=$((n + 1)); done
path="$base/custom$n/"
if [ "$list" = "@as []" ]; then
	newlist="['$path']"
else
	newlist=$(echo "$list" | sed "s|]\$|, '$path']|")
fi

echo "Будет создан ярлык custom$n: $accel -> $bin"
[ "$dry" = 1 ] && { echo "(--dry-run: ничего не изменено)"; exit 0; }

key="$schema.custom-keybinding:$path"
gsettings set "$key" name "scrawl"
gsettings set "$key" command "$bin"
gsettings set "$key" binding "$accel"
gsettings set "$schema" custom-keybindings "$newlist"
echo "Готово."
