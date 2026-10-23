package editor

import (
	"image"
	"math"

	"gioui.org/f32"
)

// The eight directions that Shift snaps a line to: every 45 degrees, clockwise from the
// right (the Y axis points down).
var snapDirs = [8]f32.Point{
	{X: 1, Y: 0}, {X: math.Sqrt2 / 2, Y: math.Sqrt2 / 2}, {X: 0, Y: 1}, {X: -math.Sqrt2 / 2, Y: math.Sqrt2 / 2},
	{X: -1, Y: 0}, {X: -math.Sqrt2 / 2, Y: -math.Sqrt2 / 2}, {X: 0, Y: -1}, {X: math.Sqrt2 / 2, Y: -math.Sqrt2 / 2},
}

// snapAngle returns the point at the same distance from a as p, on the nearest of the eight
// directions (horizontal, vertical, diagonal). The line is shortened if it would leave r.
func snapAngle(a, p f32.Point, r image.Rectangle) f32.Point {
	dx, dy := float64(p.X-a.X), float64(p.Y-a.Y)
	length := math.Hypot(dx, dy)
	if length == 0 {
		return p
	}
	k := int(math.Round(math.Atan2(dy, dx)/(math.Pi/4))+8) % 8
	d := snapDirs[k]
	// The room left in the direction of the line, so the end point stays inside r.
	room := func(pos, lo, hi, dir float32) float64 {
		switch {
		case dir > 1e-6:
			return float64((hi - pos) / dir)
		case dir < -1e-6:
			return float64((pos - lo) / -dir)
		}
		return math.Inf(1)
	}
	length = math.Min(length, room(a.X, float32(r.Min.X), float32(r.Max.X), d.X))
	length = math.Min(length, room(a.Y, float32(r.Min.Y), float32(r.Max.Y), d.Y))
	return f32.Pt(a.X+float32(length)*d.X, a.Y+float32(length)*d.Y)
}

// squareTo returns the corner of a square with the corner a, in the direction of p and with
// the longer of the two sides; the square is cut down if it would leave r.
func squareTo(a, p f32.Point, r image.Rectangle) f32.Point {
	dx, dy := p.X-a.X, p.Y-a.Y
	sx, sy := float32(1), float32(1)
	if dx < 0 {
		sx = -1
	}
	if dy < 0 {
		sy = -1
	}
	side := float32(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))
	roomX, roomY := float32(r.Max.X)-a.X, float32(r.Max.Y)-a.Y
	if sx < 0 {
		roomX = a.X - float32(r.Min.X)
	}
	if sy < 0 {
		roomY = a.Y - float32(r.Min.Y)
	}
	side = float32(math.Min(float64(side), math.Min(float64(roomX), float64(roomY))))
	return f32.Pt(a.X+sx*side, a.Y+sy*side)
}
