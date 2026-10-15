package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/jexme/screen-editor/internal/annotate"
	"github.com/jexme/screen-editor/internal/screencast"
)

// TestToolPreviews рисует карточки-образцы всех инструментов на одном кадре.
// С SCREEN_EDITOR_TEST_OUT=путь результат сохраняется для просмотра.
func TestToolPreviews(t *testing.T) {
	const W, H = 900, 330
	frame := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			v := uint8(70 + (x/18+y/18)%2*50) // шахматка, чтобы размытие было заметно
			frame.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v + 20, A: 255})
		}
	}
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: frame}, func() {})
	m.sel, m.hasSel, m.hoverSeen, m.scaleX = frame.Bounds(), true, true, 1
	m.tol = 8

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()

	var ops op.Ops
	clipImg := clip.Rect(frame.Bounds()).Push(&ops)
	paint.NewImageOp(frame).Add(&ops)
	paint.PaintOp{}.Add(&ops)
	clipImg.Pop()

	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	tools := []annotate.Tool{annotate.ToolPen, annotate.ToolLine, annotate.ToolArrow, annotate.ToolRect,
		annotate.ToolMarker, annotate.ToolBlur, annotate.ToolText}
	for i, tool := range tools {
		m.tool = tool
		m.color = annotate.Palette[[]int{0, 4, 0, 3, 2, 0, 5}[i]]
		m.widths[tool] = []float32{4, 4, 6, 4, 4, 4, 4}[i]
		m.hover = f32.Pt(float32(30+i*125), 60)
		m.drawToolPreview(gtx)
	}
	// Толстая стрелка и сильное размытие во второй строке: карточка растёт вместе с силой.
	for i, c := range []struct {
		tool annotate.Tool
		w    float32
	}{{annotate.ToolArrow, 16}, {annotate.ToolMarker, 14}, {annotate.ToolBlur, 14}, {annotate.ToolRect, 20}} {
		m.tool, m.color = c.tool, annotate.Palette[0]
		m.widths[c.tool] = c.w
		m.hover = f32.Pt(float32(30+i*200), 190)
		m.drawToolPreview(gtx)
	}

	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("SCREEN_EDITOR_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".preview.png", data, 0o644)
	}
}
