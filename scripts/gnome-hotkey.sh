#!/bin/sh
# Назначает горячую клавишу GNOME на scrawl (настройка пользовательских
# комбинаций через gsettings). Использование:
#   scripts/gnome-hotkey.sh [КОМБИНАЦИЯ] [ПУТЬ_К_БИНАРНИКУ] [--dry-run] [--auto]
# По умолчанию: Print и ~/.local/bin/scrawl. Если комбинация уже занята
# другим пользовательским ярлыком, скрипт ничего не меняет и говорит, кем занята.
#
# С --auto скрипт ничего не печатает, а итог сообщает кодом выхода (для install.sh):
#   0 назначено, 10 ярлык scrawl уже есть, 11 не GNOME или нет gsettings,
#   12 комбинация занята чужим ярлыком, 13 не удалось записать настройки.
set -eu

accel="Print"
bin="$HOME/.local/bin/scrawl"
dry=0
auto=0
i=0
for arg in "$@"; do
	case "$arg" in
	--dry-run) dry=1 ;;
	--auto) auto=1 ;;
	*)
		i=$((i + 1))
		[ "$i" = 1 ] && accel="$arg"
		[ "$i" = 2 ] && bin="$arg"
		;;
	esac
done

schema=org.gnome.settings-daemon.plugins.media-keys
base=/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings

if [ "$auto" = 1 ]; then
	case ":${XDG_CURRENT_DESKTOP:-}:" in *[Gg][Nn][Oo][Mm][Ee]*) ;; *) exit 11 ;; esac
	command -v gsettings >/dev/null 2>&1 || exit 11
	gsettings get "$schema" custom-keybindings >/dev/null 2>&1 || exit 11
fi
list=$(gsettings get "$schema" custom-keybindings)

# Занята ли комбинация другим пользовательским ярлыком.
for path in $(echo "$list" | tr -d "[]'," ); do
	cur=$(gsettings get "$schema.custom-keybinding:$path" binding)
	if [ "$cur" = "'$accel'" ]; then
		name=$(gsettings get "$schema.custom-keybinding:$path" name)
		if [ "$auto" = 1 ]; then
			[ "$name" = "'scrawl'" ] && exit 10
			exit 12
		fi
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

[ "$auto" = 1 ] || echo "Будет создан ярлык custom$n: $accel -> $bin"
[ "$dry" = 1 ] && { echo "(--dry-run: ничего не изменено)"; exit 0; }

key="$schema.custom-keybinding:$path"
if [ "$auto" = 1 ]; then
	{
		gsettings set "$key" name "scrawl" &&
			gsettings set "$key" command "$bin" &&
			gsettings set "$key" binding "$accel" &&
			gsettings set "$schema" custom-keybindings "$newlist"
	} >/dev/null 2>&1 || exit 13
	exit 0
fi
gsettings set "$key" name "scrawl"
gsettings set "$key" command "$bin"
gsettings set "$key" binding "$accel"
gsettings set "$schema" custom-keybindings "$newlist"
echo "Готово."
