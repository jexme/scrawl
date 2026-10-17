package editor

import (
	"image"
	"os"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/jexme/scrawl/internal/screencast"
)

// TestToolbarKeyCaptions рисует панель и проверяет, что клавиши написаны только под кнопками
// с быстрым вызовом, а у инструментов подписей нет. С SCRAWL_TEST_OUT=путь.png результат сохраняется.
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
	toolsCaps := light(image.Rect(capRow.Min.X, capRow.Min.Y, capRow.Min.X+8*38, capRow.Max.Y))
	if toolsCaps != 0 {
		t.Errorf("под инструментами не должно быть подписей: найдено %d светлых пикселей", toolsCaps)
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
