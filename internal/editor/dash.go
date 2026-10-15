package editor

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// dashedRect рисует пунктирную рамку: штрих длиной dash, пауза gap, толщина width.
// Узор идёт непрерывно по всему периметру, а не начинается заново на каждой стороне.
func dashedRect(ops *op.Ops, r image.Rectangle, c color.NRGBA, width, dash, gap float32) {
	pts := []f32.Point{
		f32.Pt(float32(r.Min.X), float32(r.Min.Y)), f32.Pt(float32(r.Max.X), float32(r.Min.Y)),
		f32.Pt(float32(r.Max.X), float32(r.Max.Y)), f32.Pt(float32(r.Min.X), float32(r.Max.Y)),
		f32.Pt(float32(r.Min.X), float32(r.Min.Y)),
	}
	var p clip.Path
	p.Begin(ops)
	dashPolyline(&p, pts, dash, gap)
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

// dashPolyline добавляет в путь штрихи вдоль ломаной с непрерывным узором.
func dashPolyline(p *clip.Path, pts []f32.Point, dash, gap float32) {
	if dash <= 0 || gap < 0 {
		return
	}
	period := dash + gap
	var travelled float32 // пройдено вдоль ломаной до начала текущего отрезка
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		length := float32(math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y)))
		if length == 0 {
			continue
		}
		dir := f32.Pt((b.X-a.X)/length, (b.Y-a.Y)/length)
		at := func(d float32) f32.Point { return f32.Pt(a.X+dir.X*d, a.Y+dir.Y*d) }

		// Начало штриха, который мог начаться на предыдущем отрезке.
		start := -float32(math.Mod(float64(travelled), float64(period)))
		for s := start; s < length; s += period {
			from, to := max(s, 0), min(s+dash, length)
			if to > from {
				p.MoveTo(at(from))
				p.LineTo(at(to))
			}
		}
		travelled += length
	}
}
