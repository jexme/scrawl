package editor

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// Лупа показывает увеличенные пиксели вокруг двигаемого угла или стороны рамки,
// чтобы выровнять рамку с точностью до пикселя.
const (
	loupeZoom   = 6  // во сколько раз увеличиваются пиксели кадра
	loupeCells  = 32 // сколько пикселей кадра помещается в лупе по каждой оси (чётное: фокус лежит на линии сетки)
	loupeGridAt = 6  // сетка между пикселями рисуется, если увеличение не меньше этого
)

var (
	loupeBG      = color.NRGBA{R: 24, G: 24, B: 28, A: 255}
	loupeGrid    = color.NRGBA{A: 56}
	loupeEdgeIn  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	loupeEdgeOut = color.NRGBA{A: 220}
)

// loupeFocus возвращает точку кадра (на сетке пикселей), к которой прилипает
// двигаемая сторона рамки: в её центре лупа и рисуется. Для создаваемой рамки это
// угол под курсором, для изменяемой стороны сама сторона, по другой оси курсор.
func loupeFocus(sel image.Rectangle, e edge, creating bool, hover f32.Point, bounds image.Rectangle) f32.Point {
	round := func(v float32, lo, hi int) float32 {
		return float32(clampInt(int(math.Round(float64(v))), lo, hi))
	}
	if creating {
		// Создаваемая рамка берёт целую часть координаты, как rectOf.
		return f32.Pt(
			float32(clampInt(int(hover.X), bounds.Min.X, bounds.Max.X)),
			float32(clampInt(int(hover.Y), bounds.Min.Y, bounds.Max.Y)),
		)
	}
	f := f32.Pt(round(hover.X, bounds.Min.X, bounds.Max.X), round(hover.Y, bounds.Min.Y, bounds.Max.Y))
	switch e.hx {
	case -1:
		f.X = float32(sel.Min.X)
	case 1:
		f.X = float32(sel.Max.X)
	}
	switch e.hy {
	case -1:
		f.Y = float32(sel.Min.Y)
	case 1:
		f.Y = float32(sel.Max.Y)
	}
	return f
}

// loupeDir возвращает, в какую сторону от курсора ставить лупу (по каждой оси -1, 0 или 1):
// наружу от рамки, чтобы лупа не закрывала то, что выделяют. Для новой рамки это сторона,
// куда тянут от точки нажатия, для изменяемой сторона двигаемого края.
func loupeDir(e edge, creating bool, start, hover f32.Point) image.Point {
	if creating {
		d := image.Pt(1, 1)
		if hover.X < start.X {
			d.X = -1
		}
		if hover.Y < start.Y {
			d.Y = -1
		}
		return d
	}
	return image.Pt(e.hx, e.hy)
}

// loupePlacement ставит квадрат лупы со стороной side рядом с курсором cur в стороне dir
// (по оси 0 лупа центрируется на курсоре). Не помещается с этой стороны, ставится
// с противоположной, потом прижимается к окну.
func loupePlacement(cur image.Point, side, gap int, win, dir image.Point) image.Rectangle {
	place := func(c, w, d int) int {
		switch {
		case d > 0:
			if c+gap+side <= w {
				return c + gap
			}
			return c - gap - side
		case d < 0:
			if c-gap-side >= 0 {
				return c - gap - side
			}
			return c + gap
		}
		return c - side/2
	}
	x := clampInt(place(cur.X, win.X, dir.X), 0, max(win.X-side, 0))
	y := clampInt(place(cur.Y, win.Y, dir.Y), 0, max(win.Y-side, 0))
	return image.Rect(x, y, x+side, y+side)
}

// loupeAiming: области ещё нет, курсор над окном, и лупа помогает выбрать стартовую точку.
func (m *monitor) loupeAiming() bool {
	return !m.hasSel && !m.dragging && m.adjust == adjustNone && m.hoverIn
}

// loupeActive: лупа нужна, пока целятся в первую точку и пока тянут рамку (новую или
// готовую), но не при перемещении области целиком и не при рисовании.
func (m *monitor) loupeActive() bool {
	return m.loupeAiming() || m.adjust == adjustResize || m.dragging
}

// layoutLoupe рисует лупу поверх всего остального. Рисуется в пикселях окна.
func (m *monitor) layoutLoupe(gtx layout.Context, win image.Point, sx, sy float32) {
	if !m.loupeActive() {
		return
	}
	bounds := m.frame.Img.Bounds()
	aiming := m.loupeAiming()
	creating := m.dragging || aiming
	start := m.selStart
	if aiming {
		start = m.hover
	}
	focus := loupeFocus(m.sel, m.adjEdge, creating, m.hover, bounds)

	zoom := loupeZoom * sx
	side := int(math.Round(float64(loupeCells * zoom)))
	cur := image.Pt(int(m.hover.X*sx), int(m.hover.Y*sy))
	dir := loupeDir(m.adjEdge, creating, start, m.hover)
	box := loupePlacement(cur, side, gtx.Dp(28), win, dir)
	cx, cy := float32(box.Min.X+box.Dx()/2), float32(box.Min.Y+box.Dy()/2)

	if m.loupeOp == (paint.ImageOp{}) {
		m.loupeOp = paint.NewImageOp(m.frame.Img)
		m.loupeOp.Filter = paint.FilterNearest // пиксели остаются квадратами, без сглаживания
	}

	// Тонкая тёмная кайма и фон под кадром (за краем экрана кадра нет).
	paint.FillShape(gtx.Ops, loupeEdgeOut, clip.Rect(box.Inset(-2)).Op())
	paint.FillShape(gtx.Ops, loupeBG, clip.Rect(box).Op())

	boxClip := clip.Rect(box).Push(gtx.Ops)
	// Дальше всё в пространстве кадра: фокус в центре лупы, пиксель кадра = zoom пикселей окна.
	tr := op.Affine(f32.Affine2D{}.
		Offset(f32.Pt(-focus.X, -focus.Y)).
		Scale(f32.Pt(0, 0), f32.Pt(zoom, zoom)).
		Offset(f32.Pt(cx, cy))).Push(gtx.Ops)

	imgClip := clip.Rect(bounds).Push(gtx.Ops)
	m.loupeOp.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	// Вне выделения затемнено так же, как на экране: видно, какие пиксели попадут в снимок.
	// Пока рамки нет, пиксели не затемняем: так их проще разглядеть.
	r := m.sel
	if !aiming {
		shade := shadeColor(m.store.Get().DimPercent)
		for _, side := range []image.Rectangle{
			{Min: bounds.Min, Max: image.Pt(bounds.Max.X, r.Min.Y)},
			{Min: image.Pt(bounds.Min.X, r.Max.Y), Max: bounds.Max},
			{Min: image.Pt(bounds.Min.X, r.Min.Y), Max: image.Pt(r.Min.X, r.Max.Y)},
			{Min: image.Pt(r.Max.X, r.Min.Y), Max: image.Pt(bounds.Max.X, r.Max.Y)},
		} {
			paint.FillShape(gtx.Ops, shade, clip.Rect(side).Op())
		}
	}
	imgClip.Pop()

	// Сетка между пикселями.
	if zoom >= loupeGridAt {
		half := float32(loupeCells/2 + 1)
		x0, x1 := float32(math.Floor(float64(focus.X-half))), float32(math.Ceil(float64(focus.X+half)))
		y0, y1 := float32(math.Floor(float64(focus.Y-half))), float32(math.Ceil(float64(focus.Y+half)))
		var p clip.Path
		p.Begin(gtx.Ops)
		for x := x0; x <= x1; x++ {
			p.MoveTo(f32.Pt(x, y0))
			p.LineTo(f32.Pt(x, y1))
		}
		for y := y0; y <= y1; y++ {
			p.MoveTo(f32.Pt(x0, y))
			p.LineTo(f32.Pt(x1, y))
		}
		paint.FillShape(gtx.Ops, loupeGrid, clip.Stroke{Path: p.End(), Width: 1 / zoom}.Op())
	}

	if !aiming {
		// Рамка выделения: белая линия на тёмной подложке видна на любом фоне.
		var rect clip.Path
		rect.Begin(gtx.Ops)
		rect.MoveTo(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
		rect.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
		rect.LineTo(f32.Pt(float32(r.Max.X), float32(r.Max.Y)))
		rect.LineTo(f32.Pt(float32(r.Min.X), float32(r.Max.Y)))
		rect.Close()
		spec := rect.End()
		paint.FillShape(gtx.Ops, loupeEdgeOut, clip.Stroke{Path: spec, Width: 4 / zoom}.Op())
		paint.FillShape(gtx.Ops, loupeEdgeIn, clip.Stroke{Path: spec, Width: 2 / zoom}.Op())
	}
	tr.Pop()

	// Крестик в центре: сюда встанет сторона рамки.
	const arm = 6
	var cross clip.Path
	cross.Begin(gtx.Ops)
	cross.MoveTo(f32.Pt(cx-arm, cy))
	cross.LineTo(f32.Pt(cx+arm, cy))
	cross.MoveTo(f32.Pt(cx, cy-arm))
	cross.LineTo(f32.Pt(cx, cy+arm))
	cspec := cross.End()
	paint.FillShape(gtx.Ops, loupeEdgeOut, clip.Stroke{Path: cspec, Width: 3}.Op())
	paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 60, B: 90, A: 255}, clip.Stroke{Path: cspec, Width: 1}.Op())
	boxClip.Pop()
}
