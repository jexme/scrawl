package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/op"
	"gioui.org/op/paint"
)

// TestDashedRect рисует пунктирную рамку и проверяет, что на линии есть и
// штрихи, и паузы.
func TestDashedRect(t *testing.T) {
	const W, H = 300, 200
	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()

	var ops op.Ops
	paint.Fill(&ops, color.NRGBA{A: 255})
	r := image.Rect(40, 40, 260, 160)
	dashedRect(&ops, r, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, 2, 8, 6)
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}

	// Верхняя сторона: идём вдоль неё и считаем переходы светлый/тёмный.
	lit, transitions := 0, 0
	prev := false
	for x := r.Min.X + 10; x < r.Max.X-10; x++ {
		on := out.RGBAAt(x, r.Min.Y).R > 128
		if on {
			lit++
		}
		if on != prev {
			transitions++
		}
		prev = on
	}
	if lit == 0 || lit == r.Dx()-20 {
		t.Errorf("на верхней стороне должны быть и штрихи, и паузы: светлых пикселей %d", lit)
	}
	if transitions < 10 {
		t.Errorf("слишком мало штрихов на стороне: %d переходов", transitions)
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".dash.png", data, 0o644)
	}
}
