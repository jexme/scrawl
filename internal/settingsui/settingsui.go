// Package settingsui показывает отдельное окно настроек. Изменения применяются
// сразу и записываются в config.json.
package settingsui

import (
	"image/color"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/screen-editor/internal/screencast"
	"github.com/jexme/screen-editor/internal/settings"
)

var (
	bgColor     = color.NRGBA{R: 32, G: 33, B: 36, A: 255}
	fgColor     = color.NRGBA{R: 235, G: 235, B: 235, A: 255}
	mutedColor  = color.NRGBA{R: 160, G: 163, B: 170, A: 255}
	accentColor = color.NRGBA{R: 66, G: 133, B: 244, A: 255}
	errorColor  = color.NRGBA{R: 242, G: 139, B: 130, A: 255}
	borderColor = color.NRGBA{R: 74, G: 75, B: 79, A: 255}
)

// state — состояние окна: виджеты и ссылка на настройки.
type state struct {
	th    *material.Theme
	store *settings.Store

	choice widget.Enum // двойной клик: off, copy, copy_save
	dir    widget.Editor
	reset  widget.Clickable
	close  widget.Clickable

	dirErr string
}

func newState(store *settings.Store) *state {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette = material.Palette{Bg: bgColor, Fg: fgColor, ContrastBg: accentColor, ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}

	s := &state{th: th, store: store}
	cur := store.Get()
	s.choice.Value = string(cur.DoubleClick)
	s.dir.SingleLine = true
	s.dir.SetText(cur.SaveDir)
	return s
}

// Run открывает окно настроек и работает, пока его не закроют. Вызывать из main.
func Run(store *settings.Store) {
	app.ID = screencast.AppID
	go func() {
		loop(newState(store))
		os.Exit(0)
	}()
	app.Main()
}

func loop(s *state) {
	w := new(app.Window)
	w.Option(app.Title("Screen Editor: настройки"), app.Size(unit.Dp(560), unit.Dp(520)))
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			for {
				ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
				if !ok {
					break
				}
				if k, ok := ev.(key.Event); ok && k.State == key.Press {
					w.Perform(system.ActionClose)
				}
			}
			if s.close.Clicked(gtx) {
				w.Perform(system.ActionClose)
			}
			paint.Fill(gtx.Ops, bgColor)
			s.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// apply приводит сохранённые настройки в соответствие с виджетами. Вызывается
// на каждом кадре: ничего не пишет, пока значения не изменились.
func (s *state) apply(gtx layout.Context) {
	s.choice.Update(gtx)
	if want := settings.DoubleClick(s.choice.Value); want != s.store.Get().DoubleClick {
		if err := s.store.Update(func(c *settings.Settings) { c.DoubleClick = want }); err != nil {
			s.dirErr = "Не удалось сохранить: " + err.Error()
		}
	}

	changed := false
	for {
		ev, ok := s.dir.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.ChangeEvent); ok {
			changed = true
		}
	}
	if s.reset.Clicked(gtx) {
		s.dir.SetText("")
		changed = true
	}
	if changed {
		s.applyDir(strings.TrimSpace(s.dir.Text()))
	}
}

// applyDir проверяет и сохраняет папку; при ошибке настройка не меняется.
func (s *state) applyDir(dir string) {
	if err := settings.ValidateSaveDir(dir); err != nil {
		s.dirErr = err.Error()
		return
	}
	s.dirErr = ""
	if err := s.store.Update(func(c *settings.Settings) { c.SaveDir = dir }); err != nil {
		s.dirErr = "Не удалось сохранить: " + err.Error()
	}
}

func (s *state) layout(gtx layout.Context) layout.Dimensions {
	s.apply(gtx)
	th := s.th

	heading := material.H6(th, "Настройки")
	heading.Color = fgColor
	muted := func(txt string) layout.Widget {
		l := material.Caption(th, txt)
		l.Color = mutedColor
		return l.Layout
	}
	title := func(txt string) layout.Widget {
		l := material.Body1(th, txt)
		l.Color = fgColor
		return l.Layout
	}
	gap := func(dp int) layout.FlexChild {
		return layout.Rigid(layout.Spacer{Height: unit.Dp(dp)}.Layout)
	}
	radio := func(key, label string) layout.FlexChild {
		rb := material.RadioButton(th, &s.choice, key, label)
		rb.Color = fgColor
		// Выбранный вариант синий, остальные светло-серые: на тёмном фоне иначе не видно.
		rb.IconColor = mutedColor
		if s.choice.Value == key {
			rb.IconColor = accentColor
		}
		return layout.Rigid(rb.Layout)
	}

	children := []layout.FlexChild{
		layout.Rigid(heading.Layout),
		gap(18),
		layout.Rigid(title("Двойной клик внутри выделения")),
		layout.Rigid(muted("Работает с инструментом «выделение», не на рамке.")),
		gap(6),
		radio(string(settings.DoubleClickOff), "Ничего"),
		radio(string(settings.DoubleClickCopy), "Копировать в буфер обмена"),
		radio(string(settings.DoubleClickCopySave), "Копировать и сохранить в файл"),
		gap(22),
		layout.Rigid(title("Папка для скриншотов")),
		layout.Rigid(muted("Пусто: ~/Pictures/Screenshots. Можно писать ~/Папка или полный путь.")),
		gap(8),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widget.Border{Color: borderColor, CornerRadius: unit.Dp(6), Width: unit.Dp(1)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(th, &s.dir, "~/Pictures/Screenshots")
						ed.Color = fgColor
						ed.HintColor = mutedColor
						return ed.Layout(gtx)
					})
				})
		}),
	}
	if s.dirErr != "" {
		errLabel := material.Caption(th, s.dirErr)
		errLabel.Color = errorColor
		children = append(children, gap(6), layout.Rigid(errLabel.Layout))
	}
	children = append(children,
		gap(8),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Button(th, &s.reset, "По умолчанию")
			b.Background = borderColor
			return b.Layout(gtx)
		}),
		layout.Flexed(1, layout.Spacer{}.Layout),
		layout.Rigid(muted("Изменения применяются сразу. Файл: "+settings.Path())),
		gap(10),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Button(th, &s.close, "Закрыть")
			b.Background = accentColor
			return b.Layout(gtx)
		}),
	)

	return layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}
