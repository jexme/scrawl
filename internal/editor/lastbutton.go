package editor

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/jexme/screen-editor/internal/annotate"
)

const lastButtonText = "Прошлая область  ·  R"

var lastOutline = color.NRGBA{R: 255, G: 255, B: 255, A: 110}

// applyLast возвращает область предыдущего скриншота.
func (m *monitor) applyLast() {
	if !m.hasLast || m.hasSel {
		return
	}
	m.claim()
	m.sel, m.hasSel = m.last, true
	m.tool = annotate.ToolSelect
}

// canOfferLast сообщает, нужно ли сейчас показывать кнопку прошлой области.
func (m *monitor) canOfferLast() bool {
	return m.hasLast && !m.hasSel && !m.dragging && !m.editing
}

// drawLastOutline рисует тонкую рамку на месте прошлой области, чтобы было
// видно, что вернёт кнопка.
func (m *monitor) drawLastOutline(gtx layout.Context, sx float32) {
	if !m.canOfferLast() {
		return
	}
	r := m.last
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Max.Y)))
	p.LineTo(f32.Pt(float32(r.Min.X), float32(r.Max.Y)))
	p.Close()
	paint.FillShape(gtx.Ops, lastOutline, clip.Stroke{Path: p.End(), Width: 1.5 / sx}.Op())
}

// layoutLastButton рисует кнопку «Прошлая область» по центру сверху.
func (m *monitor) layoutLastButton(gtx layout.Context, win image.Point) {
	if !m.canOfferLast() {
		return
	}
	if m.lastBtn.Clicked(gtx) {
		m.applyLast()
		return
	}

	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: iconColor}.Add(gtx.Ops)
	mat := rec.Stop()

	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(1<<12, 1<<12)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, 15, lastButtonText, mat)
	label := rec.Stop()

	icon := gtx.Dp(20)
	padX, padY, gap := gtx.Dp(14), gtx.Dp(9), gtx.Dp(8)
	w := padX + icon + gap + dims.Size.X + padX
	h := max(dims.Size.Y, icon) + 2*padY

	defer op.Offset(image.Pt((win.X-w)/2, gtx.Dp(28))).Push(gtx.Ops).Pop()
	cgtx := gtx
	cgtx.Constraints = layout.Exact(image.Pt(w, h))
	m.lastBtn.Layout(cgtx, func(g layout.Context) layout.Dimensions {
		pill := clip.UniformRRect(image.Rect(0, 0, w, h), h/2)
		paint.FillShape(g.Ops, barBG, pill.Op(g.Ops))
		if m.lastBtn.Hovered() {
			paint.FillShape(g.Ops, accent, clip.Stroke{Path: pill.Path(g.Ops), Width: 2}.Op())
		}

		st := op.Offset(image.Pt(padX, (h-icon)/2)).Push(g.Ops)
		iconUndo(g.Ops, float32(icon), iconColor)
		st.Pop()
		st = op.Offset(image.Pt(padX+icon+gap, (h-dims.Size.Y)/2)).Push(g.Ops)
		label.Add(g.Ops)
		st.Pop()

		area := clip.Rect(image.Rect(0, 0, w, h)).Push(g.Ops)
		pointer.CursorPointer.Add(g.Ops)
		area.Pop()
		return layout.Dimensions{Size: image.Pt(w, h)}
	})
}
