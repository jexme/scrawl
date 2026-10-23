// Package annotate describes annotations over a screenshot (arrows, shapes, text)
// and draws them in Gio. The same code is used for on-screen display and for
// PNG export, so the result matches what the user saw.
//
// All coordinates and sizes are in frame pixels.
package annotate

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/fonts"
	"github.com/jexme/scrawl/internal/i18n"
)

// Tool is a drawing tool.
type Tool int

const (
	ToolSelect Tool = iota // area selection
	ToolPen
	ToolLine
	ToolArrow
	ToolRect
	ToolMarker
	ToolBlur
	ToolText
	ToolStamp // numbered stamp: a circle with a digit, numbers go in order
)

// Palette is the colors in the toolbar.
var Palette = []color.NRGBA{
	{R: 229, G: 57, B: 53, A: 255},   // red
	{R: 251, G: 140, B: 0, A: 255},   // orange
	{R: 253, G: 216, B: 53, A: 255},  // yellow
	{R: 67, G: 160, B: 71, A: 255},   // green
	{R: 30, G: 136, B: 229, A: 255},  // blue
	{R: 142, G: 36, B: 170, A: 255},  // purple
	{R: 255, G: 255, B: 255, A: 255}, // white
	{R: 20, G: 20, B: 20, A: 255},    // black
}

const (
	DefaultWidth = 4
	MinWidth     = 1
	MaxWidth     = 30
)

// DefaultWidthOf returns the default thickness of a tool: for blur it is the level.
func DefaultWidthOf(t Tool) float32 {
	if t == ToolBlur {
		return DefaultBlurLevel
	}
	return DefaultWidth
}

// MaxWidthOf returns the largest thickness of a tool: for blur it is the last level.
func MaxWidthOf(t Tool) float32 {
	if t == ToolBlur {
		return MaxBlurLevel
	}
	return MaxWidth
}

// Shape is a single annotation.
type Shape struct {
	Tool  Tool
	Color color.NRGBA
	Width float32     // line thickness, frame pixels
	Pts   []f32.Point // key points
	Text  string      // only for ToolText and ToolStamp (the number)
	// Alpha is the marker opacity, 1-255; 0 means DefaultMarkerAlpha.
	Alpha uint8
}

// DefaultMarkerAlpha is the marker opacity when not set otherwise (43%).
const DefaultMarkerAlpha = 110

// TextSize returns the font size of a text annotation in frame pixels.
func TextSize(width float32) float32 { return 12 + width*3 }

// StampRadius returns the radius of a numbered stamp from the tool "thickness".
func StampRadius(width float32) float32 { return 12 + width*2.5 }

// NewShaper creates a font shaper for text annotations.
func NewShaper() *text.Shaper {
	return text.NewShaper(text.WithCollection(fonts.Collection(i18n.Language())))
}

// Font is the font of text annotations.
var Font = gofont.Collection()[0].Font

// Degenerate reports that the shape is too small to be kept.
func (s Shape) Degenerate() bool {
	switch s.Tool {
	case ToolText:
		return s.Text == ""
	case ToolBlur:
		if len(s.Pts) < 2 {
			return true
		}
		r := RectOf(s.Pts[0], s.Pts[1])
		return r.Dx() < 4 || r.Dy() < 4
	case ToolPen, ToolStamp:
		return len(s.Pts) < 1
	case ToolMarker:
		// A freehand stroke is judged by having moved at all, not by the distance between its
		// first two points: they are close together when the hand starts slowly, and the whole
		// long stroke used to vanish on release. A click without moving leaves no mark.
		return len(s.Pts) < 2
	default:
		if len(s.Pts) < 2 {
			return true
		}
		return dist(s.Pts[0], s.Pts[1]) < 3
	}
}

// Draw draws the shape. The shape's coordinates are treated as coordinates in the ops space.
func (s Shape) Draw(ops *op.Ops, env Env) {
	if len(s.Pts) == 0 {
		return
	}
	switch s.Tool {
	case ToolPen:
		stroke(ops, s.Color, s.Width, s.Pts...)
	case ToolLine:
		if len(s.Pts) >= 2 {
			stroke(ops, s.Color, s.Width, s.Pts[0], s.Pts[1])
		}
	case ToolArrow:
		if len(s.Pts) >= 2 {
			drawArrow(ops, s.Color, s.Width, s.Pts[0], s.Pts[1])
		}
	case ToolRect:
		if len(s.Pts) >= 2 {
			a, b := s.Pts[0], s.Pts[1]
			strokeClosed(ops, s.Color, s.Width,
				f32.Pt(a.X, a.Y), f32.Pt(b.X, a.Y), f32.Pt(b.X, b.Y), f32.Pt(a.X, b.Y))
		}
	case ToolMarker:
		c := s.Color
		c.A = s.Alpha
		if c.A == 0 {
			c.A = DefaultMarkerAlpha
		}
		stroke(ops, c, s.Width*4, s.Pts...)
	case ToolBlur:
		s.drawBlur(ops, env)
	case ToolText:
		drawText(ops, env.Shaper, s.Color, s.Width, s.Pts[0], s.Text)
	case ToolStamp:
		drawStamp(ops, env.Shaper, s.Color, s.Width, s.Pts[0], s.Text)
	}
}

// DrawAll draws all shapes in order.
func DrawAll(ops *op.Ops, env Env, shapes []Shape) {
	for _, s := range shapes {
		s.Draw(ops, env)
	}
}

func dist(a, b f32.Point) float32 {
	return float32(math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y)))
}

func stroke(ops *op.Ops, c color.NRGBA, width float32, pts ...f32.Point) {
	if len(pts) == 0 {
		return
	}
	if len(pts) == 1 {
		// A point: a short segment so the stroke is visible.
		pts = []f32.Point{pts[0], pts[0].Add(f32.Pt(0.01, 0))}
	}
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(pts[0])
	for _, pt := range pts[1:] {
		p.LineTo(pt)
	}
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

func strokeClosed(ops *op.Ops, c color.NRGBA, width float32, pts ...f32.Point) {
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(pts[0])
	for _, pt := range pts[1:] {
		p.LineTo(pt)
	}
	p.Close()
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

// drawArrow draws a line with a triangular arrowhead at the point to.
func drawArrow(ops *op.Ops, c color.NRGBA, width float32, from, to f32.Point) {
	d := to.Sub(from)
	length := dist(from, to)
	if length < 1 {
		return
	}
	dir := f32.Pt(d.X/length, d.Y/length)
	normal := f32.Pt(-dir.Y, dir.X)

	head := float32(math.Max(float64(width*4), 14))
	if head > length {
		head = length
	}
	half := head * 0.5
	base := to.Sub(dir.Mul(head))

	// The shaft ends inside the arrowhead so there is no gap.
	stroke(ops, c, width, from, base.Add(dir.Mul(head*0.3)))

	var p clip.Path
	p.Begin(ops)
	p.MoveTo(to)
	p.LineTo(base.Add(normal.Mul(half)))
	p.LineTo(base.Sub(normal.Mul(half)))
	p.Close()
	paint.FillShape(ops, c, clip.Outline{Path: p.End()}.Op())
}

// drawText draws text in frame space: the metric is unit, so
// the font size is given in frame pixels regardless of the window scale.
func drawText(ops *op.Ops, shaper *text.Shaper, c color.NRGBA, width float32, at f32.Point, txt string) {
	if txt == "" || shaper == nil {
		return
	}
	defer op.Offset(image.Pt(int(at.X), int(at.Y))).Push(ops).Pop()
	gtx := layout.Context{
		Ops:         ops,
		Constraints: layout.Constraints{Max: image.Pt(1<<14, 1<<14)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	rec := op.Record(ops)
	paint.ColorOp{Color: c}.Add(ops)
	mat := rec.Stop()
	widget.Label{}.Layout(gtx, shaper, Font, unit.Sp(TextSize(width)), txt, mat)
}

// onColor returns a text color readable on the background c: dark on light, white on dark.
func onColor(c color.NRGBA) color.NRGBA {
	if lum := 0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B); lum > 150 {
		return color.NRGBA{R: 20, G: 20, B: 20, A: 255}
	}
	return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}

// stampTextSize picks the digit size in a stamp: the longer the number, the smaller.
func stampTextSize(r float32, digits int) float32 {
	switch {
	case digits <= 1:
		return r * 1.25
	case digits == 2:
		return r * 1.0
	case digits == 3:
		return r * 0.8
	}
	return r * 0.64
}

// stampFont is the stamp digit font: bold, so the digits are readable on any background.
var stampFont = font.Font{Typeface: Font.Typeface, Weight: font.Bold}

// drawStamp draws a stamp: a colored circle in a rim of the digit color (light on dark,
// dark on light), a soft shadow below and a bold number exactly in the middle.
func drawStamp(ops *op.Ops, shaper *text.Shaper, c color.NRGBA, width float32, at f32.Point, num string) {
	r := StampRadius(width)
	fg := onColor(c)

	circle := func(radius, dy float32) clip.Op {
		return clip.Ellipse(image.Rect(
			int(math.Round(float64(at.X-radius))), int(math.Round(float64(at.Y+dy-radius))),
			int(math.Round(float64(at.X+radius))), int(math.Round(float64(at.Y+dy+radius))))).Op(ops)
	}
	// Shadow: several translucent circles nested in each other, shifted slightly down.
	for _, sh := range []struct {
		grow float32
		a    uint8
	}{{0.16, 14}, {0.10, 18}, {0.05, 22}} {
		paint.FillShape(ops, color.NRGBA{A: sh.a}, circle(r*(1+sh.grow), r*0.08))
	}
	paint.FillShape(ops, fg, circle(r, 0))
	paint.FillShape(ops, c, circle(r*0.84, 0))

	if num == "" || shaper == nil {
		return
	}
	gtx := layout.Context{
		Ops:         ops,
		Constraints: layout.Constraints{Max: image.Pt(1<<14, 1<<14)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	size := stampTextSize(r, len(num))
	rec := op.Record(ops)
	paint.ColorOp{Color: fg}.Add(ops)
	mat := rec.Stop()
	rec = op.Record(ops)
	dims := widget.Label{MaxLines: 1}.Layout(gtx, shaper, stampFont, unit.Sp(size), num, mat)
	label := rec.Stop()
	// A digit sits on the baseline: put the middle of the digits (half their height above the
	// line) at the circle center.
	baseline := float32(dims.Size.Y - dims.Baseline)
	top := at.Y - baseline + size*digitHeight/2
	defer op.Offset(image.Pt(int(math.Round(float64(at.X)))-dims.Size.X/2, int(math.Round(float64(top))))).Push(ops).Pop()
	label.Add(ops)
}

// digitHeight is the height of a Go font digit as a fraction of the font size.
const digitHeight = 0.72
