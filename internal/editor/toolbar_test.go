package editor

import (
	"image"
	"os"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
)

// TestToolbarKeyCaptions рисует панель и проверяет, что клавиши написаны под кнопками
// с быстрым вызовом (инструменты и действия), а под цветами подписей нет. С SCRAWL_TEST_OUT=путь.png результат сохраняется.
func TestToolbarKeyCaptions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const W, H = 1000, 300
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = 60, 70, 90, 255
	}
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(100, 20, 500, 120), true

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(image.Pt(W, H))}
	m.layoutToolbar(gtx, image.Pt(W, H), m.sel)
	if err := win.Frame(&ops); err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}

	bar := m.barRect
	if bar.Dy() < 34+13 {
		t.Fatalf("панель %v слишком низкая для подписей", bar)
	}
	// Строка подписей: под кнопками (нижняя часть панели). Светлые пиксели текста там есть
	// только под действиями с клавишей; считаем их в полосе под первой кнопкой и под кнопкой «копировать».
	capRow := image.Rect(bar.Min.X, bar.Max.Y-13, bar.Max.X, bar.Max.Y)
	light := func(r image.Rectangle) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if c := out.RGBAAt(x, y); c.R > 150 && c.G > 150 && c.B > 150 {
					n++
				}
			}
		}
		return n
	}
	// Под инструментами подписи есть (у каждого своя клавиша), под цветами нет.
	toolsEnd := bar.Min.X + 6 + 8*38
	if n := light(image.Rect(bar.Min.X, capRow.Min.Y, toolsEnd, capRow.Max.Y)); n < 60 {
		t.Errorf("под инструментами нет подписей клавиш: найдено %d светлых пикселей", n)
	}
	if n := light(image.Rect(toolsEnd+12, capRow.Min.Y, toolsEnd+12+7*38, capRow.Max.Y)); n != 0 {
		t.Errorf("под цветами не должно быть подписей: найдено %d светлых пикселей", n)
	}
	rightCaps := light(image.Rect(capRow.Max.X-6*38, capRow.Min.Y, capRow.Max.X, capRow.Max.Y))
	if rightCaps < 40 {
		t.Errorf("под кнопками действий нет подписей клавиш: найдено %d светлых пикселей", rightCaps)
	}

	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, err := encodePNG(out)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestToolKeys: клавиша инструмента включает его, но только когда область выбрана.
func TestToolKeys(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	run := func(action string) {
		t.Helper()
		for _, kb := range m.keys {
			if kb.action == action {
				kb.run()
				return
			}
		}
		t.Fatalf("действия %q нет среди клавиш", action)
	}

	m.tool = annotate.ToolSelect
	run(settings.ToolAction("pen"))
	if m.tool != annotate.ToolSelect {
		t.Errorf("без области клавиша инструмента не должна его включать: %v", m.tool)
	}
	m.sel, m.hasSel = image.Rect(10, 10, 50, 50), true
	for t2, name := range settings.ToolNames {
		run(settings.ToolAction(name))
		if m.tool != annotate.Tool(t2) {
			t.Errorf("клавиша %q: инструмент %v, ожидали %v", name, m.tool, annotate.Tool(t2))
		}
	}
}

// TestWidthKeys: плюс и минус меняют толщину текущего инструмента, у «выделения» ничего не делают.
func TestWidthKeys(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(10, 10, 50, 50), true
	run := func(action string) {
		t.Helper()
		for _, kb := range m.keys {
			if kb.action == action {
				kb.run()
				return
			}
		}
		t.Fatalf("действия %q нет среди клавиш", action)
	}

	m.tool = annotate.ToolSelect
	run(settings.ActionWidthUp)
	if _, ok := m.widths[annotate.ToolSelect]; ok {
		t.Error("у «выделения» толщины нет")
	}
	m.tool = annotate.ToolPen
	before := m.curWidth()
	run(settings.ActionWidthUp)
	if m.curWidth() != before+1 {
		t.Errorf("плюс: %v -> %v", before, m.curWidth())
	}
	run(settings.ActionWidthDown)
	run(settings.ActionWidthDown)
	if m.curWidth() != before-1 {
		t.Errorf("два минуса: %v -> %v", before, m.curWidth())
	}
}
