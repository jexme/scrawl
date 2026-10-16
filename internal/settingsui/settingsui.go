// Package settingsui показывает отдельное окно настроек. Изменения применяются
// сразу и записываются в config.json.
package settingsui

import (
	"image"
	"image/color"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/screen-editor/internal/screencast"
	"github.com/jexme/screen-editor/internal/settings"
	"github.com/jexme/screen-editor/internal/version"
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
	enter  widget.Enum // Enter: copy_save, copy, save
	notify widget.Bool // уведомления после снимка
	dir    widget.Editor
	reset  widget.Clickable
	close  widget.Clickable
	list   widget.List // прокрутка содержимого

	dirErr string
}

func newState(store *settings.Store) *state {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette = material.Palette{Bg: bgColor, Fg: fgColor, ContrastBg: accentColor, ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}

	s := &state{th: th, store: store}
	s.list.Axis = layout.Vertical
	cur := store.Get()
	s.choice.Value = string(cur.DoubleClick)
	s.enter.Value = string(cur.Enter)
	s.notify.Value = cur.Notifications
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
	w.Option(app.Title("Screen Editor: настройки"), app.Size(unit.Dp(600), unit.Dp(600)), app.MinSize(unit.Dp(420), unit.Dp(300)))
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

	s.enter.Update(gtx)
	if want := settings.Enter(s.enter.Value); want != s.store.Get().Enter {
		if err := s.store.Update(func(c *settings.Settings) { c.Enter = want }); err != nil {
			s.dirErr = "Не удалось сохранить: " + err.Error()
		}
	}

	s.notify.Update(gtx)
	if want := s.notify.Value; want != s.store.Get().Notifications {
		if err := s.store.Update(func(c *settings.Settings) { c.Notifications = want }); err != nil {
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

// row — одна настройка: название, пояснение и элемент управления.
func (s *state) row(name, hint string, control layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.label(material.Body1, name, fgColor))}
		if hint != "" {
			children = append(children, layout.Rigid(s.label(material.Caption, hint, mutedColor)))
		}
		if control != nil {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout), layout.Rigid(control))
		}
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// section — заголовок раздела с линией под ним.
func (s *state) section(name string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.label(material.Subtitle1, name, accentColor)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(1)
				paint.FillShape(gtx.Ops, borderColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, h)}.Op())
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		)
	}
}

// label делает текстовый виджет заданного стиля и цвета.
func (s *state) label(style func(*material.Theme, string) material.LabelStyle, txt string, c color.NRGBA) layout.Widget {
	l := style(s.th, txt)
	l.Color = c
	return l.Layout
}

func (s *state) radio(group *widget.Enum, key, label string) layout.Widget {
	rb := material.RadioButton(s.th, group, key, label)
	rb.Color = fgColor
	// Выбранный вариант синий, остальные светло-серые: на тёмном фоне иначе не видно.
	rb.IconColor = mutedColor
	if group.Value == key {
		rb.IconColor = accentColor
	}
	return rb.Layout
}

func (s *state) checkbox(b *widget.Bool, label string) layout.Widget {
	cb := material.CheckBox(s.th, b, label)
	cb.Color = fgColor
	cb.IconColor = mutedColor
	if b.Value {
		cb.IconColor = accentColor
	}
	return cb.Layout
}

// radios складывает варианты выбора в столбик.
func radios(items ...layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, len(items))
		for i, w := range items {
			children[i] = layout.Rigid(w)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// items — содержимое прокручиваемой части окна: разделы и настройки по порядку.
func (s *state) items() []layout.Widget {
	th := s.th

	dirField := func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
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
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, s.dirErr, errorColor)))
		}
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Button(th, &s.reset, "По умолчанию")
			b.Background = borderColor
			return b.Layout(gtx)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	return []layout.Widget{
		s.section("Действия"),
		s.row("Двойной клик внутри выделения", "Работает с инструментом «выделение», не на рамке.",
			radios(
				s.radio(&s.choice, string(settings.DoubleClickOff), "Ничего"),
				s.radio(&s.choice, string(settings.DoubleClickCopy), "Копировать в буфер обмена"),
				s.radio(&s.choice, string(settings.DoubleClickCopySave), "Копировать и сохранить в файл"),
			)),
		s.row("Клавиша Enter", "Ctrl+C, Ctrl+S и кнопки на панели от этой настройки не зависят.",
			radios(
				s.radio(&s.enter, string(settings.EnterCopySave), "Копировать и сохранить в файл"),
				s.radio(&s.enter, string(settings.EnterCopy), "Только копировать в буфер обмена"),
				s.radio(&s.enter, string(settings.EnterSave), "Только сохранить в файл"),
			)),
		s.row("Уведомления", "После снимка и действий с подборкой. Ошибки показываются всегда.",
			s.checkbox(&s.notify, "Показывать уведомления")),
		s.section("Сохранение"),
		s.row("Папка для скриншотов", "Пусто: ~/Pictures/Screenshots. Можно писать ~/Папка или полный путь.", dirField),
		s.section("О программе"),
		s.row("Screen Editor "+version.String(), version.Home, nil),
	}
}

func (s *state) layout(gtx layout.Context) layout.Dimensions {
	s.apply(gtx)
	th := s.th
	items := s.items()

	return layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.label(material.H6, "Настройки", fgColor)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				list := material.List(th, &s.list)
				return list.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
					// Справа место под полосу прокрутки, чтобы она не лежала на элементах.
					return layout.Inset{Right: unit.Dp(14)}.Layout(gtx, items[i])
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(s.label(material.Caption, "Изменения применяются сразу. Файл: "+settings.Path(), mutedColor)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := material.Button(th, &s.close, "Закрыть")
				b.Background = accentColor
				return b.Layout(gtx)
			}),
		)
	})
}
