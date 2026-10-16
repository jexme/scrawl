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

// keyLabels — названия действий оверлея в окне настроек.
var keyLabels = map[string]string{
	settings.ActionConfirm: "Копировать и/или сохранить",
	settings.ActionCancel:  "Закрыть оверлей",
	settings.ActionRestore: "Вернуть область прошлого снимка",
	settings.ActionAdd:     "Добавить в подборку",
	settings.ActionGlue:    "Склеить подборку",
	settings.ActionSend:    "Отправить подборку файлами",
	settings.ActionUndo:    "Отменить последнюю аннотацию",
	settings.ActionCopy:    "Только копировать",
	settings.ActionSave:    "Только сохранить",
}

// keyRow — строка окна: действие, кнопка с его клавишей и сброс.
type keyRow struct {
	action string
	btn    widget.Clickable // нажать и затем нажать новую комбинацию
	reset  widget.Clickable
	err    string
}

func (s *state) initKeys() {
	for _, a := range settings.KeyActions {
		s.keyRows = append(s.keyRows, &keyRow{action: a})
	}
}

// assignKey назначает действию клавишу. Клавишу, уже занятую другим действием,
// не принимает и говорит, чьей она стала.
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
	keys.Set(row.action, chord) // пустая клавиша: сброс к умолчанию
	if a, b, clash := keys.Conflict(); clash {
		other := a
		if a == row.action {
			other = b
		}
		row.err = fmt.Sprintf(i18n.T("%s уже занята действием «%s»"), keys.Of(row.action), i18n.T(keyLabels[other]))
		return
	}
	row.err = ""
	if err := s.store.Update(func(c *settings.Settings) { c.Keys = keys }); err != nil {
		row.err = i18n.T("Не удалось сохранить: ") + err.Error()
	}
}

// updateKeys обрабатывает нажатия кнопок и, пока идёт запись, нажатую комбинацию.
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

// keyControls — таблица клавиш оверлея.
func (s *state) keyControls() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		keys := s.store.Get().Keys
		var children []layout.FlexChild
		for _, row := range s.keyRows {
			row := row
			label := keys.Of(row.action)
			if s.capturing == row {
				label = i18n.T("Нажмите клавишу…")
			}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, s.label(material.Body2, i18n.T(keyLabels[row.action]), fgColor)),
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
						b := material.Button(s.th, &row.reset, i18n.T("Сброс"))
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
