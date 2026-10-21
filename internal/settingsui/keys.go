package settingsui

import (
	"fmt"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/keymap"
	"github.com/jexme/scrawl/internal/settings"
)

// keyLabels are the names of the overlay actions in the settings window.
var keyLabels = map[string]string{
	settings.ActionConfirm: "Copy and/or save",
	settings.ActionCancel:  "Close the overlay",
	settings.ActionRestore: "Restore the previous shot's area",
	settings.ActionAdd:     "Add to the collection",
	settings.ActionGlue:    "Glue the collection",
	settings.ActionSend:    "Send the collection as files",
	settings.ActionUndo:    "Undo the last annotation",
	settings.ActionCopy:    "Copy only",
	settings.ActionSave:    "Save only",

	settings.ActionWidthUp:   "Thicker (text size, blur strength)",
	settings.ActionWidthDown: "Thinner (text size, blur strength)",
}

// keyLabel returns the action name for the settings window. For tools it is
// built from the tool's name on the toolbar.
func keyLabel(action string) string {
	for i, name := range settings.ToolNames {
		if settings.ToolAction(name) == action {
			return fmt.Sprintf(i18n.T("Tool “%s”"), i18n.T(toolLabels[i]))
		}
	}
	return i18n.T(keyLabels[action])
}

// keyRow is a window row: an action, a button with its key, and a reset.
type keyRow struct {
	action string
	btn    widget.Clickable // press, then press the new combination
	reset  widget.Clickable
	err    string
}

func (s *state) initKeys() {
	for _, a := range settings.KeyActions {
		s.keyRows = append(s.keyRows, &keyRow{action: a})
	}
}

// assignKey assigns a key to an action. A key already taken by another action is
// not accepted, and the message says whose it became.
func (s *state) assignKey(row *keyRow, chord string) {
	keys := s.store.Get().Keys
	if chord != "" {
		canon, err := settings.ParseChord(chord)
		if err != nil {
			row.err = err.Error()
			return
		}
		chord = canon
	}
	keys.Set(row.action, chord) // empty key: reset to the default
	if a, b, clash := keys.Conflict(); clash {
		other := a
		if a == row.action {
			other = b
		}
		row.err = fmt.Sprintf(i18n.T("%s is already used by \"%s\""), keys.Of(row.action), keyLabel(other))
		return
	}
	row.err = ""
	if err := s.store.Update(func(c *settings.Settings) { c.Keys = keys }); err != nil {
		row.err = i18n.T("Could not save: ") + err.Error()
	}
}

// updateKeys handles button presses and, while recording, the pressed combination.
func (s *state) updateKeys(gtx layout.Context) {
	for _, row := range s.keyRows {
		if row.reset.Clicked(gtx) {
			s.capturing = nil
			s.assignKey(row, "")
		}
		if row.btn.Clicked(gtx) {
			if s.capturing == row {
				s.capturing = nil
				continue
			}
			s.capturing, row.err = row, ""
			gtx.Execute(key.FocusCmd{Tag: &s.capTag})
		}
	}
	if s.capturing == nil {
		return
	}
	for {
		ev, ok := gtx.Event(key.Filter{Focus: &s.capTag, Optional: key.ModCtrl | key.ModShift | key.ModAlt | key.ModSuper})
		if !ok {
			return
		}
		e, ok := ev.(key.Event)
		if !ok {
			continue
		}
		if chord, ok := keymap.Chord(e); ok {
			row := s.capturing
			s.capturing = nil
			s.assignKey(row, chord)
		}
	}
}

// keyControls is the table of overlay keys.
func (s *state) keyControls() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		keys := s.store.Get().Keys
		var children []layout.FlexChild
		for _, row := range s.keyRows {
			row := row
			label := keys.Of(row.action)
			if s.capturing == row {
				label = i18n.T("Press a key…")
			}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, s.label(material.Body2, keyLabel(row.action), fgColor)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(170)
						b := material.Button(s.th, &row.btn, label)
						b.Background = borderColor
						if s.capturing == row {
							b.Background = accentColor
						}
						return b.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						b := material.Button(s.th, &row.reset, i18n.T("Reset"))
						b.Background = borderColor
						return b.Layout(gtx)
					}),
				)
			}))
			if row.err != "" {
				children = append(children, layout.Rigid(s.label(material.Caption, row.err, errorColor)))
			}
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}
