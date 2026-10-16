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

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
)

// TestToast рисует плашку «Толщина: N» для разных инструментов и проверяет, что
// она нарисована, а после срока жизни исчезает. С SCRAWL_TEST_OUT=путь
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

	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
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

// TestSizeLabelSetting: подпись «W × H» рисуется, пока её не выключили в настройках.
func TestSizeLabelSetting(t *testing.T) {
	sandbox(t)
	const W, H = 400, 300
	m, _ := newTestMonitor(t)
	m.sel, m.hasSel = image.Rect(100, 120, 300, 220), true

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()

	shot := func() *image.RGBA {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		m.layout(gtx, image.Pt(W, H))
		if err := win.Frame(&ops); err != nil {
			t.Fatal(err)
		}
		out := image.NewRGBA(image.Rect(0, 0, W, H))
		if err := win.Screenshot(out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	// Участок над левым верхним углом области, где стоит подпись.
	labelArea := image.Rect(100, 90, 180, 118)
	diff := func(a, b *image.RGBA) int {
		n := 0
		for y := labelArea.Min.Y; y < labelArea.Max.Y; y++ {
			for x := labelArea.Min.X; x < labelArea.Max.X; x++ {
				if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
					n++
				}
			}
		}
		return n
	}

	with := shot()
	if err := m.store.Update(func(s *settings.Settings) { s.ShowSize = false }); err != nil {
		t.Fatal(err)
	}
	without := shot()
	if n := diff(with, without); n < 100 {
		t.Errorf("подпись размера должна исчезнуть при выключенной настройке, различий пикселей: %d", n)
	}
}
