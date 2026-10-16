package editor

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"

	"github.com/jexme/screen-editor/internal/annotate"
	"github.com/jexme/screen-editor/internal/settings"
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
		{"не инструмент выделения", func(m *monitor) { m.tool = annotate.ToolArrow }, f32.Pt(200, 150), 0},
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
		annotate.ToolRect, annotate.ToolMarker, annotate.ToolBlur, annotate.ToolText}
	if len(order) != len(settings.ToolNames) {
		t.Fatalf("инструментов %d, имён в настройках %d", len(order), len(settings.ToolNames))
	}
	for i, tool := range order {
		if int(tool) != i {
			t.Errorf("инструмент %q: номер %d, ожидали %d", settings.ToolNames[i], tool, i)
		}
	}
	if settings.MinToolWidth != annotate.MinWidth || settings.MaxToolWidth != annotate.MaxWidth {
		t.Error("границы толщины в настройках и редакторе разошлись")
	}
}
