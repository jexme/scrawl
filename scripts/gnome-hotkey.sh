#!/bin/sh
# Assigns or removes a GNOME hotkey for scrawl (custom shortcuts configured via gsettings).
# Usage:
#   scripts/gnome-hotkey.sh [COMBINATION] [PATH_TO_BINARY] [--dry-run] [--auto]
#   scripts/gnome-hotkey.sh --remove [--dry-run] [--auto]
#
# Assigning: the default is Print and ~/.local/bin/scrawl.
#   - If a custom shortcut named "scrawl" already exists, it is UPDATED (command and binding),
#     so re-running the script, or switching from ~/.local/bin to the apt package, never
#     leaves two shortcuts or an old command behind. Extra "scrawl" shortcuts left by older
#     versions of this script are dropped.
#   - If the combination is already taken by another (not "scrawl") custom shortcut, the script
#     changes nothing and says who holds it.
#   - The binary path is shell-quoted in the command, so paths with spaces work.
# Removing (--remove): deletes every custom shortcut named "scrawl". Other shortcuts are
# never touched.
#
# With --auto the script prints nothing and reports the outcome through its exit code
# (for install.sh, packaging/uninstall.sh and make):
#   0   done: a shortcut was created, updated or (with --remove) removed
#   10  nothing to do: the shortcut already is exactly as requested (with --remove: there
#       is no scrawl shortcut)
#   11  not GNOME, or no gsettings
#   12  the combination is taken by a foreign shortcut (assigning only)
#   13  could not write the settings
# Without --auto the same cases print a message; a conflict (12) and a failure (13) exit 1.
set -eu

accel="Print"
bin="$HOME/.local/bin/scrawl"
dry=0
auto=0
remove=0
i=0
for arg in "$@"; do
	case "$arg" in
	--dry-run) dry=1 ;;
	--auto) auto=1 ;;
	--remove) remove=1 ;;
	*)
		i=$((i + 1))
		[ "$i" = 1 ] && accel="$arg"
		[ "$i" = 2 ] && bin="$arg"
		;;
	esac
done

schema=org.gnome.settings-daemon.plugins.media-keys
base=/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings

say() { [ "$auto" = 1 ] || echo "$@"; }
# finish CODE MESSAGE: exits with CODE silently under --auto, otherwise prints MESSAGE
# (to stderr for failures) and maps the codes 12 and 13 to exit 1.
finish() {
	code=$1
	shift
	if [ "$auto" = 1 ]; then exit "$code"; fi
	case $code in
	12 | 13) echo "$*" >&2; exit 1 ;;
	11) echo "$*" >&2; exit 1 ;;
	*) echo "$*"; exit 0 ;;
	esac
}

if [ "$auto" = 1 ]; then
	case ":${XDG_CURRENT_DESKTOP:-}:" in *[Gg][Nn][Oo][Mm][Ee]*) ;; *) exit 11 ;; esac
fi
command -v gsettings >/dev/null 2>&1 || finish 11 "gsettings not found: this works on GNOME only."
gsettings get "$schema" custom-keybindings >/dev/null 2>&1 || finish 11 "cannot read the GNOME custom shortcuts (no GNOME session?)."

# Quoting. The value of a string key is given to gsettings as a GVariant string: wrapped in
# double quotes, so that the single quotes of a shell-quoted command survive.
shquote() {
	case $1 in
	*[!A-Za-z0-9_./:@%+=,-]*) printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")" ;;
	*) printf '%s' "$1" ;;
	esac
}
gv() { printf '"%s"' "$(printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g')"; }
# How gsettings prints a string value back (single quotes; double if it contains a single
# quote). Strings with both kinds of quotes or backslashes are not compared (always rewritten).
gprint() {
	case $1 in
	*\\*) printf '%s' '?' ;;
	*\'*) case $1 in *\"*) printf '%s' '?' ;; *) printf '"%s"' "$1" ;; esac ;;
	*) printf "'%s'" "$1" ;;
	esac
}
unq() { printf '%s' "$1" | sed -e "s/^['\"]//" -e "s/['\"]\$//"; }
lower() { printf '%s' "$1" | tr 'A-Z' 'a-z'; }

list=$(gsettings get "$schema" custom-keybindings)
paths=$(printf '%s\n' "$list" | sed 's/^@as *//' | tr -d "[]',")

# Look through the existing shortcuts: ours (named "scrawl") and whoever holds the combination.
mine=""
foreign=""
for path in $paths; do
	k="$schema.custom-keybinding:$path"
	name=$(gsettings get "$k" name)
	if [ "$name" = "'scrawl'" ]; then
		mine="$mine $path"
	elif [ "$remove" = 0 ]; then
		cur=$(unq "$(gsettings get "$k" binding)")
		if [ -n "$cur" ] && [ "$(lower "$cur")" = "$(lower "$accel")" ]; then
			foreign="$name $path"
		fi
	fi
done

# The list of shortcuts without the given paths (all of $drop), in gsettings notation.
list_without() {
	out=""
	for p in $paths; do
		case " $1 " in *" $p "*) continue ;; esac
		out="$out${out:+, }'$p'"
	done
	printf '[%s]' "$out"
}

# Clears the fields of the shortcuts given as arguments (after they left the list).
reset_fields() {
	for p in "$@"; do
		for f in name command binding; do
			gsettings reset "$schema.custom-keybinding:$p" "$f" || return 1
		done
	done
}

if [ "$remove" = 1 ]; then
	[ -n "$mine" ] || finish 10 "No scrawl shortcut found."
	if [ "$dry" = 1 ]; then
		say "Would remove:$mine"
		say "(--dry-run: nothing changed)"
		exit 0
	fi
	{
		gsettings set "$schema" custom-keybindings "$(list_without "$mine")" &&
			reset_fields $mine
	} >/dev/null 2>&1 || finish 13 "Could not remove the shortcut: do it in the keyboard settings (Shortcuts, Custom Shortcuts)."
	finish 0 "The scrawl shortcut was removed."
fi

# Assigning.
if [ -n "$foreign" ]; then
	finish 12 "The combination $accel is already taken by the shortcut ${foreign% *} (${foreign##* }). Free it in the keyboard settings."
fi

cmd=$(shquote "$bin")
if [ -n "$mine" ]; then
	first=${mine# }
	first=${first%% *}
	extra=${mine# }
	case $extra in *" "*) extra=${extra#* } ;; *) extra="" ;; esac
	k="$schema.custom-keybinding:$first"
	if [ -z "$extra" ] &&
		[ "$(gsettings get "$k" command)" = "$(gprint "$cmd")" ] &&
		[ "$(gsettings get "$k" binding)" = "$(gprint "$accel")" ]; then
		finish 10 "The scrawl shortcut already is $accel -> $cmd."
	fi
	say "The scrawl shortcut ($first) will be updated: $accel -> $cmd"
	[ -z "$extra" ] || say "Duplicate scrawl shortcuts will be removed:$(printf ' %s' $extra)"
	[ "$dry" = 1 ] && { say "(--dry-run: nothing changed)"; exit 0; }
	{
		gsettings set "$k" command "$(gv "$cmd")" &&
			gsettings set "$k" binding "$(gv "$accel")" &&
			if [ -n "$extra" ]; then
				gsettings set "$schema" custom-keybindings "$(list_without "$extra")" &&
					reset_fields $extra
			fi
	} >/dev/null 2>&1 || finish 13 "Could not update the shortcut: do it in scrawl -settings, tab \"System\"."
	finish 0 "Done."
fi

# No shortcut of ours yet: create one in the first free slot.
n=0
while printf '%s\n' "$list" | grep -q "custom-keybindings/custom$n/"; do n=$((n + 1)); done
path="$base/custom$n/"
newlist=$(printf '%s\n' "$list" | sed 's/^@as *//')
case $newlist in
"[]") newlist="['$path']" ;;
*) newlist=$(printf '%s\n' "$newlist" | sed "s|]\$|, '$path']|") ;;
esac

say "A shortcut custom$n will be created: $accel -> $cmd"
[ "$dry" = 1 ] && { say "(--dry-run: nothing changed)"; exit 0; }

key="$schema.custom-keybinding:$path"
# The shortcut goes into the list last: until its fields are filled in, GNOME does not see it.
{
	gsettings set "$key" name "$(gv scrawl)" &&
		gsettings set "$key" command "$(gv "$cmd")" &&
		gsettings set "$key" binding "$(gv "$accel")" &&
		gsettings set "$schema" custom-keybindings "$newlist"
} >/dev/null 2>&1 || finish 13 "Could not assign the key: do it in scrawl -settings, tab \"System\"."
finish 0 "Done."
