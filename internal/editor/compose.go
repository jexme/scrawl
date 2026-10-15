package editor

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/jexme/screen-editor/internal/annotate"
)

// Размеры склейки в пикселях результата.
const (
	composePad    = 28
	composeGap    = 28
	composeLabelH = 52
	composeFont   = 28
)

var (
	composeBG     = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	composeText   = color.NRGBA{R: 32, G: 33, B: 36, A: 255}
	composeBorder = color.NRGBA{R: 205, G: 209, B: 218, A: 255}
)

// composeLabels подписывает картинки: «Было» и «Стало» для двух, «Шаг N» для большего числа.
func composeLabels(n int) []string {
	if n == 2 {
		return []string{"Было", "Стало"}
	}
	labels := make([]string, n)
	for i := range labels {
		labels[i] = fmt.Sprintf("Шаг %d", i+1)
	}
	return labels
}

// loadPNGs читает картинки подборки с диска.
func loadPNGs(paths []string) ([]*image.RGBA, error) {
	out := make([]*image.RGBA, 0, len(paths))
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		src, err := png.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		rgba := image.NewRGBA(src.Bounds())
		draw.Draw(rgba, rgba.Bounds(), src, src.Bounds().Min, draw.Src)
		out = append(out, rgba)
	}
	return out, nil
}

// compose склеивает картинки в ряд: одной высоты (по самой низкой, без
// увеличения), с подписью сверху и тонкой рамкой.
func compose(imgs []*image.RGBA, labels []string, shaper *text.Shaper) (*image.RGBA, error) {
	if len(imgs) < 2 {
		return nil, fmt.Errorf("для склейки нужно минимум две картинки, есть %d", len(imgs))
	}
	if len(labels) != len(imgs) {
		return nil, fmt.Errorf("подписей %d, картинок %d", len(labels), len(imgs))
	}

	hc := imgs[0].Bounds().Dy()
	for _, im := range imgs {
		hc = min(hc, im.Bounds().Dy())
	}
	widths := make([]int, len(imgs))
	total := composePad*2 + composeGap*(len(imgs)-1)
	for i, im := range imgs {
		b := im.Bounds()
		widths[i] = int(math.Round(float64(b.Dx()) * float64(hc) / float64(b.Dy())))
		total += widths[i]
	}
	W, H := total, composePad*2+composeLabelH+hc

	win, err := headless.NewWindow(W, H)
	if err != nil {
		return nil, fmt.Errorf("headless: %w", err)
	}
	defer win.Release()

	var ops op.Ops
	paint.Fill(&ops, composeBG)
	gtx := layout.Context{
		Ops:         &ops,
		Constraints: layout.Constraints{Max: image.Pt(W, H)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}

	x := composePad
	for i, im := range imgs {
		drawLabelCentered(gtx, shaper, labels[i], x, composePad, widths[i])

		dst := image.Rect(x, composePad+composeLabelH, x+widths[i], composePad+composeLabelH+hc)
		b := im.Bounds()
		sx := float32(widths[i]) / float32(b.Dx())
		sy := float32(hc) / float32(b.Dy())
		aff := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(sx, sy)).Offset(f32.Pt(float32(dst.Min.X), float32(dst.Min.Y)))
		st := op.Affine(aff).Push(&ops)
		cl := clip.Rect(b).Push(&ops)
		paint.NewImageOp(im).Add(&ops)
		paint.PaintOp{}.Add(&ops)
		cl.Pop()
		st.Pop()

		strokeBox(&ops, dst, composeBorder, 2)
		x += widths[i] + composeGap
	}

	if err := win.Frame(&ops); err != nil {
		return nil, err
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		return nil, err
	}
	return out, nil
}

// drawLabelCentered рисует подпись по центру колонки шириной w, начиная с x.
func drawLabelCentered(gtx layout.Context, shaper *text.Shaper, txt string, x, y, w int) {
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: composeText}.Add(gtx.Ops)
	mat := rec.Stop()

	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(w, composeLabelH)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{Alignment: text.Middle, MaxLines: 1}.Layout(lgtx, shaper, annotate.Font, unit.Sp(composeFont), txt, mat)
	label := rec.Stop()

	defer op.Offset(image.Pt(x+(w-dims.Size.X)/2, y+(composeLabelH-dims.Size.Y)/2)).Push(gtx.Ops).Pop()
	label.Add(gtx.Ops)
}

func strokeBox(ops *op.Ops, r image.Rectangle, c color.NRGBA, width float32) {
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Max.Y)))
	p.LineTo(f32.Pt(float32(r.Min.X), float32(r.Max.Y)))
	p.Close()
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}
