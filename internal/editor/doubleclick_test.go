package editor

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/settings"
)

// dcMonitor готовит монитор с выделением посередине кадра и заданным режимом
// двойного клика.
func dcMonitor(t *testing.T, mode settings.DoubleClick) (*monitor, *bool, string) {
	t.Helper()
	home := sandbox(t)
	m, finished := newTestMonitor(t)
	m.tol = 8
	m.sel, m.hasSel = image.Rect(50, 50, 350, 250), true
	if err := m.store.Update(func(s *settings.Settings) { s.DoubleClick = mode }); err != nil {
		t.Fatal(err)
	}
	return m, finished, home
}

func clipboardImage(home string) bool {
	data, err := os.ReadFile(filepath.Join(home, "clipboard-image_png"))
	return err == nil && len(data) > 100
}

func TestDoubleClickCopies(t *testing.T) {
	m, finished, home := dcMonitor(t, settings.DoubleClickCopy)
	p := f32.Pt(200, 150)
	m.press(p)
	if *finished {
		t.Fatal("одиночный клик не должен завершать программу")
	}
	m.press(p)
	if !*finished || !clipboardImage(home) {
		t.Errorf("двойной клик должен скопировать и закрыть: завершение=%v, буфер=%v", *finished, clipboardImage(home))
	}
	if files, _ := filepath.Glob(filepath.Join(home, "Pictures", "Screenshots", "*.png")); len(files) != 0 {
		t.Errorf("режим copy не должен сохранять файл: %v", files)
	}
}

// Инструмент по умолчанию после выбора области — стрелка, двойной клик должен
// работать и с ним, и с пером (без случайной точки на снимке).
func TestDoubleClickWorksWithDrawingTools(t *testing.T) {
	for _, tool := range []annotate.Tool{annotate.ToolArrow, annotate.ToolLine, annotate.ToolRect, annotate.ToolBlur, annotate.ToolPen, annotate.ToolMarker} {
		m, finished, home := dcMonitor(t, settings.DoubleClickCopy)
		m.tool = tool
		p := f32.Pt(200, 150)
		for i := 0; i < 2; i++ {
			m.press(p)
			if !*finished {
				m.release(p)
			}
		}
		if !*finished || !clipboardImage(home) {
			t.Errorf("инструмент %v: двойной клик должен скопировать и закрыть: завершение=%v, буфер=%v", tool, *finished, clipboardImage(home))
		}
		if len(m.shapes) != 0 {
			t.Errorf("инструмент %v: от кликов не должно остаться фигур: %d", tool, len(m.shapes))
		}
	}
}

func TestDoubleClickCopyAndSave(t *testing.T) {
	m, finished, home := dcMonitor(t, settings.DoubleClickCopySave)
	custom := filepath.Join(home, "мои-скрины")
	if err := m.store.Update(func(s *settings.Settings) { s.SaveDir = custom }); err != nil {
		t.Fatal(err)
	}
	p := f32.Pt(200, 150)
	m.press(p)
	m.press(p)
	if !*finished || !clipboardImage(home) {
		t.Fatalf("завершение=%v, буфер=%v", *finished, clipboardImage(home))
	}
	if files, _ := filepath.Glob(filepath.Join(custom, "*.png")); len(files) != 1 {
		t.Errorf("файл должен лечь в папку из настроек: %v", files)
	}
}

func TestDoubleClickIgnored(t *testing.T) {
	cases := []struct {
		name  string
		setup func(m *monitor)
		p     f32.Point
		gap   time.Duration
	}{
		{"режим off", func(m *monitor) {
			m.store.Update(func(s *settings.Settings) { s.DoubleClick = settings.DoubleClickOff })
		}, f32.Pt(200, 150), 0},
		{"инструмент «текст»", func(m *monitor) { m.tool = annotate.ToolText }, f32.Pt(200, 150), 0},
		{"снаружи области", func(m *monitor) {}, f32.Pt(380, 280), 0},
		{"на краю рамки", func(m *monitor) {}, f32.Pt(50, 150), 0},
		{"слишком медленно", func(m *monitor) {}, f32.Pt(200, 150), doubleClickWindow + 100*time.Millisecond},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, finished, home := dcMonitor(t, settings.DoubleClickCopy)
			c.setup(m)
			m.press(c.p)
			time.Sleep(c.gap)
			m.press(c.p)
			if *finished || clipboardImage(home) {
				t.Errorf("двойной клик не должен срабатывать: завершение=%v, буфер=%v", *finished, clipboardImage(home))
			}
		})
	}
}

func TestFilePrefixUsedInSavedName(t *testing.T) {
	home := sandbox(t)
	m, _ := newTestMonitor(t)
	if err := m.store.Update(func(s *settings.Settings) { s.FilePrefix = "мой" }); err != nil {
		t.Fatal(err)
	}
	path, err := saveFile(m.store.Get(), []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(path), "мой-") || !strings.HasSuffix(path, ".png") || !strings.HasPrefix(path, home) {
		t.Errorf("имя файла не по настройке: %q", path)
	}
}

func TestToolDefaultsFromSettings(t *testing.T) {
	sandbox(t)
	m, _ := newTestMonitor(t)
	if err := m.store.Update(func(s *settings.Settings) {
		s.DefaultTool = "arrow"
		s.Tools.Arrow = settings.ToolStyle{Color: "#1E88E5", Width: 9}
		s.Tools.Text = settings.ToolStyle{Color: "#FB8C00", Width: 7}
	}); err != nil {
		t.Fatal(err)
	}

	// Область готова: включается инструмент по умолчанию, его цвет и толщина.
	m.press(f32.Pt(50, 50))
	m.release(f32.Pt(250, 200))
	if m.tool != annotate.ToolArrow {
		t.Fatalf("после выбора области ожидали стрелку, получили %v", m.tool)
	}
	if m.color != (color.NRGBA{R: 30, G: 136, B: 229, A: 255}) || m.curWidth() != 9 {
		t.Errorf("стрелка: цвет %+v толщина %v", m.color, m.curWidth())
	}
	// Остальные инструменты без настроек: стандартная толщина и тот же цвет.
	if got := m.widthOf(annotate.ToolPen); got != annotate.DefaultWidth {
		t.Errorf("перо без настроек: толщина %v", got)
	}
	// Колесо мыши меняет толщину поверх настройки.
	m.bumpWidth(1)
	if m.curWidth() != 10 {
		t.Errorf("после колеса ожидали 10, получили %v", m.curWidth())
	}
	// У текста свой цвет по умолчанию.
	m.setTool(annotate.ToolText)
	if m.color != (color.NRGBA{R: 251, G: 140, B: 0, A: 255}) || m.curWidth() != 7 {
		t.Errorf("текст: цвет %+v толщина %v", m.color, m.curWidth())
	}
	// Выбрав цвет вручную, пользователь отключает автоцвета на этот запуск.
	m.color, m.colorPicked = annotate.Palette[3], true
	m.setTool(annotate.ToolArrow)
	if m.color != annotate.Palette[3] {
		t.Errorf("цвет, выбранный вручную, не должен сбиваться: %+v", m.color)
	}
}

func TestToolNamesMatchEditorTools(t *testing.T) {
	order := []annotate.Tool{annotate.ToolSelect, annotate.ToolPen, annotate.ToolLine, annotate.ToolArrow,
		annotate.ToolRect, annotate.ToolMarker, annotate.ToolBlur, annotate.ToolText, annotate.ToolStamp}
	if len(order) != len(settings.ToolNames) {
		t.Fatalf("инструментов %d, имён в настройках %d", len(order), len(settings.ToolNames))
	}
	for i, tool := range order {
		if int(tool) != i {
			t.Errorf("инструмент %q: номер %d, ожидали %d", settings.ToolNames[i], tool, i)
		}
	}
	if settings.MinToolWidth != annotate.MinWidth || settings.MaxToolWidth != annotate.MaxWidth || settings.MaxBlurLevel != annotate.MaxBlurLevel {
		t.Error("границы толщины в настройках и редакторе разошлись")
	}
}

func TestPaletteFromSettings(t *testing.T) {
	sandbox(t)
	for i, c := range annotate.Palette {
		if settings.FormatColor(c) != settings.DefaultPalette[i] {
			t.Errorf("цвет %d: в редакторе %s, в настройках %s", i, settings.FormatColor(c), settings.DefaultPalette[i])
		}
	}
	if len(annotate.Palette) != settings.PaletteSize {
		t.Fatalf("в панели %d цветов, в настройках %d", len(annotate.Palette), settings.PaletteSize)
	}

	s := settings.Default()
	s.Palette[0] = "#112233"
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
	m, _ := newTestMonitor(t)
	want := color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}
	if m.palette[0] != want || m.color != want {
		t.Errorf("первый цвет палитры должен быть и текущим: палитра %+v, текущий %+v", m.palette[0], m.color)
	}
	if m.palette[1] != annotate.Palette[1] {
		t.Errorf("остальные цвета по умолчанию: %+v", m.palette[1])
	}
}

func TestMarkerOpacityFromSettings(t *testing.T) {
	sandbox(t)
	m, _ := newTestMonitor(t)
	m.press(f32.Pt(50, 50))
	m.release(f32.Pt(300, 250))
	m.setTool(annotate.ToolMarker)

	m.press(f32.Pt(100, 100))
	if m.cur == nil || m.cur.Alpha != 110 {
		t.Fatalf("маркер по умолчанию: альфа 110, получили %+v", m.cur)
	}
	m.release(f32.Pt(200, 100))

	if err := m.store.Update(func(s *settings.Settings) { s.MarkerOpacity = 80 }); err != nil {
		t.Fatal(err)
	}
	m.press(f32.Pt(100, 150))
	if m.cur == nil || m.cur.Alpha != 204 {
		t.Errorf("80%% это альфа 204: %+v", m.cur)
	}
}

func TestSaveFileFormat(t *testing.T) {
	sandbox(t)
	m, _ := newTestMonitor(t)
	// Картинка с шумом, чтобы качество JPEG влияло на размер.
	img := image.NewRGBA(image.Rect(0, 0, 200, 150))
	for i := range img.Pix {
		img.Pix[i] = uint8(i * 7 % 251)
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	data, err := encodePNG(img)
	if err != nil {
		t.Fatal(err)
	}

	pngPath, err := saveFile(m.store.Get(), data)
	if err != nil || !strings.HasSuffix(pngPath, ".png") {
		t.Fatalf("PNG по умолчанию: %q, %v", pngPath, err)
	}
	if saved, _ := os.ReadFile(pngPath); !bytes.Equal(saved, data) {
		t.Error("PNG записывается без перекодирования")
	}

	sizeAt := func(q int) int {
		if err := m.store.Update(func(s *settings.Settings) { s.Format, s.JPEGQuality = settings.FormatJPEG, q }); err != nil {
			t.Fatal(err)
		}
		path, err := saveNumbered(m.store.Get(), data, "20260101-120000", q)
		if err != nil || !strings.HasSuffix(path, ".jpg") {
			t.Fatalf("JPEG: %q, %v", path, err)
		}
		saved, err := os.ReadFile(path)
		if err != nil || len(saved) < 3 || saved[0] != 0xFF || saved[1] != 0xD8 {
			t.Fatalf("файл не похож на JPEG: %v", err)
		}
		if got, err := jpeg.Decode(bytes.NewReader(saved)); err != nil || got.Bounds().Dx() != 200 {
			t.Fatalf("JPEG не читается: %v", err)
		}
		return len(saved)
	}
	if low, high := sizeAt(10), sizeAt(95); low >= high {
		t.Errorf("качество 10 должно давать файл меньше, чем 95: %d и %d байт", low, high)
	}
}

func TestKeepOpenAfterExport(t *testing.T) {
	home := sandbox(t)
	shots := func() int {
		files, _ := filepath.Glob(filepath.Join(home, "Pictures", "Screenshots", "*.png"))
		return len(files)
	}

	m, finished := newTestMonitor(t)
	m.sel, m.hasSel = image.Rect(10, 10, 200, 150), true
	m.export(true, true)
	if !*finished {
		t.Fatal("по умолчанию после копирования программа закрывается")
	}

	if err := m.store.Update(func(s *settings.Settings) { s.KeepOpen = true }); err != nil {
		t.Fatal(err)
	}
	m, finished = newTestMonitor(t)
	m.sel, m.hasSel = image.Rect(10, 10, 200, 150), true
	m.export(true, false)
	if *finished {
		t.Fatal("с включённой настройкой оверлей остаётся открытым")
	}
	if m.toastMsg != "Скопировано в буфер обмена" || time.Until(m.toastUntil) <= 0 {
		t.Errorf("должна быть плашка о копировании: %q", m.toastMsg)
	}
	if !m.hasSel {
		t.Error("область должна остаться для дальнейшей правки")
	}

	before := shots()
	m.export(false, true)
	m.export(true, true)
	if *finished || shots() != before+2 {
		t.Errorf("повторные сохранения должны работать: файлов было %d, стало %d", before, shots())
	}
	if !strings.HasPrefix(m.toastMsg, "Сохранено: ") || !strings.HasSuffix(m.toastMsg, ", скопировано") {
		t.Errorf("плашка после сохранения и копирования: %q", m.toastMsg)
	}
	// Ширина обычной плашки возвращается после смены толщины.
	m.setTool(annotate.ToolPen)
	m.bumpWidth(1)
	if m.toastMsg != "" {
		t.Error("плашка толщины должна заменить сообщение")
	}
}

func TestSaveFileDoesNotOverwrite(t *testing.T) {
	sandbox(t)
	m, _ := newTestMonitor(t)
	var paths []string
	for i := 0; i < 3; i++ {
		p, err := saveNumbered(m.store.Get(), []byte{byte(i)}, "20260101-120000", 1)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	for i, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil || len(data) != 1 || data[0] != byte(i) {
			t.Errorf("файл %d (%s): %v %v", i, p, data, err)
		}
	}
	if paths[0] == paths[1] || paths[1] == paths[2] || !strings.HasSuffix(paths[1], "-1-2.png") || !strings.HasSuffix(paths[2], "-1-3.png") {
		t.Errorf("имена должны различаться: %v", paths)
	}
}

// actionFor возвращает действие, на которое сработает нажатие, или "".
func actionFor(m *monitor, name key.Name, mods key.Modifiers) string {
	for _, kb := range m.keys {
		if kb.binding.Matches(key.Event{Name: name, Modifiers: mods, State: key.Press}) {
			return kb.action
		}
	}
	return ""
}

func TestKeysFromSettings(t *testing.T) {
	sandbox(t)
	m, _ := newTestMonitor(t)
	for want, ev := range map[string]key.Event{
		settings.ActionConfirm: {Name: key.NameReturn}, settings.ActionCancel: {Name: key.NameEscape},
		settings.ActionRestore: {Name: "R"}, settings.ActionAdd: {Name: "A"}, settings.ActionGlue: {Name: "G"},
		settings.ActionSend: {Name: "D"}, settings.ActionUndo: {Name: "Z", Modifiers: key.ModCtrl},
		settings.ActionCopy: {Name: "C", Modifiers: key.ModCtrl}, settings.ActionSave: {Name: "S", Modifiers: key.ModCtrl},
	} {
		if got := actionFor(m, ev.Name, ev.Modifiers); got != want {
			t.Errorf("клавиша %v по умолчанию: действие %q, ожидали %q", ev.Name, got, want)
		}
	}
	if len(m.keyFilters()) < len(settings.KeyActions) {
		t.Errorf("фильтров %d, действий %d", len(m.keyFilters()), len(settings.KeyActions))
	}

	if err := m.store.Update(func(s *settings.Settings) {
		s.Keys.Set(settings.ActionRestore, "F5")
		s.Keys.Set(settings.ActionSave, "Ctrl+Shift+S")
		s.Keys.Set(settings.ActionCancel, "Q")
	}); err != nil {
		t.Fatal(err)
	}
	m, finished := newTestMonitor(t)
	if got := actionFor(m, "F5", 0); got != settings.ActionRestore {
		t.Errorf("F5: %q", got)
	}
	if got := actionFor(m, "R", 0); got != "" {
		t.Errorf("старая клавиша R больше ничего не делает: %q", got)
	}
	if got := actionFor(m, "S", key.ModCtrl); got != "" {
		t.Errorf("Ctrl+S больше не сохраняет: %q", got)
	}
	if got := actionFor(m, "S", key.ModCtrl|key.ModShift); got != settings.ActionSave {
		t.Errorf("Ctrl+Shift+S: %q", got)
	}
	// «Выход» на Q действительно закрывает оверлей.
	for _, kb := range m.keys {
		if kb.action == settings.ActionCancel {
			kb.run()
		}
	}
	if !*finished {
		t.Error("Q должна закрывать оверлей")
	}
}
