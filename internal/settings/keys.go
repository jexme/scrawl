package settings

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jexme/scrawl/internal/i18n"
)

// Overlay actions that can be assigned keys.
const (
	ActionConfirm = "confirm" // copy and/or save (per the Enter setting)
	ActionCancel  = "cancel"  // close the overlay
	ActionRestore = "restore" // restore the previous capture's area
	ActionAdd     = "add"     // add to the collection
	ActionGlue    = "glue"    // glue the collection
	ActionSend    = "send"    // send the collection as separate files
	ActionUndo    = "undo"    // undo the last annotation
	ActionCopy    = "copy"    // copy only
	ActionSave    = "save"    // save only

	ActionWidthUp   = "width_up"   // thicker: thickness, text size or blur strength
	ActionWidthDown = "width_down" // thinner

	// Tool selection: the action is named "tool_" plus a name from ToolNames.
	ActionToolSelect = "tool_select"
	ActionToolPen    = "tool_pen"
	ActionToolLine   = "tool_line"
	ActionToolArrow  = "tool_arrow"
	ActionToolRect   = "tool_rect"
	ActionToolMarker = "tool_marker"
	ActionToolBlur   = "tool_blur"
	ActionToolText   = "tool_text"
	ActionToolStamp  = "tool_stamp"
)

// ToolAction returns the action that activates the tool with a name from ToolNames.
func ToolAction(tool string) string { return "tool_" + tool }

// KeyActions are all the actions in the order they are shown in the settings window.
var KeyActions = []string{
	ActionConfirm, ActionCancel, ActionRestore, ActionAdd, ActionGlue, ActionSend, ActionUndo, ActionCopy, ActionSave,
	ActionWidthUp, ActionWidthDown,
	ActionToolSelect, ActionToolPen, ActionToolLine, ActionToolArrow, ActionToolRect, ActionToolMarker, ActionToolBlur, ActionToolText, ActionToolStamp,
}

// Keys are the action keys written like "Ctrl+Shift+S", "R", "Enter". Empty: the default.
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
	ToolStamp  string `json:"tool_stamp,omitempty"`
}

// DefaultKeys are the keys the program ships with.
var DefaultKeys = map[string]string{
	ActionConfirm: "Enter", ActionCancel: "Escape", ActionRestore: "R", ActionAdd: "A",
	ActionGlue: "G", ActionSend: "D", ActionUndo: "Ctrl+Z", ActionCopy: "Ctrl+C", ActionSave: "Ctrl+S",
	// Plus and minus: Shift for "+" is not needed on a regular layout, the program ignores it.
	ActionWidthUp: "Plus", ActionWidthDown: "Minus",
	// Tools: W from "arroW", B from "Box", U from "blUr"; A, R, D, G are taken by the actions
	// above.
	ActionToolSelect: "V", ActionToolPen: "P", ActionToolLine: "L", ActionToolArrow: "W",
	ActionToolRect: "B", ActionToolMarker: "M", ActionToolBlur: "U", ActionToolText: "T", ActionToolStamp: "N",
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
	case ActionToolStamp:
		return &k.ToolStamp
	}
	return nil
}

// Of returns the key of an action: the assigned one or the default.
func (k Keys) Of(action string) string {
	if p := k.ptr(action); p != nil && *p != "" {
		return *p
	}
	return DefaultKeys[action]
}

// Set assigns a key to an action; a value equal to the default is stored as empty.
func (k *Keys) Set(action, chord string) {
	if p := k.ptr(action); p != nil {
		if chord == DefaultKeys[action] {
			chord = ""
		}
		*p = chord
	}
}

// Allowed keys besides letters, digits and F1-F12.
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

// ParseChord parses a key combination ("ctrl+shift+s", "Enter") and returns it in
// canonical form: modifiers Ctrl, Shift, Alt in order, the key capitalized.
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
			return "", fmt.Errorf(i18n.T("unknown modifier \"%s\": use Ctrl, Shift, Alt"), strings.TrimSpace(p))
		}
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	if key == "" {
		return "", errors.New(i18n.T("no key given"))
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
		return "", fmt.Errorf(i18n.T("unknown key \"%s\": a letter, a digit, F1-F12, Enter, Escape, Space, Tab, Backspace, Delete, Home, End, PageUp, PageDown, arrows (Up, Down, Left, Right)"), key)
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

// normalize replaces invalid keys with the defaults. If two keys then coincide,
// all keys are reset to the defaults: it is unclear which action takes priority.
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

// Conflict finds two actions with the same key.
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
