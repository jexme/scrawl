package editor

import (
	"strings"
	"unicode/utf8"
)

// captionLen — длиннее этого подпись под кнопкой не помещается, и «Ctrl+» сокращается до «^».
const captionLen = 6

// keyNames — короткие названия клавиш для подписи под кнопкой панели.
var keyNames = map[string]string{
	"Escape": "Esc", "Backspace": "Bksp", "Delete": "Del", "PageUp": "PgUp", "PageDown": "PgDn",
	"Up": "↑", "Down": "↓", "Left": "←", "Right": "→", "Plus": "+", "Minus": "-",
}

// keyCaption превращает сочетание из настроек («Ctrl+Z», «Escape») в короткую подпись
// под кнопкой: «Ctrl+Z», «Esc». Не умещается, «Ctrl+» сокращается до «^».
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
