// Package keymap maps the keys from the settings ("Ctrl+Z", "Enter") to Gio events.
package keymap

import (
	"strings"

	"gioui.org/io/key"
)

// Modifiers that are taken into account when comparing. The rest (Super) are ignored.
const mask = key.ModCtrl | key.ModShift | key.ModAlt

// Gio key names for names from the settings. The rest (letters, digits, F1-F12)
// are named in Gio the same way as in the settings.
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
	// Plus and minus: Gio names a key by the character it prints. "+" is Shift+"="
	// on a regular layout, so Shift is ignored for these two keys.
	"Plus":  {"+", "="},
	"Minus": {"-", "_"},
}

// noShift lists keys for which Shift is not distinguished: "+" is typed with Shift, "="
// without it.
var noShift = map[string]bool{"Plus": true, "Minus": true}

// Binding is a key combination in a form convenient for Gio.
type Binding struct {
	Mods  key.Modifiers
	Names []key.Name // Enter has two names: the main key and the numpad one
	// IgnoreShift: Shift is ignored when comparing (plus and minus).
	IgnoreShift bool
}

// Parse parses the canonical form from the settings ("Ctrl+Shift+S").
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

// Filters returns event filters for gtx.Event. Extra modifiers are let through by the
// filter and rejected in Matches: otherwise Ctrl+Shift+R would act as R.
func (b Binding) Filters() []key.Filter {
	fs := make([]key.Filter, len(b.Names))
	for i, n := range b.Names {
		fs[i] = key.Filter{Name: n, Required: b.Mods, Optional: mask &^ b.Mods}
	}
	return fs
}

// Matches reports that the key press event matches the combination exactly.
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

// Chord returns the canonical form for a key press event: this is how the settings window
// learns which combination the user pressed. Pressing modifiers alone and
// unknown keys do not count.
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
		mods &^= key.ModShift // Shift+"=" is just a plus
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
