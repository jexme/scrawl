// Package keymap связывает клавиши из настроек («Ctrl+Z», «Enter») с событиями Gio.
package keymap

import (
	"strings"

	"gioui.org/io/key"
)

// Модификаторы, которые учитываются при сравнении. Остальные (Super) игнорируются.
const mask = key.ModCtrl | key.ModShift | key.ModAlt

// Имена клавиш Gio для названий из настроек. Остальные (буквы, цифры, F1-F12)
// называются в Gio так же, как в настройках.
var named = map[string][]key.Name{
	"Enter":     {key.NameReturn, key.NameEnter},
	"Escape":    {key.NameEscape},
	"Space":     {key.NameSpace},
	"Tab":       {key.NameTab},
	"Backspace": {key.NameDeleteBackward},
	"Delete":    {key.NameDeleteForward},
	"Home":      {key.NameHome},
	"End":       {key.NameEnd},
	"PageUp":    {key.NamePageUp},
	"PageDown":  {key.NamePageDown},
	"Up":        {key.NameUpArrow},
	"Down":      {key.NameDownArrow},
	"Left":      {key.NameLeftArrow},
	"Right":     {key.NameRightArrow},
	// Плюс и минус: Gio называет клавишу символом, который она печатает. «+» это Shift+«=»
	// на обычной раскладке, поэтому Shift у этих двух клавиш не учитывается.
	"Plus":  {"+", "="},
	"Minus": {"-", "_"},
}

// noShift — клавиши, у которых Shift не различается: «+» набирают с Shift, «=» без него.
var noShift = map[string]bool{"Plus": true, "Minus": true}

// Binding — комбинация клавиш в виде, удобном для Gio.
type Binding struct {
	Mods  key.Modifiers
	Names []key.Name // у Enter два имени: основная клавиша и цифровая
	// IgnoreShift: Shift при сравнении не учитывается (плюс и минус).
	IgnoreShift bool
}

// Parse разбирает каноническую запись из настроек («Ctrl+Shift+S»).
func Parse(chord string) (Binding, bool) {
	parts := strings.Split(chord, "+")
	var b Binding
	for _, p := range parts[:len(parts)-1] {
		switch p {
		case "Ctrl":
			b.Mods |= key.ModCtrl
		case "Shift":
			b.Mods |= key.ModShift
		case "Alt":
			b.Mods |= key.ModAlt
		default:
			return Binding{}, false
		}
	}
	name := parts[len(parts)-1]
	if name == "" {
		return Binding{}, false
	}
	if names, ok := named[name]; ok {
		b.Names = names
		b.IgnoreShift = noShift[name]
	} else {
		b.Names = []key.Name{key.Name(name)}
	}
	return b, true
}

// Filters возвращает фильтры событий для gtx.Event. Лишние модификаторы
// пропускаются фильтром, а отсеиваются в Matches: иначе Ctrl+Shift+R сработало бы как R.
func (b Binding) Filters() []key.Filter {
	fs := make([]key.Filter, len(b.Names))
	for i, n := range b.Names {
		fs[i] = key.Filter{Name: n, Required: b.Mods, Optional: mask &^ b.Mods}
	}
	return fs
}

// Matches сообщает, что событие нажатия соответствует комбинации точно.
func (b Binding) Matches(e key.Event) bool {
	if e.State != key.Press {
		return false
	}
	have, want := e.Modifiers&mask, b.Mods
	if b.IgnoreShift {
		have, want = have&^key.ModShift, want&^key.ModShift
	}
	if have != want {
		return false
	}
	for _, n := range b.Names {
		if e.Name == n {
			return true
		}
	}
	return false
}

// Chord возвращает каноническую запись для события нажатия: так окно настроек
// узнаёт, какую комбинацию нажал пользователь. Нажатие одних модификаторов и
// неизвестные клавиши не считаются.
func Chord(e key.Event) (string, bool) {
	if e.State != key.Press {
		return "", false
	}
	var name string
	for n, names := range named {
		for _, gn := range names {
			if gn == e.Name {
				name = n
			}
		}
	}
	if name == "" {
		s := string(e.Name)
		valid := len(s) == 1 && (s[0] >= 'A' && s[0] <= 'Z' || s[0] >= '0' && s[0] <= '9')
		if !valid && !(len(s) >= 2 && len(s) <= 3 && s[0] == 'F' && s[1] >= '1' && s[1] <= '9') {
			return "", false
		}
		name = s
	}
	mods := e.Modifiers
	if noShift[name] {
		mods &^= key.ModShift // Shift+«=» это просто плюс
	}
	var out []string
	if mods.Contain(key.ModCtrl) {
		out = append(out, "Ctrl")
	}
	if mods.Contain(key.ModShift) {
		out = append(out, "Shift")
	}
	if mods.Contain(key.ModAlt) {
		out = append(out, "Alt")
	}
	return strings.Join(append(out, name), "+"), true
}
