package editor

import (
	"image"
	"image/color"
	"os"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/jexme/screen-editor/internal/annotate"
	"github.com/jexme/screen-editor/internal/screencast"
)

// TestToast рисует плашку «Толщина: N» для разных инструментов и проверяет, что
// она нарисована, а после срока жизни исчезает. С SCREEN_EDITOR_TEST_OUT=путь
// картинка сохраняется для просмотра.
func TestToast(t *testing.T) {
	const W, H = 560, 420
	frame := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := 0; i < len(frame.Pix); i += 4 {
		frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2], frame.Pix[i+3] = 90, 96, 110, 255
	}
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: frame}, func() {})

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()

	render := func(tool annotate.Tool, width float32, live bool) *image.RGBA {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		paint.Fill(&ops, color.NRGBA{R: 90, G: 96, B: 110, A: 255})
		m.tool, m.color = tool, annotate.Palette[0]
		m.widths[tool] = width
		m.showToast()
		if !live {
			m.toastUntil = time.Now().Add(-time.Second)
		}
		m.layoutToast(gtx, image.Pt(W, H), image.Rect(20, 20, W-20, H-20))
		if err := win.Frame(&ops); err != nil {
			t.Fatal(err)
		}
		out := image.NewRGBA(image.Rect(0, 0, W, H))
		if err := win.Screenshot(out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	differs := func(img *image.RGBA) bool {
		bg := color.RGBA{R: 90, G: 96, B: 110, A: 255}
		for y := 0; y < 80; y++ {
			for x := 0; x < W; x++ {
				if img.RGBAAt(x, y) != bg {
					return true
				}
			}
		}
		return false
	}
	if !differs(render(annotate.ToolArrow, 8, true)) {
		t.Error("плашка не нарисована")
	}
	if differs(render(annotate.ToolArrow, 8, false)) {
		t.Error("плашка должна исчезнуть после срока жизни")
	}

	if path := os.Getenv("SCREEN_EDITOR_TEST_OUT"); path != "" {
		// Несколько плашек подряд на одном кадре для просмотра.
		sheet := image.NewRGBA(image.Rect(0, 0, W, 4*70))
		for i, c := range []struct {
			tool annotate.Tool
			w    float32
		}{{annotate.ToolArrow, 4}, {annotate.ToolArrow, 14}, {annotate.ToolBlur, 10}, {annotate.ToolText, 6}} {
			img := render(c.tool, c.w, true)
			for y := 0; y < 70; y++ {
				for x := 0; x < W; x++ {
					sheet.SetRGBA(x, i*70+y, img.RGBAAt(x, y+8))
				}
			}
		}
		data, _ := encodePNG(sheet)
		os.WriteFile(path+".toast.png", data, 0o644)
	}
}
