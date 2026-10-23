package hotkey

import (
	"errors"
	"strings"
	"testing"
)

// fakeGSettings imitates the gsettings storage: the key "schema key" -> a GVariant value.
type fakeGSettings struct {
	values map[string]string
	sets   []string
}

func (f *fakeGSettings) run(args ...string) (string, error) {
	key := strings.Join(args[1:len(args)-boolInt(args[0] == "set")], " ")
	switch args[0] {
	case "get":
		v, ok := f.values[key]
		if !ok {
			return "", errors.New("нет ключа " + key)
		}
		return v, nil
	case "set":
		f.values[key] = args[len(args)-1]
		f.sets = append(f.sets, key)
		return "", nil
	case "reset":
		delete(f.values, key)
		return "", nil
	}
	return "", errors.New("неизвестная команда")
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func install(t *testing.T, values map[string]string) *fakeGSettings {
	t.Helper()
	f := &fakeGSettings{values: values}
	old, oldWin := gsettings, onWindows
	gsettings, onWindows = f.run, false
	t.Cleanup(func() { gsettings, onWindows = old, oldWin })
	return f
}

func entry(f map[string]string, n int, name, command, binding string) string {
	path := base + "custom" + string(rune('0'+n)) + "/"
	f[schema+".custom-keybinding:"+path+" name"] = quote(name)
	f[schema+".custom-keybinding:"+path+" command"] = quote(command)
	f[schema+".custom-keybinding:"+path+" binding"] = quote(binding)
	return path
}

func TestParseAndQuote(t *testing.T) {
	if got := parseStrings("@as []"); len(got) != 0 {
		t.Errorf("пустой список: %v", got)
	}
	if got := parseStrings(`['/a/', '/b/']`); len(got) != 2 || got[1] != "/b/" {
		t.Errorf("список: %v", got)
	}
	if got := parseStrings(`'it\'s'`); len(got) != 1 || got[0] != "it's" {
		t.Errorf("экранирование: %v", got)
	}
	if got := quote("a'b\\c"); got != `'a\'b\\c'` {
		t.Errorf("quote: %s", got)
	}
	if got := quoteList([]string{"/a/", "/b/"}); got != "['/a/', '/b/']" {
		t.Errorf("quoteList: %s", got)
	}
}

func TestValidateAccel(t *testing.T) {
	for _, ok := range []string{"Print", "<Super>s", "<Control><Alt>p", "F12"} {
		if err := ValidateAccel(ok); err != nil {
			t.Errorf("%q должна быть допустима: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Ctrl+S", "<Super>", "<Bogus>s", "a b"} {
		if ValidateAccel(bad) == nil {
			t.Errorf("%q должна быть отклонена", bad)
		}
	}
}

func TestAssignCreatesNewShortcut(t *testing.T) {
	v := map[string]string{}
	p0 := entry(v, 0, "flameshot", "flameshot gui", "")
	p1 := entry(v, 1, "Handy", "sh -c 'kill -USR2 $(pgrep -x handy)'", "<Control>space")
	v[schema+" custom-keybindings"] = quoteList([]string{p0, p1})
	v["org.gnome.shell.keybindings show-screenshot-ui"] = "['<Shift>Print']"
	f := install(t, v)

	res, err := Assign("Print", "/home/me/.local/bin/scrawl")
	if err != nil || res.Updated || res.Warning != "" {
		t.Fatalf("итог %+v, ошибка %v", res, err)
	}
	newPath := base + "custom2/"
	key := schema + ".custom-keybinding:" + newPath
	if v[key+" name"] != "'scrawl'" || v[key+" command"] != "'/home/me/.local/bin/scrawl'" || v[key+" binding"] != "'Print'" {
		t.Errorf("поля нового ярлыка: %v", v)
	}
	if got := parseStrings(v[schema+" custom-keybindings"]); len(got) != 3 || got[2] != newPath {
		t.Errorf("список ярлыков: %v", got)
	}
	// The list is updated after the fields are filled in.
	if last := f.sets[len(f.sets)-1]; last != schema+" custom-keybindings" {
		t.Errorf("список должен записываться последним: %v", f.sets)
	}
	if Current() != "Print" {
		t.Errorf("Current: %q", Current())
	}
}

func TestAssignUpdatesOwnShortcutIncludingOldName(t *testing.T) {
	v := map[string]string{}
	p0 := entry(v, 0, "screen-editor", "/home/me/bin/screen-editor", "Print")
	v[schema+" custom-keybindings"] = quoteList([]string{p0})
	install(t, v)

	res, err := Assign("<Super>s", "/home/me/.local/bin/scrawl")
	if err != nil || !res.Updated {
		t.Fatalf("итог %+v, ошибка %v", res, err)
	}
	key := schema + ".custom-keybinding:" + p0
	if v[key+" name"] != "'scrawl'" || v[key+" command"] != "'/home/me/.local/bin/scrawl'" || v[key+" binding"] != "'<Super>s'" {
		t.Errorf("ярлык не обновился: %v", v)
	}
	if got := parseStrings(v[schema+" custom-keybindings"]); len(got) != 1 {
		t.Errorf("новых ярлыков быть не должно: %v", got)
	}
}

func TestAssignRefusesForeignConflict(t *testing.T) {
	v := map[string]string{}
	p0 := entry(v, 0, "Handy", "handy", "<Control>space")
	v[schema+" custom-keybindings"] = quoteList([]string{p0})
	f := install(t, v)

	_, err := Assign("<control>space", "/x/scrawl")
	var c *Conflict
	if !errors.As(err, &c) || c.Name != "Handy" {
		t.Fatalf("ожидали конфликт с Handy: %v", err)
	}
	if len(f.sets) != 0 {
		t.Errorf("при конфликте ничего не меняется: %v", f.sets)
	}
	if _, err := Assign("не комбинация", "/x/scrawl"); err == nil {
		t.Error("неверная запись комбинации должна отклоняться")
	}
}

func TestAssignWarnsAboutBuiltin(t *testing.T) {
	v := map[string]string{schema + " custom-keybindings": "@as []", "org.gnome.shell.keybindings show-screenshot-ui": "['Print']"}
	install(t, v)
	res, err := Assign("Print", "/x/scrawl")
	if err != nil || !strings.Contains(res.Warning, "show-screenshot-ui") {
		t.Errorf("итог %+v, ошибка %v", res, err)
	}
	if got := parseStrings(v[schema+" custom-keybindings"]); len(got) != 1 || got[0] != base+"custom0/" {
		t.Errorf("первый ярлык: %v", got)
	}
}

func TestFreeBuiltinRemovesOnlyThatCombination(t *testing.T) {
	v := map[string]string{
		"org.gnome.shell.keybindings show-screenshot-ui":       "['<Shift><Super>s', 'Print']",
		"org.gnome.shell.keybindings screenshot":               "['<Alt>Print']",
		"org.gnome.shell.keybindings show-screen-recording-ui": "['Print']",
	}
	f := install(t, v)
	changed, err := FreeBuiltin("Print")
	if err != nil {
		t.Fatal(err)
	}
	if got := parseStrings(v["org.gnome.shell.keybindings show-screenshot-ui"]); len(got) != 1 || got[0] != "<Shift><Super>s" {
		t.Errorf("only Print must go, the other binding stays: %v", v["org.gnome.shell.keybindings show-screenshot-ui"])
	}
	if got := v["org.gnome.shell.keybindings show-screen-recording-ui"]; len(parseStrings(got)) != 0 {
		t.Errorf("a setting with only Print becomes empty: %q", got)
	}
	if v["org.gnome.shell.keybindings screenshot"] != "['<Alt>Print']" {
		t.Errorf("a different combination must not be touched: %q", v["org.gnome.shell.keybindings screenshot"])
	}
	if len(changed) != 2 || len(f.sets) != 2 {
		t.Errorf("changed %v, writes %v", changed, f.sets)
	}
	// A second call changes nothing.
	f.sets = nil
	if changed, err := FreeBuiltin("Print"); err != nil || len(changed) != 0 || len(f.sets) != 0 {
		t.Errorf("the second call must be a no-op: %v %v %v", changed, err, f.sets)
	}
	if _, err := FreeBuiltin("not a key!"); err == nil {
		t.Error("an invalid combination must be refused")
	}
}

func TestNormalizeAccel(t *testing.T) {
	for in, want := range map[string]string{
		"Ctrl+Alt+P":      "<Control><Alt>p",
		" super + S ":     "<Super>s",
		"Shift+Ctrl+F5":   "<Shift><Control>F5",
		"Print":           "Print",
		"<Control><Alt>p": "<Control><Alt>p", // already in the GNOME form
		"Hyper+x":         "Hyper+x",         // an unknown modifier is left for the validator
		"":                "",
	} {
		if got := NormalizeAccel(in); got != want {
			t.Errorf("NormalizeAccel(%q) = %q, expected %q", in, got, want)
		}
	}
	if err := ValidateAccel(NormalizeAccel("Ctrl+Shift+S")); err != nil {
		t.Errorf("a normalised combination must be valid: %v", err)
	}
}

func TestRemoveDeletesOnlyOurShortcut(t *testing.T) {
	v := map[string]string{}
	p0 := entry(v, 0, "Handy", "handy", "<Control>space")
	p1 := entry(v, 1, "scrawl", "/home/me/.local/bin/scrawl", "Print")
	v[schema+" custom-keybindings"] = quoteList([]string{p0, p1})
	install(t, v)

	n, err := Remove()
	if err != nil || n != 1 {
		t.Fatalf("removed %d, error %v", n, err)
	}
	if got := parseStrings(v[schema+" custom-keybindings"]); len(got) != 1 || got[0] != p0 {
		t.Errorf("only the foreign shortcut stays in the list: %v", got)
	}
	if _, left := v[schema+".custom-keybinding:"+p1+" binding"]; left {
		t.Error("the fields of the removed shortcut must be reset")
	}
	if v[schema+".custom-keybinding:"+p0+" binding"] != "'<Control>space'" {
		t.Error("the foreign shortcut must stay as it was")
	}
	if n, err := Remove(); err != nil || n != 0 {
		t.Errorf("the second call must be a no-op: %d %v", n, err)
	}
}
