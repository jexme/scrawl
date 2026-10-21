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

// dashedRect draws a dashed border around r: dash length dash, gap gap, thickness width.
// The border lies entirely outside r, not a single pixel goes inside the area. The pattern
// runs continuously around the whole perimeter instead of starting over on each side.
//
// The dashes are whole rectangles, not a stroked path: when Gio strokes hundreds of short
// segments, it mixes +1..+2 brightness levels into neighboring pixels in vertical stripes,
// and they are visible inside the area on a flat background.
func dashedRect(ops *op.Ops, r image.Rectangle, c color.NRGBA, width, dash, gap float32) {
	if dash <= 0 || gap < 0 {
		return
	}
	w := max(1, int(math.Round(float64(width))))
	h := float32(w) / 2
	// A polyline along the middle of the border band: half a thickness farther from the area.
	x0, y0 := float32(r.Min.X)-h, float32(r.Min.Y)-h
	x1, y1 := float32(r.Max.X)+h, float32(r.Max.Y)+h
	pts := []f32.Point{f32.Pt(x0, y0), f32.Pt(x1, y0), f32.Pt(x1, y1), f32.Pt(x0, y1), f32.Pt(x0, y0)}

	rects := dashRects(r, pts, w, dash, gap)
	for _, d := range rects {
		paint.FillShape(ops, c, clip.Rect(d).Op())
	}
}

// dashRects lays dashes along polyline pts (the middle of the border band around r,
// clockwise from the top-left corner) and returns them as rectangles of
// thickness w lying flush against the outside of r.
func dashRects(r image.Rectangle, pts []f32.Point, w int, dash, gap float32) []image.Rectangle {
	var out []image.Rectangle
	period := dash + gap
	rnd := func(v float32) int { return int(math.Round(float64(v))) }
	var travelled float32 // distance traveled along the polyline up to the start of the current segment
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		length := float32(math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y)))
		if length == 0 {
			continue
		}
		dir := f32.Pt((b.X-a.X)/length, (b.Y-a.Y)/length)
		at := func(d float32) f32.Point { return f32.Pt(a.X+dir.X*d, a.Y+dir.Y*d) }

		// Start of a dash that may have begun on the previous segment.
		start := -float32(math.Mod(float64(travelled), float64(period)))
		for s := start; s < length; s += period {
			from, to := max(s, 0), min(s+dash, length)
			if to <= from {
				continue
			}
			p, q := at(from), at(to)
			var d image.Rectangle
			switch i {
			case 0: // top
				d = image.Rect(rnd(p.X), r.Min.Y-w, rnd(q.X), r.Min.Y)
			case 1: // right
				d = image.Rect(r.Max.X, rnd(p.Y), r.Max.X+w, rnd(q.Y))
			case 2: // bottom: going right to left
				d = image.Rect(rnd(q.X), r.Max.Y, rnd(p.X), r.Max.Y+w)
			default: // left: going bottom to top
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
