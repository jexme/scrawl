// Package settingsui показывает отдельное окно настроек. Изменения применяются
// сразу и записываются в config.json.
package settingsui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"time"

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

	choice   widget.Enum // двойной клик: off, copy, copy_save
	enter    widget.Enum // Enter: copy_save, copy, save
	notify   widget.Bool // уведомления после снимка
	remember widget.Bool // запоминать прошлую область
	showSize widget.Bool // подпись размера области
	dir      widget.Editor
	prefix   widget.Editor // начало имени файла
	reset    widget.Clickable
	openDir  widget.Clickable
	close    widget.Clickable
	list     widget.List // прокрутка содержимого
	sliders  []*slider   // все ползунки окна
	dim      *slider     // затемнение вне области
	bWidth   *slider     // рамка: толщина
	bDash    *slider     // рамка: штрих
	bGap     *slider     // рамка: пауза
	handle   *slider     // радиус угловых ручек
	bColor   widget.Editor
	colorErr string

	dirErr    string
	prefixErr string
}

func newState(store *settings.Store) *state {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette = material.Palette{Bg: bgColor, Fg: fgColor, ContrastBg: accentColor, ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}

	s := &state{th: th, store: store}
	s.list.Axis = layout.Vertical
	px := func(v float64) string { return fmt.Sprintf("%.1f px", v) }
	s.dim = s.addSlider(0, 100, 1, func(v float64) string { return fmt.Sprintf("%.0f%%", v) },
		func(c settings.Settings) float64 { return float64(c.DimPercent) },
		func(c *settings.Settings, v float64) { c.DimPercent = int(v) })
	s.bWidth = s.addSlider(settings.MinBorderWidth, settings.MaxBorderWidth, 0.5, px,
		func(c settings.Settings) float64 { return c.BorderWidth },
		func(c *settings.Settings, v float64) { c.BorderWidth = v })
	s.bDash = s.addSlider(settings.MinBorderDash, 12, 0.2, px,
		func(c settings.Settings) float64 { return c.BorderDash },
		func(c *settings.Settings, v float64) { c.BorderDash = v })
	s.bGap = s.addSlider(0, 12, 0.5, px,
		func(c settings.Settings) float64 { return c.BorderGap },
		func(c *settings.Settings, v float64) { c.BorderGap = v })
	s.handle = s.addSlider(0, settings.MaxHandleRadius, 0.5, func(v float64) string {
		if v == 0 {
			return "нет"
		}
		return fmt.Sprintf("%.1f px", v)
	},
		func(c settings.Settings) float64 { return c.HandleRadius },
		func(c *settings.Settings, v float64) { c.HandleRadius = v })
	cur := store.Get()
	for _, sl := range s.sliders {
		sl.Set(sl.get(cur))
	}
	s.bColor.SingleLine = true
	s.bColor.SetText(cur.BorderColor)
	s.choice.Value = string(cur.DoubleClick)
	s.enter.Value = string(cur.Enter)
	s.notify.Value = cur.Notifications
	s.remember.Value = cur.RememberRegion
	s.showSize.Value = cur.ShowSize
	s.dir.SingleLine = true
	s.dir.SetText(cur.SaveDir)
	s.prefix.SingleLine = true
	s.prefix.SetText(cur.FilePrefix)
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

	for _, sl := range s.sliders {
		sl.Update(gtx)
		if want := sl.Value(); want != sl.get(s.store.Get()) {
			if err := s.store.Update(func(c *settings.Settings) { sl.set(c, want) }); err != nil {
				s.dirErr = "Не удалось сохранить: " + err.Error()
			}
		}
	}
	if edited(gtx, &s.bColor) {
		s.applyBorderColor(s.bColor.Text())
	}

	s.showSize.Update(gtx)
	if want := s.showSize.Value; want != s.store.Get().ShowSize {
		if err := s.store.Update(func(c *settings.Settings) { c.ShowSize = want }); err != nil {
			s.dirErr = "Не удалось сохранить: " + err.Error()
		}
	}

	s.remember.Update(gtx)
	if want := s.remember.Value; want != s.store.Get().RememberRegion {
		if err := s.store.Update(func(c *settings.Settings) { c.RememberRegion = want }); err != nil {
			s.dirErr = "Не удалось сохранить: " + err.Error()
		}
	}

	if s.openDir.Clicked(gtx) {
		s.openOutputDir()
	}

	changed := edited(gtx, &s.dir)
	if s.reset.Clicked(gtx) {
		s.dir.SetText("")
		changed = true
	}
	if changed {
		s.applyDir(strings.TrimSpace(s.dir.Text()))
	}
	if edited(gtx, &s.prefix) {
		s.applyPrefix(s.prefix.Text())
	}
}

// addSlider создаёт ползунок и записывает его в список окна.
func (s *state) addSlider(min, max, step float64, format func(float64) string,
	get func(settings.Settings) float64, set func(*settings.Settings, float64)) *slider {
	sl := &slider{min: min, max: max, step: step, format: format, get: get, set: set}
	s.sliders = append(s.sliders, sl)
	return sl
}

// applyBorderColor проверяет и сохраняет цвет рамки; при ошибке настройка не меняется.
func (s *state) applyBorderColor(txt string) {
	c, err := settings.ParseColor(txt)
	if err != nil {
		s.colorErr = err.Error()
		return
	}
	s.colorErr = ""
	if err := s.store.Update(func(cfg *settings.Settings) { cfg.BorderColor = settings.FormatColor(c) }); err != nil {
		s.colorErr = "Не удалось сохранить: " + err.Error()
	}
}

// openOutputDir открывает папку скриншотов в файловом менеджере. Пока команда
// запускается, окно не ждёт её завершения.
func (s *state) openOutputDir() {
	dir, err := s.store.Get().OutputDir()
	if err == nil {
		err = openFolder(dir)
	}
	if err != nil {
		s.dirErr = "Не удалось открыть папку: " + err.Error()
		return
	}
	s.dirErr = ""
}

// openFolder запускает xdg-open для папки и не дожидается его завершения.
var openFolder = func(dir string) error {
	cmd := exec.Command("xdg-open", dir)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // чтобы не оставлять зомби-процесс
	return nil
}

// edited сообщает, менялся ли текст в поле с прошлого кадра.
func edited(gtx layout.Context, ed *widget.Editor) bool {
	changed := false
	for {
		ev, ok := ed.Update(gtx)
		if !ok {
			return changed
		}
		if _, ok := ev.(widget.ChangeEvent); ok {
			changed = true
		}
	}
}

// applyPrefix проверяет и сохраняет начало имени файла; при ошибке настройка не меняется.
func (s *state) applyPrefix(prefix string) {
	if err := settings.ValidateFilePrefix(prefix); err != nil {
		s.prefixErr = err.Error()
		return
	}
	s.prefixErr = ""
	if err := s.store.Update(func(c *settings.Settings) { c.FilePrefix = prefix }); err != nil {
		s.prefixErr = "Не удалось сохранить: " + err.Error()
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

// field — однострочное поле ввода в рамке.
func (s *state) field(ed *widget.Editor, hint string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return widget.Border{Color: borderColor, CornerRadius: unit.Dp(6), Width: unit.Dp(1)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					e := material.Editor(s.th, ed, hint)
					e.Color = fgColor
					e.HintColor = mutedColor
					return e.Layout(gtx)
				})
			})
	}
}

// borderControls — ползунки рамки и поле цвета с образцом.
func (s *state) borderControls() layout.Widget {
	labeled := func(name string, sl *slider) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(80)
					return s.label(material.Body2, name, mutedColor)(gtx)
				}),
				layout.Flexed(1, s.slider(sl)),
			)
		})
	}
	return func(gtx layout.Context) layout.Dimensions {
		cfg := s.store.Get()
		swatch := func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Dp(36)
			c := cfg.BorderNRGBA()
			if parsed, err := settings.ParseColor(s.bColor.Text()); err == nil {
				c = parsed
			}
			rr := clip.UniformRRect(image.Rect(0, 0, sz, sz), gtx.Dp(6))
			paint.FillShape(gtx.Ops, borderColor, rr.Op(gtx.Ops))
			inner := clip.UniformRRect(image.Rect(gtx.Dp(2), gtx.Dp(2), sz-gtx.Dp(2), sz-gtx.Dp(2)), gtx.Dp(4))
			paint.FillShape(gtx.Ops, c, inner.Op(gtx.Ops))
			return layout.Dimensions{Size: image.Pt(sz, sz)}
		}
		children := []layout.FlexChild{
			labeled("Толщина", s.bWidth),
			labeled("Штрих", s.bDash),
			labeled("Пауза", s.bGap),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(80)
						return s.label(material.Body2, "Цвет", mutedColor)(gtx)
					}),
					layout.Rigid(swatch),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Flexed(1, s.field(&s.bColor, "#FFFFFF")),
				)
			}),
		}
		if s.colorErr != "" {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, s.colorErr, errorColor)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
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
		children := []layout.FlexChild{layout.Rigid(s.field(&s.dir, "~/Pictures/Screenshots"))}
		if s.dirErr != "" {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, s.dirErr, errorColor)))
		}
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := func(c *widget.Clickable, txt string) layout.Widget {
				b := material.Button(th, c, txt)
				b.Background = borderColor
				return b.Layout
			}
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(btn(&s.reset, "По умолчанию")),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(btn(&s.openDir, "Открыть папку")),
			)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	prefixField := func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.field(&s.prefix, settings.DefaultFilePrefix))}
		if s.prefixErr != "" {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, s.prefixErr, errorColor)))
		}
		current := s.store.Get()
		current.FilePrefix = s.prefix.Text()
		if settings.ValidateFilePrefix(current.FilePrefix) != nil {
			current = s.store.Get()
		}
		example := current.FileName(time.Now().Format(settings.StampLayout))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, "Например: "+example, mutedColor)))
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
		s.row("Прошлая область", "Показывается сразу при запуске и по клавише R. Без неё «было/стало» не подставит область само.",
			s.checkbox(&s.remember, "Запоминать область снимка")),
		s.section("Вид"),
		s.row("Рамка выделения", "Пунктир вокруг выбранной области.", s.borderControls()),
		s.row("Размер области", "Подпись «ширина × высота» над левым верхним углом.",
			s.checkbox(&s.showSize, "Показывать размер")),
		s.row("Угловые ручки", "Кружки в углах рамки, за них меняют размер области. «нет» прячет их.", s.slider(s.handle)),
		s.row("Затемнение вне области", "0% не затемняет совсем, 100% закрашивает чёрным.", s.slider(s.dim)),
		s.section("Сохранение"),
		s.row("Папка для скриншотов", "Пусто: ~/Pictures/Screenshots. Можно писать ~/Папка или полный путь.", dirField),
		s.row("Имя файла", "Начало имени, дальше идут дата и время. Пусто: "+settings.DefaultFilePrefix+".", prefixField),
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
