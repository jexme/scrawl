package editor

import (
	"strings"
	"unicode/utf8"
)

// captionLen: a caption longer than this does not fit under the button, and "Ctrl+" is
// shortened to "^".
const captionLen = 6

// keyNames are short key names for the caption under a toolbar button.
var keyNames = map[string]string{
	"Escape": "Esc", "Backspace": "Bksp", "Delete": "Del", "PageUp": "PgUp", "PageDown": "PgDn",
	"Up": "↑", "Down": "↓", "Left": "←", "Right": "→", "Plus": "+", "Minus": "-",
}

// keyCaption turns a combination from the settings ("Ctrl+Z", "Escape") into a short caption
// under the button: "Ctrl+Z", "Esc". If it does not fit, "Ctrl+" is shortened to "^".
func keyCaption(chord string) string {
	parts := strings.Split(chord, "+")
	for i, p := range parts {
		if short, ok := keyNames[p]; ok {
			parts[i] = short
		}
	}
	s := strings.Join(parts, "+")
	if utf8.RuneCountInString(s) > captionLen {
		s = strings.ReplaceAll(s, "Ctrl+", "^")
	}
	return s
}
