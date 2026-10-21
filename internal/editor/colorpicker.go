package editor

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/jexme/scrawl/internal/settings"
)

// hsv is a colour as hue (0-360), saturation and value (0-1): convenient for drawing the palette.
type hsv struct{ H, S, V float32 }

func (c hsv) rgb() color.NRGBA {
	h := math.Mod(float64(c.H), 360)
	if h < 0 {
		h += 360
	}
	s, v := float64(c.S), float64(c.V)
	ch := v * s
	x := ch * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - ch
	var r, g, b float64
	switch int(h / 60) {
	case 0:
		r, g, b = ch, x, 0
	case 1:
		r, g, b = x, ch, 0
	case 2:
		r, g, b = 0, ch, x
	case 3:
		r, g, b = 0, x, ch
	case 4:
		r, g, b = x, 0, ch
	default:
		r, g, b = ch, 0, x
	}
	to8 := func(f float64) uint8 { return uint8(math.Round((f + m) * 255)) }
	return color.NRGBA{R: to8(r), G: to8(g), B: to8(b), A: 255}
}

func hsvOf(c color.NRGBA) hsv {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := mx - mn
	var h float64
	switch {
	case d == 0:
		h = 0
	case mx == r:
		h = math.Mod((g-b)/d, 6)
	case mx == g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	var s float64
	if mx > 0 {
		s = d / mx
	}
	return hsv{H: float32(h), S: float32(s), V: float32(mx)}
}

// picker is the small window for choosing a custom colour: a saturation/value square and a hue strip.
type picker struct {
	open     bool
	hsv      hsv
	sv, hue  int // input area tags
	area     int // backdrop tag: clicks on the window do not reach the canvas
	picked   bool
	released bool // the choice is finished: the colour can be remembered
}

var hueStops = [...]color.NRGBA{
	{R: 255, A: 255}, {R: 255, G: 255, A: 255}, {G: 255, A: 255},
	{G: 255, B: 255, A: 255}, {B: 255, A: 255}, {R: 255, B: 255, A: 255}, {R: 255, A: 255},
}

// pickerSize returns the sizes of the window and its parts in window pixels.
func pickerSize(gtx layout.Context) (w, h, pad, svH, hueH, rowH int) {
	pad = gtx.Dp(10)
	w = gtx.Dp(224)
	svH = gtx.Dp(130)
	hueH = gtx.Dp(16)
	rowH = gtx.Dp(26)
	h = pad + svH + pad/2 + hueH + pad/2 + rowH + pad
	return
}

// pickerOrigin returns the top-left corner of the window: below the toolbar, or above
// it if there is no room, centred on the custom colour button (anchorX).
func pickerOrigin(gtx layout.Context, win image.Point, bar image.Rectangle, anchorX int) image.Point {
	w, h, _, _, _, _ := pickerSize(gtx)
	margin := gtx.Dp(8)
	x := clampInt(anchorX-w/2, margin, max(win.X-w-margin, margin))
	y := bar.Max.Y + margin
	if y+h > win.Y-margin {
		y = bar.Min.Y - margin - h
	}
	return image.Pt(x, max(y, margin))
}

// layoutPicker draws the window below the toolbar (above it if there is no room),
// centred on the custom colour button at anchorX. Changes the colour as the mouse is dragged.
func (m *monitor) layoutPicker(gtx layout.Context, win image.Point, bar image.Rectangle, anchorX int) {
	p := &m.pick
	if !p.open {
		return
	}
	w, h, pad, svH, hueH, rowH := pickerSize(gtx)
	defer op.Offset(pickerOrigin(gtx, win, bar, anchorX)).Push(gtx.Ops).Pop()

	// The backdrop and an area that stops clicks from reaching the canvas.
	bg := image.Rect(0, 0, w, h)
	paint.FillShape(gtx.Ops, barBG, clip.UniformRRect(bg, gtx.Dp(8)).Op(gtx.Ops))
	area := clip.Rect(bg).Push(gtx.Ops)
	event.Op(gtx.Ops, &p.area)
	area.Pop()
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &p.area, Kinds: pointer.Press | pointer.Release | pointer.Drag}); !ok {
			break
		}
	}

	inner := w - 2*pad
	svRect := image.Rect(pad, pad, pad+inner, pad+svH)
	hueRect := image.Rect(pad, svRect.Max.Y+pad/2, pad+inner, svRect.Max.Y+pad/2+hueH)

	// Events: press and drag in the square and on the strip.
	m.pickerInput(gtx, svRect, &p.sv, func(pt f32.Point) {
		p.hsv.S = clamp01(pt.X / float32(svRect.Dx()))
		p.hsv.V = 1 - clamp01(pt.Y/float32(svRect.Dy()))
	})
	m.pickerInput(gtx, hueRect, &p.hue, func(pt f32.Point) {
		p.hsv.H = clamp01(pt.X/float32(hueRect.Dx())) * 360
	})

	// The square: white on the left, the pure hue on the right; on top, transparent at the
	// top and black at the bottom.
	pure := hsv{H: p.hsv.H, S: 1, V: 1}.rgb()
	func() {
		defer clip.UniformRRect(svRect, gtx.Dp(4)).Push(gtx.Ops).Pop()
		paint.LinearGradientOp{
			Stop1: f32.Pt(float32(svRect.Min.X), 0), Color1: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
			Stop2: f32.Pt(float32(svRect.Max.X), 0), Color2: pure,
		}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		paint.LinearGradientOp{
			Stop1: f32.Pt(0, float32(svRect.Min.Y)), Color1: color.NRGBA{A: 0},
			Stop2: f32.Pt(0, float32(svRect.Max.Y)), Color2: color.NRGBA{A: 255},
		}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}()
	// A circle at the selected point.
	cx := float32(svRect.Min.X) + p.hsv.S*float32(svRect.Dx())
	cy := float32(svRect.Min.Y) + (1-p.hsv.V)*float32(svRect.Dy())
	ring(gtx, cx, cy, float32(gtx.Dp(6)))

	// The hue strip made of six gradient pieces.
	func() {
		defer clip.UniformRRect(hueRect, gtx.Dp(4)).Push(gtx.Ops).Pop()
		seg := float32(hueRect.Dx()) / 6
		for i := 0; i < 6; i++ {
			x0 := float32(hueRect.Min.X) + float32(i)*seg
			part := clip.Rect(image.Rect(int(x0), hueRect.Min.Y, int(x0+seg)+1, hueRect.Max.Y)).Push(gtx.Ops)
			paint.LinearGradientOp{
				Stop1: f32.Pt(x0, 0), Color1: hueStops[i],
				Stop2: f32.Pt(x0+seg, 0), Color2: hueStops[i+1],
			}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			part.Pop()
		}
	}()
	hx := float32(hueRect.Min.X) + p.hsv.H/360*float32(hueRect.Dx())
	mark := clip.Rect(image.Rect(int(hx)-gtx.Dp(2), hueRect.Min.Y-gtx.Dp(2), int(hx)+gtx.Dp(2), hueRect.Max.Y+gtx.Dp(2)))
	paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, mark.Op())

	// The sample and the colour code.
	cur := p.hsv.rgb()
	rowY := hueRect.Max.Y + pad/2
	chip := image.Rect(pad, rowY, pad+gtx.Dp(44), rowY+rowH)
	paint.FillShape(gtx.Ops, cur, clip.UniformRRect(chip, gtx.Dp(4)).Op(gtx.Ops))
	txt, size := m.label(gtx, settings.FormatColor(cur), 13, iconColor)
	st := op.Offset(image.Pt(chip.Max.X+pad, rowY+(rowH-size.Y)/2)).Push(gtx.Ops)
	txt.Add(gtx.Ops)
	st.Pop()

	if p.released {
		p.released = false
		c := settings.FormatColor(m.customColor)
		_ = m.store.Update(func(s *settings.Settings) { s.CustomColor = c }) // not saved: the colour stays until the end of the screenshot
	}
}

// pickerInput reads press and drag in the rectangle r and passes the point in its own
// coordinates. Every change is applied to the drawing colour immediately.
func (m *monitor) pickerInput(gtx layout.Context, r image.Rectangle, tag event.Tag, set func(f32.Point)) {
	p := &m.pick
	stack := clip.Rect(r).Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	pointer.CursorCrosshair.Add(gtx.Ops)
	stack.Pop()
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Drag | pointer.Release})
		if !ok {
			return
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press, pointer.Drag:
			// The position arrives in window coordinates, not square coordinates: shift it by the square's corner.
			set(e.Position.Sub(f32.Pt(float32(r.Min.X), float32(r.Min.Y))))
			m.applyPicked()
		case pointer.Release:
			p.released = true
		}
	}
}

// applyPicked makes the colour chosen in the window the drawing colour.
func (m *monitor) applyPicked() {
	c := m.pick.hsv.rgb()
	m.color, m.colorPicked = c, true
	m.customColor, m.hasCustom = c, true
}

// ring draws a marker circle: a white ring with a dark outline.
func ring(gtx layout.Context, cx, cy, r float32) {
	for _, s := range []struct {
		r, w float32
		c    color.NRGBA
	}{{r, 3, color.NRGBA{A: 200}}, {r, 1.5, color.NRGBA{R: 255, G: 255, B: 255, A: 255}}} {
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(f32.Pt(cx+s.r, cy))
		path.ArcTo(f32.Pt(cx, cy), f32.Pt(cx, cy), 2*math.Pi)
		paint.FillShape(gtx.Ops, s.c, clip.Stroke{Path: path.End(), Width: s.w}.Op())
	}
}

func clamp01(v float32) float32 { return float32(math.Min(math.Max(float64(v), 0), 1)) }
