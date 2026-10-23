package editor

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"
)

func near(a, b f32.Point) bool {
	return math.Abs(float64(a.X-b.X)) < 1e-3 && math.Abs(float64(a.Y-b.Y)) < 1e-3
}

func TestSnapAngle(t *testing.T) {
	area := image.Rect(0, 0, 400, 300)
	a := f32.Pt(100, 100)
	for _, c := range []struct {
		name string
		p    f32.Point
		want f32.Point
	}{
		{"almost horizontal", f32.Pt(200, 108), f32.Pt(200.3, 100)},
		{"almost vertical, up", f32.Pt(94, 20), f32.Pt(100, 20.2)},
		{"diagonal down-right", f32.Pt(180, 170), f32.Pt(100+75*1.0, 100+75*1.0)},
		{"diagonal up-left", f32.Pt(60, 70), f32.Pt(100-36.0, 100-36.0)},
		{"same point", a, a},
	} {
		got := snapAngle(a, c.p, area)
		if c.name == "same point" {
			if got != c.want {
				t.Errorf("%s: %v", c.name, got)
			}
			continue
		}
		// Horizontal and vertical lines must be exact; diagonals have equal offsets.
		dx, dy := got.X-a.X, got.Y-a.Y
		switch c.name {
		case "almost horizontal":
			if got.Y != a.Y || dx < 99 {
				t.Errorf("%s: %v", c.name, got)
			}
		case "almost vertical, up":
			if got.X != a.X || dy > -79 {
				t.Errorf("%s: %v", c.name, got)
			}
		default:
			if math.Abs(float64(math.Abs(float64(dx))-math.Abs(float64(dy)))) > 1e-3 {
				t.Errorf("%s: not a diagonal: %v", c.name, got)
			}
		}
	}
}

func TestSnapAngleStaysInside(t *testing.T) {
	area := image.Rect(0, 0, 200, 100)
	a := f32.Pt(150, 50)
	got := snapAngle(a, f32.Pt(400, 55), area) // would be 250 px long to the right
	if got.X > 200 || got.Y != 50 {
		t.Errorf("the line leaves the area: %v", got)
	}
	got = snapAngle(a, f32.Pt(260, 160), area) // diagonal down-right, 50 px of room below
	if got.X > 200 || got.Y > 100 || !near(f32.Pt(got.X-a.X, got.Y-a.Y), f32.Pt(50, 50)) {
		t.Errorf("the diagonal is not cut at the edge: %v", got)
	}
}

func TestSquareTo(t *testing.T) {
	area := image.Rect(0, 0, 400, 300)
	a := f32.Pt(100, 100)
	if got := squareTo(a, f32.Pt(220, 150), area); got != f32.Pt(220, 220) {
		t.Errorf("wider than tall: %v", got)
	}
	if got := squareTo(a, f32.Pt(70, 40), area); got != f32.Pt(40, 40) {
		t.Errorf("up-left: %v", got)
	}
	if got := squareTo(a, f32.Pt(390, 130), area); got != f32.Pt(300, 300) { // 200 px of room below
		t.Errorf("cut by the bottom edge: %v", got)
	}
}
