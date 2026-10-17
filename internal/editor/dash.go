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

// dashedRect рисует пунктирную рамку вокруг r: штрих длиной dash, пауза gap, толщина
// width. Рамка целиком лежит снаружи r, внутрь области не заходит ни пикселя. Узор идёт
// непрерывно по всему периметру, а не начинается заново на каждой стороне.
//
// Штрихи — целые прямоугольники, а не обводка пути: Gio при обводке сотен коротких
// отрезков подмешивает к соседним пикселям +1…+2 уровня яркости вертикальными
// полосами, и они видны внутри области на ровном фоне.
func dashedRect(ops *op.Ops, r image.Rectangle, c color.NRGBA, width, dash, gap float32) {
	if dash <= 0 || gap < 0 {
		return
	}
	w := max(1, int(math.Round(float64(width))))
	h := float32(w) / 2
	// Ломаная по середине полосы рамки: на полтолщины дальше от области.
	x0, y0 := float32(r.Min.X)-h, float32(r.Min.Y)-h
	x1, y1 := float32(r.Max.X)+h, float32(r.Max.Y)+h
	pts := []f32.Point{f32.Pt(x0, y0), f32.Pt(x1, y0), f32.Pt(x1, y1), f32.Pt(x0, y1), f32.Pt(x0, y0)}

	rects := dashRects(r, pts, w, dash, gap)
	for _, d := range rects {
		paint.FillShape(ops, c, clip.Rect(d).Op())
	}
}

// dashRects раскладывает штрихи вдоль ломаной pts (середина полосы рамки вокруг r,
// по часовой стрелке от левого верхнего угла) и возвращает их прямоугольниками
// толщиной w, лежащими вплотную снаружи r.
func dashRects(r image.Rectangle, pts []f32.Point, w int, dash, gap float32) []image.Rectangle {
	var out []image.Rectangle
	period := dash + gap
	rnd := func(v float32) int { return int(math.Round(float64(v))) }
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
			if to <= from {
				continue
			}
			p, q := at(from), at(to)
			var d image.Rectangle
			switch i {
			case 0: // верх
				d = image.Rect(rnd(p.X), r.Min.Y-w, rnd(q.X), r.Min.Y)
			case 1: // право
				d = image.Rect(r.Max.X, rnd(p.Y), r.Max.X+w, rnd(q.Y))
			case 2: // низ: идём справа налево
				d = image.Rect(rnd(q.X), r.Max.Y, rnd(p.X), r.Max.Y+w)
			default: // лево: идём снизу вверх
				d = image.Rect(r.Min.X-w, rnd(q.Y), r.Min.X, rnd(p.Y))
			}
			if !d.Empty() {
				out = append(out, d)
			}
		}
		travelled += length
	}
	return out
}
