// Package hotkey assigns the launch hotkey through GNOME custom shortcuts (gsettings).
// Only the program's own shortcut is changed; other shortcuts are left alone, and a
// combination they already use results in an error.
package hotkey

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jexme/scrawl/internal/i18n"
)

const (
	schema = "org.gnome.settings-daemon.plugins.media-keys"
	base   = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
	// ShortcutName is the name of the shortcut in the GNOME keyboard settings.
	ShortcutName = "scrawl"
)

// Built-in GNOME shortcuts the combination may clash with.
var builtin = [][2]string{
	{"org.gnome.shell.keybindings", "show-screenshot-ui"},
	{"org.gnome.shell.keybindings", "screenshot"},
	{"org.gnome.shell.keybindings", "screenshot-window"},
	{"org.gnome.shell.keybindings", "show-screen-recording-ui"},
}

// gsettings runs the gsettings command and returns its output. It is replaced in tests.
var gsettings = func(args ...string) (string, error) {
	out, err := exec.Command("gsettings", args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return "", errors.New(text)
	}
	return text, nil
}

var accelRe = regexp.MustCompile(`(?i)^(<(Super|Control|Primary|Shift|Alt|Mod1|Ctrl)>)*[A-Za-z0-9_]+$`)

// NormalizeAccel turns the combination as people write it ("Ctrl+Alt+P", "super + s") into the
// GTK form ("<Control><Alt>p", "<Super>s"). A string that already has angle brackets, or is
// a single key name such as Print, is returned trimmed and otherwise unchanged.
func NormalizeAccel(in string) string {
	in = strings.TrimSpace(in)
	if in == "" || strings.Contains(in, "<") || !strings.Contains(in, "+") {
		return in
	}
	parts := strings.Split(in, "+")
	var out strings.Builder
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if i == len(parts)-1 {
			if len(part) == 1 {
				part = strings.ToLower(part) // GTK writes letter keys in lower case
			}
			out.WriteString(part)
			continue
		}
		switch strings.ToLower(part) {
		case "ctrl", "control", "primary":
			out.WriteString("<Control>")
		case "alt", "mod1":
			out.WriteString("<Alt>")
		case "shift":
			out.WriteString("<Shift>")
		case "super", "win", "windows", "meta", "cmd":
			out.WriteString("<Super>")
		default:
			return in // unknown modifier: leave it for ValidateAccel to refuse
		}
	}
	return out.String()
}

// ValidateAccel checks a combination written in GTK format: Print, <Super>s,
// <Control><Alt>p.
func ValidateAccel(accel string) error {
	if !accelRe.MatchString(accel) {
		return errors.New(i18n.T("use the GNOME format: Print, <Super>s, <Control><Alt>p"))
	}
	return nil
}

// Result describes the outcome of an assignment.
type Result struct {
	Updated bool   // an existing shortcut of the program was updated, not a new one created
	Warning string // the combination coincides with a built-in GNOME shortcut
}

// Conflict is a combination taken by someone else's custom shortcut.
type Conflict struct{ Name, Accel string }

func (c *Conflict) Error() string {
	return fmt.Sprintf(i18n.T("the combination %s is already taken by the shortcut \"%s\": free it in the keyboard settings"), c.Accel, c.Name)
}

type shortcut struct{ path, name, command, binding string }

// parseStrings parses a list of strings in GVariant notation: ['a', 'b'] or @as [].
func parseStrings(v string) []string {
	var out []string
	for i := 0; i < len(v); i++ {
		if v[i] != '\'' {
			continue
		}
		var sb strings.Builder
		for i++; i < len(v) && v[i] != '\''; i++ {
			if v[i] == '\\' && i+1 < len(v) {
				i++
			}
			sb.WriteByte(v[i])
		}
		out = append(out, sb.String())
	}
	return out
}

// quote writes a string as a GVariant.
func quote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

func quoteList(items []string) string {
	q := make([]string, len(items))
	for i, it := range items {
		q[i] = quote(it)
	}
	return "[" + strings.Join(q, ", ") + "]"
}

func (s shortcut) key() string { return schema + ".custom-keybinding:" + s.path }

func listShortcuts() ([]shortcut, error) {
	out, err := gsettings("get", schema, "custom-keybindings")
	if err != nil {
		return nil, fmt.Errorf("gsettings: %w", err)
	}
	var list []shortcut
	for _, path := range parseStrings(out) {
		sc := shortcut{path: path}
		for _, f := range []struct {
			key string
			dst *string
		}{{"name", &sc.name}, {"command", &sc.command}, {"binding", &sc.binding}} {
			v, err := gsettings("get", sc.key(), f.key)
			if err != nil {
				return nil, fmt.Errorf("gsettings: %w", err)
			}
			if s := parseStrings(v); len(s) > 0 {
				*f.dst = s[0]
			}
		}
		list = append(list, sc)
	}
	return list, nil
}

// ours reports that the shortcut belongs to this program (including under its former
// name).
func (s shortcut) ours() bool {
	exe := filepath.Base(strings.Fields(s.command + " ")[0])
	return s.name == ShortcutName || exe == "scrawl" || exe == "screen-editor"
}

func sameAccel(a, b string) bool { return strings.EqualFold(a, b) }

// Current returns the combination of the program's shortcut, or "" if there is none.
func Current() string {
	if onWindows {
		return winCurrent()
	}
	list, err := listShortcuts()
	if err != nil {
		return ""
	}
	for _, sc := range list {
		if sc.ours() {
			return sc.binding
		}
	}
	return ""
}

// Assign assigns the combination accel to launch bin. If the program's shortcut exists, it
// is updated. If not, a new one is created. Someone else's shortcut with the same
// combination yields *Conflict.
func Assign(accel, bin string) (Result, error) {
	if onWindows {
		return winAssign(accel)
	}
	var res Result
	if err := ValidateAccel(accel); err != nil {
		return res, err
	}
	list, err := listShortcuts()
	if err != nil {
		return res, err
	}
	var mine *shortcut
	for i, sc := range list {
		if sc.ours() {
			mine = &list[i]
			continue
		}
		if sameAccel(sc.binding, accel) {
			return res, &Conflict{Name: sc.name, Accel: accel}
		}
	}
	for _, b := range builtin {
		if out, err := gsettings("get", b[0], b[1]); err == nil {
			for _, other := range parseStrings(out) {
				if sameAccel(other, accel) {
					res.Warning = fmt.Sprintf(i18n.T("the combination %s is also used by a built-in GNOME shortcut (%s)"), accel, b[1])
				}
			}
		}
	}

	var newList []string // the list of shortcuts with the new one; empty if we update our own
	if mine != nil {
		res.Updated = true
	} else {
		taken := map[string]bool{}
		for _, sc := range list {
			taken[sc.path] = true
			newList = append(newList, sc.path)
		}
		n := 0
		for taken[fmt.Sprintf("%scustom%d/", base, n)] {
			n++
		}
		mine = &shortcut{path: fmt.Sprintf("%scustom%d/", base, n)}
		newList = append(newList, mine.path)
	}
	for _, f := range [][2]string{{"name", ShortcutName}, {"command", bin}, {"binding", accel}} {
		if _, err := gsettings("set", mine.key(), f[0], quote(f[1])); err != nil {
			return res, fmt.Errorf("gsettings: %w", err)
		}
	}
	// The shortcut goes into the list last: until its fields are filled in, GNOME does not see
	// it.
	if newList != nil {
		if _, err := gsettings("set", schema, "custom-keybindings", quoteList(newList)); err != nil {
			return res, fmt.Errorf("gsettings: %w", err)
		}
	}
	return res, nil
}

// Remove deletes the program's own shortcut (or shortcuts) from the GNOME keyboard settings and
// returns how many were removed. A package manager cannot do this for a user on removal, so
// the user runs `scrawl -remove-hotkey` first. Other shortcuts are not touched.
func Remove() (int, error) {
	if onWindows {
		return 0, nil // the combination lives in a file next to the settings, nothing in the system
	}
	list, err := listShortcuts()
	if err != nil {
		return 0, err
	}
	var keep []string
	var gone []shortcut
	for _, sc := range list {
		if sc.ours() {
			gone = append(gone, sc)
		} else {
			keep = append(keep, sc.path)
		}
	}
	if len(gone) == 0 {
		return 0, nil
	}
	// The list first: GNOME stops looking at the shortcut at once, and only then are its fields reset.
	if _, err := gsettings("set", schema, "custom-keybindings", quoteList(keep)); err != nil {
		return 0, fmt.Errorf("gsettings: %w", err)
	}
	for _, sc := range gone {
		for _, f := range []string{"name", "command", "binding"} {
			gsettings("reset", sc.key(), f) // a failure here leaves a harmless unlisted entry
		}
	}
	return len(gone), nil
}

// FreeBuiltin removes the combination accel from the built-in GNOME shortcuts that use it: the
// GNOME screenshot tool takes Print by default, so without this the Print key opens it instead
// of Scrawl. It returns the names of the settings that were changed. Nothing else is touched,
// and the user can bring a setting back with `gsettings reset org.gnome.shell.keybindings NAME`.
// This only runs on the user's explicit request.
func FreeBuiltin(accel string) ([]string, error) {
	if onWindows {
		return nil, nil
	}
	if err := ValidateAccel(accel); err != nil {
		return nil, err
	}
	var changed []string
	for _, b := range builtin {
		out, err := gsettings("get", b[0], b[1])
		if err != nil {
			continue // not GNOME, or no such setting in this version
		}
		var keep []string
		hit := false
		for _, other := range parseStrings(out) {
			if sameAccel(other, accel) {
				hit = true
			} else {
				keep = append(keep, other)
			}
		}
		if !hit {
			continue
		}
		if _, err := gsettings("set", b[0], b[1], quoteList(keep)); err != nil {
			return changed, fmt.Errorf("gsettings: %w", err)
		}
		changed = append(changed, b[1])
	}
	return changed, nil
}

// PreferredBinary returns the path worth writing into the shortcut: the permanent
// ~/.local/bin/scrawl if it leads to the running binary, otherwise the binary itself.
func PreferredBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if home, err := os.UserHomeDir(); err == nil {
		link := filepath.Join(home, ".local", "bin", "scrawl")
		a, errA := filepath.EvalSymlinks(link)
		b, errB := filepath.EvalSymlinks(exe)
		if errA == nil && errB == nil && a == b {
			return link, nil
		}
	}
	return exe, nil
}
