package settings

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jexme/scrawl/internal/i18n"
)

// Действия оверлея, которым можно назначить клавиши.
const (
	ActionConfirm = "confirm" // скопировать и/или сохранить (по настройке Enter)
	ActionCancel  = "cancel"  // закрыть оверлей
	ActionRestore = "restore" // вернуть область прошлого снимка
	ActionAdd     = "add"     // добавить в подборку
	ActionGlue    = "glue"    // склеить подборку
	ActionSend    = "send"    // отправить подборку отдельными файлами
	ActionUndo    = "undo"    // отменить последнюю аннотацию
	ActionCopy    = "copy"    // только копировать
	ActionSave    = "save"    // только сохранить

	ActionWidthUp   = "width_up"   // толще: толщина, размер текста или сила размытия
	ActionWidthDown = "width_down" // тоньше

	// Выбор инструмента: действие называется «tool_» и имя из ToolNames.
	ActionToolSelect = "tool_select"
	ActionToolPen    = "tool_pen"
	ActionToolLine   = "tool_line"
	ActionToolArrow  = "tool_arrow"
	ActionToolRect   = "tool_rect"
	ActionToolMarker = "tool_marker"
	ActionToolBlur   = "tool_blur"
	ActionToolText   = "tool_text"
)

// ToolAction возвращает действие, которое включает инструмент с именем из ToolNames.
func ToolAction(tool string) string { return "tool_" + tool }

// KeyActions — все действия в порядке показа в окне настроек.
var KeyActions = []string{
	ActionConfirm, ActionCancel, ActionRestore, ActionAdd, ActionGlue, ActionSend, ActionUndo, ActionCopy, ActionSave,
	ActionWidthUp, ActionWidthDown,
	ActionToolSelect, ActionToolPen, ActionToolLine, ActionToolArrow, ActionToolRect, ActionToolMarker, ActionToolBlur, ActionToolText,
}

// Keys — клавиши действий в виде «Ctrl+Shift+S», «R», «Enter». Пусто: по умолчанию.
type Keys struct {
	Confirm string `json:"confirm,omitempty"`
	Cancel  string `json:"cancel,omitempty"`
	Restore string `json:"restore,omitempty"`
	Add     string `json:"add,omitempty"`
	Glue    string `json:"glue,omitempty"`
	Send    string `json:"send,omitempty"`
	Undo    string `json:"undo,omitempty"`
	Copy    string `json:"copy,omitempty"`
	Save    string `json:"save,omitempty"`

	WidthUp   string `json:"width_up,omitempty"`
	WidthDown string `json:"width_down,omitempty"`

	ToolSelect string `json:"tool_select,omitempty"`
	ToolPen    string `json:"tool_pen,omitempty"`
	ToolLine   string `json:"tool_line,omitempty"`
	ToolArrow  string `json:"tool_arrow,omitempty"`
	ToolRect   string `json:"tool_rect,omitempty"`
	ToolMarker string `json:"tool_marker,omitempty"`
	ToolBlur   string `json:"tool_blur,omitempty"`
	ToolText   string `json:"tool_text,omitempty"`
}

// DefaultKeys — клавиши, с которыми программа поставляется.
var DefaultKeys = map[string]string{
	ActionConfirm: "Enter", ActionCancel: "Escape", ActionRestore: "R", ActionAdd: "A",
	ActionGlue: "G", ActionSend: "D", ActionUndo: "Ctrl+Z", ActionCopy: "Ctrl+C", ActionSave: "Ctrl+S",
	// Плюс и минус: Shift у «+» на обычной раскладке не нужен, программа его игнорирует.
	ActionWidthUp: "Plus", ActionWidthDown: "Minus",
	// Инструменты: W от «arroW», B от «Box», U от «blUr»; A, R, D, G заняты действиями выше.
	ActionToolSelect: "V", ActionToolPen: "P", ActionToolLine: "L", ActionToolArrow: "W",
	ActionToolRect: "B", ActionToolMarker: "M", ActionToolBlur: "U", ActionToolText: "T",
}

func (k *Keys) ptr(action string) *string {
	switch action {
	case ActionConfirm:
		return &k.Confirm
	case ActionCancel:
		return &k.Cancel
	case ActionRestore:
		return &k.Restore
	case ActionAdd:
		return &k.Add
	case ActionGlue:
		return &k.Glue
	case ActionSend:
		return &k.Send
	case ActionUndo:
		return &k.Undo
	case ActionCopy:
		return &k.Copy
	case ActionSave:
		return &k.Save
	case ActionWidthUp:
		return &k.WidthUp
	case ActionWidthDown:
		return &k.WidthDown
	case ActionToolSelect:
		return &k.ToolSelect
	case ActionToolPen:
		return &k.ToolPen
	case ActionToolLine:
		return &k.ToolLine
	case ActionToolArrow:
		return &k.ToolArrow
	case ActionToolRect:
		return &k.ToolRect
	case ActionToolMarker:
		return &k.ToolMarker
	case ActionToolBlur:
		return &k.ToolBlur
	case ActionToolText:
		return &k.ToolText
	}
	return nil
}

// Of возвращает клавишу действия: заданную или по умолчанию.
func (k Keys) Of(action string) string {
	if p := k.ptr(action); p != nil && *p != "" {
		return *p
	}
	return DefaultKeys[action]
}

// Set назначает клавишу действию; значение, равное умолчанию, хранится пустым.
func (k *Keys) Set(action, chord string) {
	if p := k.ptr(action); p != nil {
		if chord == DefaultKeys[action] {
			chord = ""
		}
		*p = chord
	}
}

// Допустимые клавиши кроме букв, цифр и F1-F12.
var namedKeys = []string{"Enter", "Escape", "Space", "Tab", "Backspace", "Delete", "Home", "End", "PageUp", "PageDown", "Up", "Down", "Left", "Right", "Plus", "Minus"}

func validKeyName(k string) bool {
	if len(k) == 1 && (k[0] >= 'A' && k[0] <= 'Z' || k[0] >= '0' && k[0] <= '9') {
		return true
	}
	for _, n := range namedKeys {
		if k == n {
			return true
		}
	}
	var n int
	if _, err := fmt.Sscanf(k, "F%d", &n); err == nil && n >= 1 && n <= 12 && k == fmt.Sprintf("F%d", n) {
		return true
	}
	return false
}

// ParseChord разбирает комбинацию («ctrl+shift+s», «Enter») и возвращает её в
// каноническом виде: модификаторы Ctrl, Shift, Alt по порядку, клавиша с большой буквы.
func ParseChord(s string) (string, error) {
	parts := strings.Split(strings.TrimSpace(s), "+")
	var ctrl, shift, alt bool
	for _, p := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "ctrl", "control":
			ctrl = true
		case "shift":
			shift = true
		case "alt":
			alt = true
		default:
			return "", fmt.Errorf(i18n.T("неизвестный модификатор «%s»: можно Ctrl, Shift, Alt"), strings.TrimSpace(p))
		}
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	if key == "" {
		return "", errors.New(i18n.T("не указана клавиша"))
	}
	canon := key
	if len(key) == 1 {
		canon = strings.ToUpper(key)
	} else {
		for _, n := range namedKeys {
			if strings.EqualFold(key, n) {
				canon = n
			}
		}
		if strings.HasPrefix(strings.ToLower(key), "f") {
			canon = strings.ToUpper(key)
		}
	}
	if !validKeyName(canon) {
		return "", fmt.Errorf(i18n.T("неизвестная клавиша «%s»: буква, цифра, F1-F12, Enter, Escape, Space, Tab, Backspace, Delete, Home, End, PageUp, PageDown, стрелки (Up, Down, Left, Right)"), key)
	}
	var out []string
	if ctrl {
		out = append(out, "Ctrl")
	}
	if shift {
		out = append(out, "Shift")
	}
	if alt {
		out = append(out, "Alt")
	}
	return strings.Join(append(out, canon), "+"), nil
}

// normalize заменяет неверные клавиши умолчаниями. Если после этого две клавиши
// совпали, все клавиши возвращаются к умолчаниям: неясно, какое действие главнее.
func (k *Keys) normalize() {
	for _, a := range KeyActions {
		p := k.ptr(a)
		if *p == "" {
			continue
		}
		canon, err := ParseChord(*p)
		if err != nil || canon == DefaultKeys[a] {
			canon = ""
		}
		*p = canon
	}
	if _, _, clash := k.Conflict(); clash {
		*k = Keys{}
	}
}

// Conflict находит два действия с одной клавишей.
func (k Keys) Conflict() (a, b string, clash bool) {
	seen := map[string]string{}
	for _, action := range KeyActions {
		chord := k.Of(action)
		if other, ok := seen[chord]; ok {
			return other, action, true
		}
		seen[chord] = action
	}
	return "", "", false
}
