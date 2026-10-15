package editor

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/jexme/screen-editor/internal/annotate"
)

// cardRect — прямоугольник в пикселях кадра с дробными границами.
type cardRect struct{ Min, Max f32.Point }

// drawToolPreview рисует рядом с курсором карточку-образец: как будет выглядеть
// стрелка, линия, рамка, перо, маркер, текст или размытие с текущими толщиной и
// цветом. Карточка стоит со смещением от курсора, чтобы не выглядеть готовой
// фигурой под ним. Вызывается внутри клипа выделения.
func (m *monitor) drawToolPreview(gtx layout.Context) {
	if !m.hasSel || !m.hoverSeen || m.tool == annotate.ToolSelect {
		return
	}
	if m.cur != nil || m.editing || m.dragging || m.adjust != adjustNone || !inRect(m.hover, m.sel) {
		return
	}
	// Над ручками рамки рисовать нельзя, поэтому образец там не нужен.
	if _, onHandle := hitHandle(m.sel, m.hover, m.tol, false); onHandle {
		return
	}

	w := m.curWidth()
	card := m.previewCard(w)
	r := image.Rect(int(card.Min.X), int(card.Min.Y), int(card.Max.X), int(card.Max.Y))

	// Фон карточки. Для размытия фоном служит само размытое содержимое.
	if m.tool != annotate.ToolBlur {
		bg := cardBackground(m.color)
		paint.FillShape(gtx.Ops, bg, clip.UniformRRect(r, int(8/m.scaleX)).Op(gtx.Ops))
	}

	// Образец не должен вылезать за карточку.
	cl := clip.Rect(r).Push(gtx.Ops)
	for _, s := range m.previewShapes(card, w) {
		s.Draw(gtx.Ops, m.env)
	}
	cl.Pop()

	strokeRect(gtx, r.Inset(-1), color.NRGBA{A: 170}, 1.5/m.scaleX)
	strokeRect(gtx, r, color.NRGBA{R: 255, G: 255, B: 255, A: 230}, 1.5/m.scaleX)
}

// previewCard выбирает положение и размер карточки. Размер растёт с толщиной,
// чтобы крупная стрелка или маркер помещались; положение со смещением от
// курсора, с переходом на другую сторону у края выделения.
func (m *monitor) previewCard(w float32) cardRect {
	side := float32(math.Min(math.Max(float64(52+w*5), 64), 210))
	if m.tool == annotate.ToolBlur {
		side = 54
	}
	side /= m.scaleX
	gap := 26 / m.scaleX

	c := m.hover
	x, y := c.X+gap, c.Y+gap
	if x+side > float32(m.sel.Max.X) {
		x = c.X - gap - side
	}
	if y+side > float32(m.sel.Max.Y) {
		y = c.Y - gap - side
	}
	return cardRect{Min: f32.Pt(x, y), Max: f32.Pt(x+side, y+side)}
}

// previewShapes возвращает образец инструмента, вписанный в карточку.
func (m *monitor) previewShapes(card cardRect, w float32) []annotate.Shape {
	pad := float32(math.Max(float64(w*0.8), 12))
	if m.tool == annotate.ToolArrow {
		pad = float32(math.Max(float64(w*1.3), 14))
	}
	x0, y0, x1, y1 := card.Min.X+pad, card.Min.Y+pad, card.Max.X-pad, card.Max.Y-pad
	cy := (y0 + y1) / 2
	mk := func(pts ...f32.Point) []annotate.Shape {
		return []annotate.Shape{{Tool: m.tool, Color: m.color, Width: w, Pts: pts}}
	}

	switch m.tool {
	case annotate.ToolArrow, annotate.ToolLine:
		return mk(f32.Pt(x0, y1), f32.Pt(x1, y0))
	case annotate.ToolRect:
		return mk(f32.Pt(x0, y0), f32.Pt(x1, y1))
	case annotate.ToolPen:
		return mk(f32.Pt(x0, cy+(y1-cy)*0.6), f32.Pt(x0+(x1-x0)*0.25, y0+(y1-y0)*0.2),
			f32.Pt(x0+(x1-x0)*0.5, cy+(y1-cy)*0.6), f32.Pt(x0+(x1-x0)*0.75, y0+(y1-y0)*0.2),
			f32.Pt(x1, cy))
	case annotate.ToolMarker:
		return mk(f32.Pt(x0, cy), f32.Pt(x1, cy))
	case annotate.ToolBlur:
		return mk(card.Min, card.Max)
	case annotate.ToolText:
		return []annotate.Shape{{Tool: annotate.ToolText, Color: m.color, Width: w,
			Pts: []f32.Point{f32.Pt(card.Min.X+8/m.scaleX, cy-annotate.TextSize(w)*0.6)}, Text: "Aa"}}
	}
	return nil
}

// cardBackground выбирает фон карточки, на котором виден цвет инструмента:
// тёмный для светлых цветов и светлый для тёмных.
func cardBackground(c color.NRGBA) color.NRGBA {
	lum := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	if lum > 150 {
		return color.NRGBA{R: 40, G: 42, B: 46, A: 240}
	}
	return color.NRGBA{R: 236, G: 238, B: 242, A: 240}
}

// strokeRect рисует контур прямоугольника толщиной width (пиксели кадра).
func strokeRect(gtx layout.Context, r image.Rectangle, c color.NRGBA, width float32) {
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Max.Y)))
	p.LineTo(f32.Pt(float32(r.Min.X), float32(r.Max.Y)))
	p.Close()
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}
