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

	"github.com/jexme/scrawl/internal/fonts"
	"github.com/jexme/scrawl/internal/hotkey"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
	"github.com/jexme/scrawl/internal/version"
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
	keepOpen widget.Bool // не закрывать оверлей после копирования
	cursor   widget.Bool // курсор мыши в снимке
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
	marker   *slider     // непрозрачность маркера

	palEd    [settings.PaletteSize]widget.Editor // цвета панели
	palErr   string
	palReset widget.Clickable

	glueLayout widget.Enum      // расположение склейки: side, stack
	format     widget.Enum      // формат файла: png, jpeg
	quality    *slider          // качество JPEG
	glueBg     widget.Enum      // фон склейки: light, dark
	gluePad    *slider          // отступы склейки
	glueFont   *slider          // размер подписи склейки
	glue       [3]widget.Editor // подписи склейки: было, стало, шаг
	glueErr    [3]string

	toolChoice widget.Enum // инструмент после выбора области
	toolRows   []*toolRow
	bColor     widget.Editor
	colorErr   string

	invalidate func() // перерисовать окно из другой горутины
	grant      task   // выдача разрешения на запасной снимок
	monitors   task   // повторный выбор мониторов
	hotkey     task   // назначение горячей клавиши
	hotkeyEd   widget.Editor
	lang       widget.Enum        // язык интерфейса: auto, ru, en
	tab        int                // выбранная вкладка
	tabBtns    []widget.Clickable // кнопки вкладок
	shownTab   int                // вкладка, чьё содержимое нарисовано в последнем кадре
	checkUpd   widget.Bool        // проверять обновления при старте
	update     task               // проверка обновлений
	autostart  widget.Bool        // запуск значка при входе
	autoErr    string
	keyRows    []*keyRow
	capturing  *keyRow // строка, для которой ждём нажатия клавиши
	capTag     bool    // тег фокуса для записи клавиши
	dirErr     string
	prefixErr  string
}

// newShaper собирает шрифты для языка интерфейса: для китайского, японского и
// корейского к встроенным добавляется системный шрифт с иероглифами.
func newShaper() *text.Shaper {
	return text.NewShaper(text.WithCollection(fonts.Collection(i18n.Language())))
}

func newState(store *settings.Store) *state {
	th := material.NewTheme()
	th.Shaper = newShaper()
	th.Palette = material.Palette{Bg: bgColor, Fg: fgColor, ContrastBg: accentColor, ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}

	s := &state{th: th, store: store}
	s.list.Axis = layout.Vertical
	s.tabBtns = make([]widget.Clickable, len(s.tabs()))
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
	s.marker = s.addSlider(settings.MinMarkerOpacity, 100, 1, func(v float64) string { return fmt.Sprintf("%.0f%%", v) },
		func(c settings.Settings) float64 { return float64(c.MarkerOpacity) },
		func(c *settings.Settings, v float64) { c.MarkerOpacity = int(v) })
	s.handle = s.addSlider(0, settings.MaxHandleRadius, 0.5, func(v float64) string {
		if v == 0 {
			return i18n.T("нет")
		}
		return fmt.Sprintf("%.1f px", v)
	},
		func(c settings.Settings) float64 { return c.HandleRadius },
		func(c *settings.Settings, v float64) { c.HandleRadius = v })
	cur := store.Get()
	for _, sl := range s.sliders {
		sl.Set(sl.get(cur))
	}
	s.initKeys()
	s.checkUpd.Value = store.Get().CheckUpdates
	s.autostart.Value = autostartOn()
	s.lang.Value = store.Get().Language
	s.hotkeyEd.SingleLine = true
	s.hotkeyEd.SetText("Print")
	if cur := currentHotkey(); cur != "" {
		s.hotkeyEd.SetText(cur)
	}
	s.initTools(cur)
	s.glueLayout.Value = string(cur.Glue.Layout)
	s.format.Value = string(cur.Format)
	s.quality = s.addSlider(1, 100, 1, func(v float64) string { return fmt.Sprintf("%.0f", v) },
		func(c settings.Settings) float64 { return float64(c.JPEGQuality) },
		func(c *settings.Settings, v float64) { c.JPEGQuality = int(v) })
	s.quality.Set(float64(cur.JPEGQuality))
	s.glueBg.Value = string(cur.Glue.Background)
	whole := func(v float64) string { return fmt.Sprintf("%.0f px", v) }
	s.gluePad = s.addSlider(0, settings.MaxGluePad, 1, whole,
		func(c settings.Settings) float64 { return float64(c.Glue.Pad) },
		func(c *settings.Settings, v float64) { c.Glue.Pad = int(v) })
	s.glueFont = s.addSlider(settings.MinGlueLabelSize, settings.MaxGlueLabelSize, 1, whole,
		func(c settings.Settings) float64 { return float64(c.Glue.LabelSize) },
		func(c *settings.Settings, v float64) { c.Glue.LabelSize = int(v) })
	for _, sl := range s.sliders[len(s.sliders)-2:] {
		sl.Set(sl.get(cur))
	}
	for i, v := range []string{cur.Glue.Before, cur.Glue.After, cur.Glue.Step} {
		s.glue[i].SingleLine = true
		s.glue[i].SetText(v)
	}
	for i := range s.palEd {
		s.palEd[i].SingleLine = true
		s.palEd[i].SetText(cur.Palette[i])
	}
	s.bColor.SingleLine = true
	s.bColor.SetText(cur.BorderColor)
	s.choice.Value = string(cur.DoubleClick)
	s.enter.Value = string(cur.Enter)
	s.notify.Value = cur.Notifications
	s.remember.Value = cur.RememberRegion
	s.showSize.Value = cur.ShowSize
	s.keepOpen.Value = cur.KeepOpen
	s.cursor.Value = cur.IncludeCursor
	s.dir.SingleLine = true
	s.dir.SetText(cur.SaveDir)
	s.prefix.SingleLine = true
	s.prefix.SetText(cur.FilePrefix)
	return s
}

// Run открывает окно настроек и работает, пока его не закроют. Вызывать из main.
func Run(store *settings.Store) {
	i18n.SetLanguage(store.Get().Language)
	app.ID = screencast.AppID
	go func() {
		s := newState(store)
		if store.Get().CheckUpdates {
			s.checkNow()
		}
		loop(s)
		os.Exit(0)
	}()
	app.Main()
}

func loop(s *state) {
	w := new(app.Window)
	s.invalidate = w.Invalidate
	w.Option(app.Title(i18n.T("Scrawl: настройки")), app.Size(unit.Dp(820), unit.Dp(640)), app.MinSize(unit.Dp(700), unit.Dp(360)))
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
					if s.capturing != nil {
						s.capturing = nil // Esc отменяет запись клавиши, а не закрывает окно
					} else {
						w.Perform(system.ActionClose)
					}
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
	// Клик по вкладке обрабатывается до того, как кадр выберет, что рисовать:
	// иначе содержимое сменится только на следующем кадре, а он не придёт,
	// пока не будет нового события, и клик покажется запоздавшим.
	for i := range s.tabBtns {
		if s.tabBtns[i].Clicked(gtx) {
			s.selectTab(i)
		}
	}
	s.checkUpd.Update(gtx)
	if want := s.checkUpd.Value; want != s.store.Get().CheckUpdates {
		if err := s.store.Update(func(c *settings.Settings) { c.CheckUpdates = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}
	if s.update.btn.Clicked(gtx) {
		s.checkNow()
	}
	s.autostart.Update(gtx)
	if want := s.autostart.Value; want != autostartOn() {
		s.autoErr = ""
		var err error
		if want {
			var bin string
			if bin, err = hotkeyBinary(); err == nil {
				err = autostartEnable(bin)
			}
		} else {
			err = autostartOff()
		}
		if err != nil {
			s.autoErr = fmt.Sprintf(i18n.T("Не удалось изменить автозапуск: %v"), err)
			s.autostart.Value = autostartOn() // вернуть галочку в состояние файла
		}
	}
	s.lang.Update(gtx)
	if want := s.lang.Value; want != s.store.Get().Language {
		if err := s.store.Update(func(c *settings.Settings) { c.Language = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
		i18n.SetLanguage(s.store.Get().Language) // окно перерисуется на новом языке сразу
		s.th.Shaper = newShaper()                // и получит шрифт для его письменности
	}
	s.updateKeys(gtx)
	if s.hotkey.btn.Clicked(gtx) {
		accel := strings.TrimSpace(s.hotkeyEd.Text())
		s.start(&s.hotkey, func(func(string)) (string, error) {
			if err := hotkey.ValidateAccel(accel); err != nil {
				return "", err
			}
			bin, err := hotkeyBinary()
			if err != nil {
				return "", fmt.Errorf(i18n.T("не найден путь к программе: %w"), err)
			}
			res, err := assignHotkey(accel, bin)
			if err != nil {
				return "", err
			}
			msg := i18n.T("Горячая клавиша ") + accel + i18n.T(" назначена на ") + bin
			if res.Updated {
				msg = i18n.T("Ярлык обновлён: ") + accel + i18n.T(" запускает ") + bin
			}
			if res.Warning != "" {
				msg += i18n.T(". Внимание: ") + res.Warning
			}
			return msg, nil
		})
	}
	if s.monitors.btn.Clicked(gtx) {
		s.start(&s.monitors, func(progress func(string)) (string, error) {
			progress(i18n.T("Ждём диалоги GNOME…"))
			return setupMonitors(progress)
		})
	}
	if s.grant.btn.Clicked(gtx) {
		s.start(&s.grant, func(func(string)) (string, error) {
			if err := grantPermission(); err != nil {
				return "", fmt.Errorf(i18n.T("Не удалось выдать разрешение: %w"), err)
			}
			return i18n.T("Разрешение выдано приложению ") + screencast.AppID, nil
		})
	}

	s.choice.Update(gtx)
	if want := settings.DoubleClick(s.choice.Value); want != s.store.Get().DoubleClick {
		if err := s.store.Update(func(c *settings.Settings) { c.DoubleClick = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}

	s.enter.Update(gtx)
	if want := settings.Enter(s.enter.Value); want != s.store.Get().Enter {
		if err := s.store.Update(func(c *settings.Settings) { c.Enter = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}

	s.notify.Update(gtx)
	if want := s.notify.Value; want != s.store.Get().Notifications {
		if err := s.store.Update(func(c *settings.Settings) { c.Notifications = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}

	for _, sl := range s.sliders {
		sl.Update(gtx)
		if want := sl.Value(); want != sl.get(s.store.Get()) {
			if err := s.store.Update(func(c *settings.Settings) { sl.set(c, want) }); err != nil {
				s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
			}
		}
	}
	palChanged := s.palReset.Clicked(gtx)
	if palChanged {
		for i := range s.palEd {
			s.palEd[i].SetText(settings.DefaultPalette[i])
		}
	}
	for i := range s.palEd {
		if edited(gtx, &s.palEd[i]) {
			palChanged = true
		}
	}
	if palChanged {
		s.applyPalette()
	}

	s.glueLayout.Update(gtx)
	if want := settings.GlueLayout(s.glueLayout.Value); want != s.store.Get().Glue.Layout {
		if err := s.store.Update(func(c *settings.Settings) { c.Glue.Layout = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}
	s.format.Update(gtx)
	if want := settings.FileFormat(s.format.Value); want != s.store.Get().Format {
		if err := s.store.Update(func(c *settings.Settings) { c.Format = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}
	s.glueBg.Update(gtx)
	if want := settings.GlueBackground(s.glueBg.Value); want != s.store.Get().Glue.Background {
		if err := s.store.Update(func(c *settings.Settings) { c.Glue.Background = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}
	for i := range s.glue {
		if edited(gtx, &s.glue[i]) {
			s.applyGlueLabel(i)
		}
	}

	s.toolChoice.Update(gtx)
	if want := s.toolChoice.Value; want != s.store.Get().DefaultTool {
		if err := s.store.Update(func(c *settings.Settings) { c.DefaultTool = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}
	for _, row := range s.toolRows {
		if edited(gtx, &row.ed) {
			s.applyToolColor(row)
		}
	}
	if edited(gtx, &s.bColor) {
		s.applyBorderColor(s.bColor.Text())
	}

	s.cursor.Update(gtx)
	if want := s.cursor.Value; want != s.store.Get().IncludeCursor {
		if err := s.store.Update(func(c *settings.Settings) { c.IncludeCursor = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}

	s.keepOpen.Update(gtx)
	if want := s.keepOpen.Value; want != s.store.Get().KeepOpen {
		if err := s.store.Update(func(c *settings.Settings) { c.KeepOpen = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}

	s.showSize.Update(gtx)
	if want := s.showSize.Value; want != s.store.Get().ShowSize {
		if err := s.store.Update(func(c *settings.Settings) { c.ShowSize = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
		}
	}

	s.remember.Update(gtx)
	if want := s.remember.Value; want != s.store.Get().RememberRegion {
		if err := s.store.Update(func(c *settings.Settings) { c.RememberRegion = want }); err != nil {
			s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
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
		s.colorErr = i18n.T("Не удалось сохранить: ") + err.Error()
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
		s.dirErr = i18n.T("Не удалось открыть папку: ") + err.Error()
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

// checkNow запрашивает у GitHub последнюю версию в фоне.
func (s *state) checkNow() {
	s.start(&s.update, func(progress func(string)) (string, error) {
		progress(i18n.T("Проверяем…"))
		res, err := checkUpdate()
		if err != nil {
			return "", err
		}
		return res.Message(), nil
	})
}

// applyPrefix проверяет и сохраняет начало имени файла; при ошибке настройка не меняется.
func (s *state) applyPrefix(prefix string) {
	if err := settings.ValidateFilePrefix(prefix); err != nil {
		s.prefixErr = err.Error()
		return
	}
	s.prefixErr = ""
	if err := s.store.Update(func(c *settings.Settings) { c.FilePrefix = prefix }); err != nil {
		s.prefixErr = i18n.T("Не удалось сохранить: ") + err.Error()
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
		s.dirErr = i18n.T("Не удалось сохранить: ") + err.Error()
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

// swatchBox рисует квадратный образец цвета в рамке.
func swatchBox(gtx layout.Context, c color.NRGBA, dp int) layout.Dimensions {
	sz := gtx.Dp(unit.Dp(dp))
	rr := clip.UniformRRect(image.Rect(0, 0, sz, sz), gtx.Dp(6))
	paint.FillShape(gtx.Ops, borderColor, rr.Op(gtx.Ops))
	inner := clip.UniformRRect(image.Rect(gtx.Dp(2), gtx.Dp(2), sz-gtx.Dp(2), sz-gtx.Dp(2)), gtx.Dp(4))
	paint.FillShape(gtx.Ops, c, inner.Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

// toolRow — строка настроек одного инструмента: цвет и толщина по умолчанию.
type toolRow struct {
	name  string // имя в настройках
	label string
	ed    widget.Editor
	sl    *slider
	err   string
}

// toolLabels — названия инструментов в окне, в порядке settings.ToolNames.
var toolLabels = []string{"Выделение", "Перо", "Линия", "Стрелка", "Рамка", "Маркер", "Размытие", "Текст"}

// initTools создаёт строки инструментов и выбор инструмента по умолчанию.
func (s *state) initTools(cur settings.Settings) {
	s.toolChoice.Value = cur.DefaultTool
	for i, name := range settings.ToolNames {
		if name == "select" {
			continue
		}
		name := name
		row := &toolRow{name: name, label: toolLabels[i]} // русский ключ: переводится при показе
		row.ed.SingleLine = true
		row.ed.SetText(cur.Tools.Of(name).Color)
		row.sl = s.addSlider(0, settings.MaxToolWidth, 1, func(v float64) string {
			if v == 0 {
				return i18n.T("станд.")
			}
			return fmt.Sprintf("%.0f", v)
		},
			func(c settings.Settings) float64 { return c.Tools.Of(name).Width },
			func(c *settings.Settings, v float64) { c.Tools.Of(name).Width = v })
		s.toolRows = append(s.toolRows, row)
	}
}

// applyToolColor проверяет и сохраняет цвет инструмента; пусто значит «не менять».
func (s *state) applyToolColor(row *toolRow) {
	txt := strings.TrimSpace(row.ed.Text())
	value := ""
	if txt != "" {
		c, err := settings.ParseColor(txt)
		if err != nil {
			row.err = err.Error()
			return
		}
		value = settings.FormatColor(c)
	}
	row.err = ""
	if err := s.store.Update(func(c *settings.Settings) { c.Tools.Of(row.name).Color = value }); err != nil {
		row.err = i18n.T("Не удалось сохранить: ") + err.Error()
	}
}

// toolControls — выбор инструмента после выделения области и вид каждого инструмента.
func (s *state) toolControls() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(s.toolRows)*2)
		for _, row := range s.toolRows {
			row := row
			c := color.NRGBA{}
			if parsed, err := settings.ParseColor(row.ed.Text()); err == nil {
				c = parsed
			}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(84)
						return s.label(material.Body2, i18n.T(row.label), fgColor)(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if c == (color.NRGBA{}) {
							// Цвет не задан: пустая рамка вместо образца.
							sz := gtx.Dp(28)
							return layout.Dimensions{Size: image.Pt(sz, sz)}
						}
						return swatchBox(gtx, c, 28)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.X = gtx.Dp(120)
						gtx.Constraints.Min.X = gtx.Dp(120)
						return s.field(&row.ed, "#RRGGBB")(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, s.slider(row.sl)),
				)
			}))
			if row.err != "" {
				children = append(children, layout.Rigid(s.label(material.Caption, i18n.T(row.label)+": "+row.err, errorColor)))
			}
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
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
			c := cfg.BorderNRGBA()
			if parsed, err := settings.ParseColor(s.bColor.Text()); err == nil {
				c = parsed
			}
			return swatchBox(gtx, c, 36)
		}
		children := []layout.FlexChild{
			labeled(i18n.T("Толщина"), s.bWidth),
			labeled(i18n.T("Штрих"), s.bDash),
			labeled(i18n.T("Пауза"), s.bGap),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(80)
						return s.label(material.Body2, i18n.T("Цвет"), mutedColor)(gtx)
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

// applyPalette сохраняет цвета панели. Пустые поля берут цвет по умолчанию; пока
// в каком-то поле неверный цвет, настройка не меняется.
func (s *state) applyPalette() {
	var pal [settings.PaletteSize]string
	for i := range s.palEd {
		txt := strings.TrimSpace(s.palEd[i].Text())
		if txt == "" {
			pal[i] = settings.DefaultPalette[i]
			continue
		}
		c, err := settings.ParseColor(txt)
		if err != nil {
			s.palErr = fmt.Sprintf(i18n.T("Цвет %d: %v"), i+1, err)
			return
		}
		pal[i] = settings.FormatColor(c)
	}
	s.palErr = ""
	if err := s.store.Update(func(c *settings.Settings) { c.Palette = pal }); err != nil {
		s.palErr = i18n.T("Не удалось сохранить: ") + err.Error()
	}
}

// paletteControls — цвета панели в две колонки и кнопка сброса.
func (s *state) paletteControls() layout.Widget {
	cell := func(i int) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			c, _ := settings.ParseColor(settings.DefaultPalette[i])
			if parsed, err := settings.ParseColor(s.palEd[i].Text()); err == nil {
				c = parsed
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(s.label(material.Body2, fmt.Sprintf("%d", i+1), mutedColor)),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return swatchBox(gtx, c, 28) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Flexed(1, s.field(&s.palEd[i], settings.DefaultPalette[i])),
			)
		}
	}
	return func(gtx layout.Context) layout.Dimensions {
		const half = settings.PaletteSize / 2
		var children []layout.FlexChild
		for r := 0; r < half; r++ {
			r := r
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Flexed(1, cell(r)),
						layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
						layout.Flexed(1, cell(r+half)),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		if s.palErr != "" {
			children = append(children, layout.Rigid(s.label(material.Caption, s.palErr, errorColor)), layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Button(s.th, &s.palReset, i18n.T("По умолчанию"))
			b.Background = borderColor
			return b.Layout(gtx)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// glueField возвращает указатель на подпись склейки в настройках по номеру поля.
func glueField(g *settings.Glue, i int) *string {
	return [...]*string{&g.Before, &g.After, &g.Step}[i]
}

// applyGlueLabel проверяет и сохраняет подпись склейки; при ошибке настройка не меняется.
func (s *state) applyGlueLabel(i int) {
	txt := strings.TrimSpace(s.glue[i].Text())
	if err := settings.ValidateGlueLabel(txt); err != nil {
		s.glueErr[i] = err.Error()
		return
	}
	s.glueErr[i] = ""
	if err := s.store.Update(func(c *settings.Settings) { *glueField(&c.Glue, i) = txt }); err != nil {
		s.glueErr[i] = i18n.T("Не удалось сохранить: ") + err.Error()
	}
}

// glueControls — три поля со словами подписей склейки.
func (s *state) glueControls() layout.Widget {
	names := [3]string{i18n.T("Первая из двух"), i18n.T("Вторая из двух"), i18n.T("Если больше двух")}
	hints := [3]string{i18n.T(settings.DefaultGlueBefore), i18n.T(settings.DefaultGlueAfter), i18n.T(settings.DefaultGlueStep)}
	return func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		for i := range s.glue {
			i := i
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(130)
						return s.label(material.Body2, names[i], mutedColor)(gtx)
					}),
					layout.Flexed(1, s.field(&s.glue[i], hints[i])),
				)
			}))
			if s.glueErr[i] != "" {
				children = append(children, layout.Rigid(s.label(material.Caption, names[i]+": "+s.glueErr[i], errorColor)))
			}
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// autostartControl — переключатель автозапуска и сообщение об ошибке под ним.
func (s *state) autostartControl() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.checkbox(&s.autostart, i18n.T("Запускать значок при входе в систему")))}
		if s.autoErr != "" {
			children = append(children, layout.Rigid(s.label(material.Caption, s.autoErr, errorColor)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// updateControl — переключатель проверки при старте и кнопка «Проверить сейчас».
func (s *state) updateControl() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.checkbox(&s.checkUpd, i18n.T("Проверять при запуске окна настроек и значка"))),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(s.taskControl(&s.update, i18n.T("Проверить сейчас"))),
		)
	}
}

// hotkeyControl — поле комбинации и кнопка назначения.
func (s *state) hotkeyControl() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.field(&s.hotkeyEd, "Print")),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(s.taskControl(&s.hotkey, i18n.T("Назначить"))),
		)
	}
}

// toolChoiceRadios — выбор инструмента по умолчанию.
func (s *state) toolChoiceRadios() layout.Widget {
	items := make([]layout.Widget, len(settings.ToolNames))
	for i, name := range settings.ToolNames {
		items[i] = s.radio(&s.toolChoice, name, i18n.T(toolLabels[i]))
	}
	return radios(items...)
}

// languageRadios — выбор языка двумя столбцами: «как в системе» и все языки по
// их собственным названиям (их надо узнавать, даже не понимая остального окна).
func (s *state) languageRadios() layout.Widget {
	items := []layout.Widget{s.radio(&s.lang, i18n.Auto, i18n.T("Как в системе"))}
	for _, l := range i18n.Languages {
		name := l.Name
		if l.Hint != "" {
			name += "  ·  " + l.Hint // без шрифта с иероглифами подпись всё равно читается
		}
		items = append(items, s.radio(&s.lang, l.Code, name))
	}
	half := (len(items) + 1) / 2
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{}.Layout(gtx,
			layout.Flexed(1, radios(items[:half]...)),
			layout.Flexed(1, radios(items[half:]...)),
		)
	}
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

// settingsTab — вкладка окна: название и строки настроек в ней.
type settingsTab struct {
	label string
	rows  []layout.Widget
}

// tabs — все вкладки окна и их содержимое. Окно строится заново на каждом кадре,
// поэтому тексты сразу следуют выбранному языку.
func (s *state) tabs() []settingsTab {
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
				layout.Rigid(btn(&s.reset, i18n.T("По умолчанию"))),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(btn(&s.openDir, i18n.T("Открыть папку"))),
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
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, i18n.T("Например: ")+example, mutedColor)))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	general := []layout.Widget{
		s.row(i18n.T("Язык интерфейса"), i18n.T("Окно настроек меняется сразу, оверлей и уведомления при следующем снимке."),
			s.languageRadios()),
		s.section(i18n.T("Действия")),
		s.row(i18n.T("Двойной клик внутри выделения"), i18n.T("Работает с инструментом «выделение», не на рамке."),
			radios(
				s.radio(&s.choice, string(settings.DoubleClickOff), i18n.T("Ничего")),
				s.radio(&s.choice, string(settings.DoubleClickCopy), i18n.T("Копировать в буфер обмена")),
				s.radio(&s.choice, string(settings.DoubleClickCopySave), i18n.T("Копировать и сохранить в файл")),
			)),
		s.row(i18n.T("Клавиша Enter"), i18n.T("Ctrl+C, Ctrl+S и кнопки на панели от этой настройки не зависят."),
			radios(
				s.radio(&s.enter, string(settings.EnterCopySave), i18n.T("Копировать и сохранить в файл")),
				s.radio(&s.enter, string(settings.EnterCopy), i18n.T("Только копировать в буфер обмена")),
				s.radio(&s.enter, string(settings.EnterSave), i18n.T("Только сохранить в файл")),
			)),
		s.row(i18n.T("Курсор мыши"), i18n.T("Работает при обычном захвате экрана. В запасном способе через Screenshot portal курсора в снимке нет."),
			s.checkbox(&s.cursor, i18n.T("Показывать курсор на снимке"))),
		s.row(i18n.T("После копирования"), i18n.T("Enter, Ctrl+C, Ctrl+S, двойной клик и кнопки панели. Выход по Esc."),
			s.checkbox(&s.keepOpen, i18n.T("Не закрывать, а продолжать редактировать"))),
		s.row(i18n.T("Уведомления"), i18n.T("После снимка и действий с подборкой. Ошибки показываются всегда."),
			s.checkbox(&s.notify, i18n.T("Показывать уведомления"))),
		s.row(i18n.T("Прошлая область"), i18n.T("Показывается сразу при запуске и по клавише R. Без неё «было/стало» не подставит область само."),
			s.checkbox(&s.remember, i18n.T("Запоминать область снимка"))),
	}
	tools := []layout.Widget{
		s.row(i18n.T("Инструмент после выбора области"), i18n.T("Включается сам, когда область выделена. «Выделение»: ничего не включать."),
			s.toolChoiceRadios()),
		s.row(i18n.T("Цвет и толщина по умолчанию"), i18n.T("Цвет #RRGGBB: пусто значит не менять. Цвет работает, пока вы сами не выбрали цвет в панели. Толщина для текста это размер, для размытия сила."),
			s.toolControls()),
		s.row(i18n.T("Цвета панели"), i18n.T("Восемь цветов в панели инструментов, #RRGGBB. Пустое поле: цвет по умолчанию. Применяется при следующем снимке."),
			s.paletteControls()),
		s.row(i18n.T("Прозрачность маркера"), i18n.T("Чем больше процент, тем плотнее закраска."), s.slider(s.marker)),
	}
	look := []layout.Widget{
		s.row(i18n.T("Рамка выделения"), i18n.T("Пунктир вокруг выбранной области."), s.borderControls()),
		s.row(i18n.T("Размер области"), i18n.T("Подпись «ширина × высота» над левым верхним углом."),
			s.checkbox(&s.showSize, i18n.T("Показывать размер"))),
		s.row(i18n.T("Угловые ручки"), i18n.T("Кружки в углах рамки, за них меняют размер области. «нет» прячет их."), s.slider(s.handle)),
		s.row(i18n.T("Затемнение вне области"), i18n.T("0% не затемняет совсем, 100% закрашивает чёрным."), s.slider(s.dim)),
	}
	glue := []layout.Widget{
		s.row(i18n.T("Расположение"), i18n.T("Картинки выравниваются по самой низкой (рядом) или по самой узкой (друг под другом)."),
			radios(
				s.radio(&s.glueLayout, string(settings.GlueSide), i18n.T("Рядом, слева направо")),
				s.radio(&s.glueLayout, string(settings.GlueStack), i18n.T("Друг под другом, сверху вниз")),
			)),
		s.row(i18n.T("Фон"), "", radios(
			s.radio(&s.glueBg, string(settings.GlueLight), i18n.T("Белый")),
			s.radio(&s.glueBg, string(settings.GlueDark), i18n.T("Тёмный")),
		)),
		s.row(i18n.T("Отступы"), i18n.T("От краёв и между картинками."), s.slider(s.gluePad)),
		s.row(i18n.T("Размер подписи"), "", s.slider(s.glueFont)),
		s.row(i18n.T("Подписи над картинками"), i18n.T("Пусто: слова по умолчанию. При трёх и больше картинок к слову добавляется номер: «Шаг 1»."),
			s.glueControls()),
	}
	files := []layout.Widget{
		s.row(i18n.T("Папка для скриншотов"), i18n.T("Пусто: ~/Pictures/Screenshots. Можно писать ~/Папка или полный путь."), dirField),
		s.row(i18n.T("Имя файла"), i18n.T("Начало имени, дальше идут дата и время. Пусто: ")+settings.DefaultFilePrefix+".", prefixField),
		s.row(i18n.T("Формат файла"), i18n.T("В буфер обмена снимок всегда идёт в PNG."), radios(
			s.radio(&s.format, string(settings.FormatPNG), i18n.T("PNG: без потерь")),
			s.radio(&s.format, string(settings.FormatJPEG), i18n.T("JPEG: файл меньше, есть потери")),
		)),
		s.row(i18n.T("Качество JPEG"), i18n.T("Чем выше, тем лучше картинка и больше файл."), s.slider(s.quality)),
	}
	keys := []layout.Widget{
		s.row(i18n.T("Клавиши"), i18n.T("Нажмите кнопку и затем новую комбинацию. Esc отменяет запись. Применяются при следующем снимке."),
			s.keyControls()),
	}
	system := append([]layout.Widget{s.section(i18n.T("Захват экрана"))}, s.captureRows()...)
	system = append(system,
		s.section(i18n.T("Запуск")),
		s.row(i18n.T("Автозапуск"), i18n.T("При входе в систему запускается значок в области уведомлений (scrawl -tray): левый клик делает снимок. На GNOME нужно расширение AppIndicator."),
			s.autostartControl()),
	)
	about := []layout.Widget{
		s.row("Scrawl "+version.String(), version.Home, nil),
		s.row(i18n.T("Обновления"), i18n.T("Единственное место, где программа выходит в интернет: запрос к GitHub Releases. По умолчанию выключено; кнопка работает и при выключенной проверке."),
			s.updateControl()),
	}

	return []settingsTab{
		{label: i18n.T("Общие"), rows: general},
		{label: i18n.T("Инструменты"), rows: tools},
		{label: i18n.T("Вид"), rows: look},
		{label: i18n.T("Склейка"), rows: glue},
		{label: i18n.T("Файлы"), rows: files},
		{label: i18n.T("Клавиши"), rows: keys},
		{label: i18n.T("Система"), rows: system},
		{label: i18n.T("О программе"), rows: about},
	}
}

// captureRows — настройки захвата. В сеансе X11 портала нет: мониторы выбирать и
// разрешение выдавать не нужно, вместо этих кнопок короткое пояснение.
func (s *state) captureRows() []layout.Widget {
	rows := []layout.Widget{
		s.row(i18n.T("Горячая клавиша запуска"), i18n.T("Создаёт или обновляет ярлык в настройках клавиатуры GNOME. Формат: Print, <Super>s, <Control><Alt>p. Чужие ярлыки не меняются."),
			s.hotkeyControl()),
	}
	if x11Session() {
		return append(rows, s.row(i18n.T("Сеанс X11"), i18n.T("Экран снимается напрямую, без портала: выбирать мониторы и выдавать разрешение не нужно. Для буфера обмена нужен xclip."), nil))
	}
	return append(rows,
		s.row(i18n.T("Мониторы"), i18n.T("Выбрать мониторы заново: GNOME покажет диалог на каждый монитор. Нужно после подключения или замены монитора. Раньше: scrawl -setup."),
			s.taskControl(&s.monitors, i18n.T("Выбрать мониторы заново"))),
		s.row(i18n.T("Разрешение на запасной снимок"), i18n.T("Нужно запасному способу захвата через Screenshot portal, когда PipeWire не отдал кадр. Раньше: scrawl -grant-screenshot."),
			s.taskControl(&s.grant, i18n.T("Выдать разрешение"))),
	)
}

// rows возвращает строки выбранной вкладки.
func (s *state) rows() []layout.Widget {
	tabs := s.tabs()
	return tabs[min(max(s.tab, 0), len(tabs)-1)].rows
}

// selectTab переключает вкладку и возвращает прокрутку в начало.
func (s *state) selectTab(i int) {
	if i != s.tab {
		s.tab = i
		s.list.Position = layout.Position{}
	}
}

// tabBar рисует вкладки в ряд и под ним тонкую линию на всю ширину.
func (s *state) tabBar(gtx layout.Context, tabs []settingsTab) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return s.tabRow(gtx, tabs) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := gtx.Dp(1)
			w := gtx.Constraints.Max.X
			paint.FillShape(gtx.Ops, borderColor, clip.Rect{Max: image.Pt(w, h)}.Op())
			return layout.Dimensions{Size: image.Pt(w, h)}
		}),
	)
}

// tabRow — сами вкладки: выбранная подчёркнута акцентным цветом, под курсором
// подсвечивается.
func (s *state) tabRow(gtx layout.Context, tabs []settingsTab) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(tabs))
	for i := range tabs {
		i := i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			selected := i == s.tab
			return s.tabBtns[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						size := gtx.Constraints.Min
						if s.tabBtns[i].Hovered() && !selected {
							paint.FillShape(gtx.Ops, borderColor, clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(6)).Op(gtx.Ops))
						}
						if selected {
							h := gtx.Dp(2)
							paint.FillShape(gtx.Ops, accentColor, clip.Rect{Min: image.Pt(0, size.Y-h), Max: size}.Op())
						}
						return layout.Dimensions{Size: size}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						c := mutedColor
						if selected || s.tabBtns[i].Hovered() {
							c = fgColor
						}
						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(9), Left: unit.Dp(11), Right: unit.Dp(11)}.Layout(gtx,
							s.label(material.Body2, tabs[i].label, c))
					}),
				)
			})
		}))
	}
	return layout.Flex{}.Layout(gtx, children...)
}

func (s *state) layout(gtx layout.Context) layout.Dimensions {
	s.apply(gtx)
	th := s.th
	tabs := s.tabs()
	s.tab = min(max(s.tab, 0), len(tabs)-1)
	s.shownTab = s.tab
	items := tabs[s.tab].rows

	return layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.label(material.H6, i18n.T("Настройки"), fgColor)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return s.tabBar(gtx, tabs) }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				list := material.List(th, &s.list)
				return list.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
					// Справа место под полосу прокрутки, чтобы она не лежала на элементах.
					return layout.Inset{Right: unit.Dp(14)}.Layout(gtx, items[i])
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(s.label(material.Caption, i18n.T("Изменения применяются сразу. Файл: ")+settings.Path(), mutedColor)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := material.Button(th, &s.close, i18n.T("Закрыть"))
				b.Background = accentColor
				return b.Layout(gtx)
			}),
		)
	})
}
