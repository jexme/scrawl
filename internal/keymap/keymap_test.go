package keymap

import (
	"testing"

	"gioui.org/io/key"

	"github.com/jexme/scrawl/internal/settings"
)

func press(name key.Name, mods key.Modifiers) key.Event {
	return key.Event{Name: name, Modifiers: mods, State: key.Press}
}

func TestParseAndMatch(t *testing.T) {
	undo, ok := Parse("Ctrl+Z")
	if !ok || undo.Mods != key.ModCtrl || len(undo.Names) != 1 || undo.Names[0] != "Z" {
		t.Fatalf("Ctrl+Z: %+v %v", undo, ok)
	}
	if !undo.Matches(press("Z", key.ModCtrl)) {
		t.Error("Ctrl+Z должно совпасть")
	}
	if undo.Matches(press("Z", key.ModCtrl|key.ModShift)) || undo.Matches(press("Z", 0)) {
		t.Error("лишний или недостающий модификатор не должен совпадать")
	}
	if undo.Matches(key.Event{Name: "Z", Modifiers: key.ModCtrl, State: key.Release}) {
		t.Error("отпускание клавиши не считается")
	}

	r, _ := Parse("R")
	if r.Matches(press("R", key.ModCtrl)) || !r.Matches(press("R", 0)) {
		t.Error("простая клавиша R без модификаторов")
	}
	if !r.Matches(press("R", key.ModSuper)) {
		t.Error("Super игнорируется при сравнении")
	}

	enter, _ := Parse("Enter")
	if !enter.Matches(press(key.NameReturn, 0)) || !enter.Matches(press(key.NameEnter, 0)) {
		t.Error("Enter это и основная, и цифровая клавиша")
	}
	if fs := enter.Filters(); len(fs) != 2 || fs[0].Optional&key.ModCtrl == 0 {
		t.Errorf("фильтры Enter: %+v", fs)
	}
	if fs := undo.Filters(); fs[0].Required != key.ModCtrl || fs[0].Optional&key.ModCtrl != 0 || fs[0].Optional&key.ModShift == 0 {
		t.Errorf("обязательные и необязательные модификаторы: %+v", fs[0])
	}

	for _, bad := range []string{"Super+S", "Ctrl+"} {
		if _, ok := Parse(bad); ok {
			t.Errorf("Parse(%q) должен отказать", bad)
		}
	}
}

func TestPlusMinus(t *testing.T) {
	plus, ok := Parse("Plus")
	if !ok || !plus.IgnoreShift {
		t.Fatalf("Plus: %+v %v", plus, ok)
	}
	for _, e := range []key.Event{press("+", 0), press("+", key.ModShift), press("=", 0), press("=", key.ModShift)} {
		if !plus.Matches(e) {
			t.Errorf("Plus должен совпасть с %+v", e)
		}
	}
	if plus.Matches(press("+", key.ModCtrl)) || plus.Matches(press("-", 0)) {
		t.Error("Ctrl+«+» и «-» не плюс")
	}
	minus, _ := Parse("Minus")
	if !minus.Matches(press("-", 0)) || !minus.Matches(press("_", key.ModShift)) || minus.Matches(press("=", 0)) {
		t.Error("Minus совпадает с «-» и «_», но не с «=»")
	}
	ctrlPlus, _ := Parse("Ctrl+Plus")
	if !ctrlPlus.Matches(press("+", key.ModCtrl|key.ModShift)) || ctrlPlus.Matches(press("+", 0)) {
		t.Error("Ctrl+Plus: Ctrl обязателен, Shift нет")
	}
}

func TestChordFromEvent(t *testing.T) {
	cases := []struct {
		e    key.Event
		want string
		ok   bool
	}{
		{press("R", 0), "R", true},
		{press("S", key.ModCtrl|key.ModShift), "Ctrl+Shift+S", true},
		{press(key.NameReturn, 0), "Enter", true},
		{press(key.NameEnter, 0), "Enter", true},
		{press(key.NameEscape, 0), "Escape", true},
		{press(key.NameDeleteForward, key.ModAlt), "Alt+Delete", true},
		{press("F5", 0), "F5", true},
		{press("F12", key.ModCtrl), "Ctrl+F12", true},
		{press(key.NameCtrl, key.ModCtrl), "", false}, // only a modifier was pressed
		{press("+", key.ModShift), "Plus", true},      // "+" is typed with Shift: it is just a plus
		{press("=", 0), "Plus", true},
		{press("+", 0), "Plus", true}, // numpad
		{press("-", 0), "Minus", true},
		{press("_", key.ModShift), "Minus", true},
		{press("+", key.ModCtrl|key.ModShift), "Ctrl+Plus", true},
		{press("*", 0), "", false},
		{key.Event{Name: "R", State: key.Release}, "", false},
	}
	for _, c := range cases {
		got, ok := Chord(c.e)
		if got != c.want || ok != c.ok {
			t.Errorf("%+v: получили %q, %v; ожидали %q, %v", c.e, got, ok, c.want, c.ok)
		}
	}
}

// Everything the settings accept must parse here, and vice versa: an entry
// from a Gio event must pass the settings validation.
func TestAgreesWithSettings(t *testing.T) {
	for action, chord := range settings.DefaultKeys {
		b, ok := Parse(chord)
		if !ok || len(b.Names) == 0 {
			t.Errorf("%s: %q не разбирается", action, chord)
			continue
		}
		back, ok := Chord(key.Event{Name: b.Names[0], Modifiers: b.Mods, State: key.Press})
		if !ok || back != chord {
			t.Errorf("%s: %q -> %q", action, chord, back)
		}
		if canon, err := settings.ParseChord(back); err != nil || canon != chord {
			t.Errorf("%s: настройки не принимают %q: %v", action, back, err)
		}
	}
	for _, name := range []string{"Enter", "Escape", "Space", "Tab", "Backspace", "Delete", "Home", "End", "PageUp", "PageDown", "Up", "Down", "Left", "Right", "Plus", "Minus", "F1", "F9", "F10", "F12", "A", "0"} {
		b, ok := Parse(name)
		if !ok {
			t.Errorf("%s не разбирается", name)
			continue
		}
		got, ok := Chord(key.Event{Name: b.Names[0], State: key.Press})
		if !ok || got != name {
			t.Errorf("%s: круг замкнулся на %q (%v)", name, got, ok)
		}
	}
}
