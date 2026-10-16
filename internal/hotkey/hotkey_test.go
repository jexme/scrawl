package hotkey

import (
	"errors"
	"strings"
	"testing"
)

// fakeGSettings имитирует хранилище gsettings: ключ «схема ключ» -> значение GVariant.
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
	old := gsettings
	gsettings = f.run
	t.Cleanup(func() { gsettings = old })
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
	// Список обновляется после заполнения полей.
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
