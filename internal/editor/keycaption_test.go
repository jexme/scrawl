package editor

import "testing"

func TestKeyCaption(t *testing.T) {
	for chord, want := range map[string]string{
		"A":            "A",
		"Escape":       "Esc",
		"Enter":        "Enter",
		"Ctrl+Z":       "Ctrl+Z",
		"Ctrl+S":       "Ctrl+S",
		"Ctrl+Shift+S": "^Shift+S",
		"Backspace":    "Bksp",
		"PageDown":     "PgDn",
		"Ctrl+Delete":  "^Del",
		"F5":           "F5",
		"Left":         "←",
		"":             "",
	} {
		if got := keyCaption(chord); got != want {
			t.Errorf("%q: %q, ожидали %q", chord, got, want)
		}
	}
}
