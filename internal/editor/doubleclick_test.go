package editor

import (
	"image"
	"os"
	"path/filepath"
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
