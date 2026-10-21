#!/bin/sh
# Assigns a GNOME hotkey to scrawl (custom shortcuts configured via gsettings). Usage:
#   scripts/gnome-hotkey.sh [COMBINATION] [PATH_TO_BINARY] [--dry-run] [--auto]
# Default: Print and ~/.local/bin/scrawl. If the combination is already taken by
# another custom shortcut, the script changes nothing and says who holds it.
#
# With --auto the script prints nothing and reports the outcome through its exit code
# (for install.sh):
#   0 assigned, 10 a scrawl shortcut already exists, 11 not GNOME or no gsettings,
#   12 the combination is taken by a foreign shortcut, 13 could not write the settings.
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

# Whether the combination is taken by another custom shortcut.
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

# The first free shortcut number.
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
