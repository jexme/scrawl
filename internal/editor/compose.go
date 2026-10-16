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
	"github.com/jexme/screen-editor/internal/settings"
)

// Цвета склейки для светлого и тёмного фона.
var (
	composeLight = composeColors{
		bg:     color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		text:   color.NRGBA{R: 32, G: 33, B: 36, A: 255},
		border: color.NRGBA{R: 205, G: 209, B: 218, A: 255},
	}
	composeDark = composeColors{
		bg:     color.NRGBA{R: 32, G: 33, B: 36, A: 255},
		text:   color.NRGBA{R: 235, G: 235, B: 235, A: 255},
		border: color.NRGBA{R: 74, G: 75, B: 79, A: 255},
	}
)

type composeColors struct{ bg, text, border color.NRGBA }

func composePalette(g settings.Glue) composeColors {
	if g.Background == settings.GlueDark {
		return composeDark
	}
	return composeLight
}

// composeMetrics возвращает отступ (он же промежуток между картинками) и высоту
// строки подписи в пикселях результата.
func composeMetrics(g settings.Glue) (pad, labelH int) {
	return g.Pad, g.LabelSize + 24
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

// composeSlot — место одной картинки в склейке: подпись и сама картинка.
type composeSlot struct {
	label image.Rectangle // область подписи
	dst   image.Rectangle // область картинки
}

// composeSlots раскладывает картинки: рядом (одной высоты, по самой низкой) или
// друг под другом (одной ширины, по самой узкой), без увеличения. Возвращает
// места картинок и размер результата.
func composeSlots(imgs []*image.RGBA, g settings.Glue) (slots []composeSlot, size image.Point) {
	composePad, composeLabelH := composeMetrics(g)
	composeGap := composePad
	slots = make([]composeSlot, len(imgs))
	if g.Layout == settings.GlueStack {
		wc := imgs[0].Bounds().Dx()
		for _, im := range imgs {
			wc = min(wc, im.Bounds().Dx())
		}
		y := composePad
		for i, im := range imgs {
			b := im.Bounds()
			h := int(math.Round(float64(b.Dy()) * float64(wc) / float64(b.Dx())))
			slots[i].label = image.Rect(composePad, y, composePad+wc, y+composeLabelH)
			slots[i].dst = image.Rect(composePad, y+composeLabelH, composePad+wc, y+composeLabelH+h)
			y += composeLabelH + h + composeGap
		}
		return slots, image.Pt(composePad*2+wc, y-composeGap+composePad)
	}

	hc := imgs[0].Bounds().Dy()
	for _, im := range imgs {
		hc = min(hc, im.Bounds().Dy())
	}
	x := composePad
	for i, im := range imgs {
		b := im.Bounds()
		w := int(math.Round(float64(b.Dx()) * float64(hc) / float64(b.Dy())))
		slots[i].label = image.Rect(x, composePad, x+w, composePad+composeLabelH)
		slots[i].dst = image.Rect(x, composePad+composeLabelH, x+w, composePad+composeLabelH+hc)
		x += w + composeGap
	}
	return slots, image.Pt(x-composeGap+composePad, composePad*2+composeLabelH+hc)
}

// compose склеивает картинки в одну: с подписью сверху у каждой и тонкой рамкой.
func compose(imgs []*image.RGBA, labels []string, shaper *text.Shaper, glue settings.Glue) (*image.RGBA, error) {
	if len(imgs) < 2 {
		return nil, fmt.Errorf("для склейки нужно минимум две картинки, есть %d", len(imgs))
	}
	if len(labels) != len(imgs) {
		return nil, fmt.Errorf("подписей %d, картинок %d", len(labels), len(imgs))
	}

	slots, size := composeSlots(imgs, glue)
	W, H := size.X, size.Y

	win, err := headless.NewWindow(W, H)
	if err != nil {
		return nil, fmt.Errorf("headless: %w", err)
	}
	defer win.Release()

	var ops op.Ops
	colors := composePalette(glue)
	paint.Fill(&ops, colors.bg)
	gtx := layout.Context{
		Ops:         &ops,
		Constraints: layout.Constraints{Max: image.Pt(W, H)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}

	for i, im := range imgs {
		slot := slots[i]
		drawLabelCentered(gtx, shaper, labels[i], slot.label, glue.LabelSize, colors.text)

		b := im.Bounds()
		sx := float32(slot.dst.Dx()) / float32(b.Dx())
		sy := float32(slot.dst.Dy()) / float32(b.Dy())
		aff := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(sx, sy)).Offset(f32.Pt(float32(slot.dst.Min.X), float32(slot.dst.Min.Y)))
		st := op.Affine(aff).Push(&ops)
		cl := clip.Rect(b).Push(&ops)
		paint.NewImageOp(im).Add(&ops)
		paint.PaintOp{}.Add(&ops)
		cl.Pop()
		st.Pop()

		strokeBox(&ops, slot.dst, colors.border, 2)
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

// drawLabelCentered рисует подпись по центру области r.
func drawLabelCentered(gtx layout.Context, shaper *text.Shaper, txt string, r image.Rectangle, size int, col color.NRGBA) {
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := rec.Stop()

	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(r.Dx(), r.Dy())}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{Alignment: text.Middle, MaxLines: 1}.Layout(lgtx, shaper, annotate.Font, unit.Sp(size), txt, mat)
	label := rec.Stop()

	defer op.Offset(image.Pt(r.Min.X+(r.Dx()-dims.Size.X)/2, r.Min.Y+(r.Dy()-dims.Size.Y)/2)).Push(gtx.Ops).Pop()
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
