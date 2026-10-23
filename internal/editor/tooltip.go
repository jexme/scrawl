package editor

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/annotate"
)

// tipDelay is how long the cursor must rest on a toolbar button before its tooltip appears.
const tipDelay = 450 * time.Millisecond

// toolTipNames are the tool names for the tooltips, in the order of the toolbar buttons.
var toolTipNames = []string{"Select", "Pen", "Line", "Arrow", "Box", "Marker", "Blur", "Text", "Number"}

// tipText joins a tooltip name and the key that does the same thing.
func tipText(name, key string) string {
	if key == "" {
		return name
	}
	return name + "  ·  " + key
}

// hoverTip offers the tooltip text of a button when the cursor is over it. origin is the
// corner of the row in window pixels, x0 the middle of the button in the row, y its top.
func (m *monitor) hoverTip(hovered bool, origin image.Point, x0, y, btn int, text string) {
	if !hovered {
		return
	}
	m.tipCand = text
	m.tipCandRect = image.Rect(origin.X+x0-btn/2, origin.Y+y, origin.X+x0+btn/2, origin.Y+y+btn)
}

// layoutTip draws the tooltip of the hovered toolbar button once the cursor has rested on it
// for a moment. It hangs on the far side of the toolbar from the selection, so it never
// covers the buttons it describes.
func (m *monitor) layoutTip(gtx layout.Context, win image.Point) {
	if m.tipCand == "" || m.menuOpen { // an open menu is drawn over the toolbar: no tooltip under it
		m.tipText = ""
		return
	}
	if m.tipCand != m.tipText {
		m.tipText, m.tipRect, m.tipSince = m.tipCand, m.tipCandRect, time.Now()
	}
	if wait := tipDelay - time.Since(m.tipSince); wait > 0 {
		gtx.Execute(op.InvalidateCmd{At: time.Now().Add(wait)})
		return
	}
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: iconColor}.Add(gtx.Ops)
	mat := rec.Stop()
	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(1<<12, 1<<12)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, 14, m.tipText, mat)
	label := rec.Stop()

	padX, padY, margin := gtx.Dp(10), gtx.Dp(6), gtx.Dp(6)
	w, h := dims.Size.X+2*padX, dims.Size.Y+2*padY
	x := clampInt(m.tipRect.Min.X+m.tipRect.Dx()/2-w/2, gtx.Dp(8), max(win.X-w-gtx.Dp(8), gtx.Dp(8)))
	y := m.barRect.Max.Y + margin
	if !m.barBelow {
		y = m.barRect.Min.Y - margin - h
	}
	y = clampInt(y, gtx.Dp(8), max(win.Y-h-gtx.Dp(8), gtx.Dp(8)))
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	pill := clip.UniformRRect(image.Rect(0, 0, w, h), min(h/2, gtx.Dp(8)))
	paint.FillShape(gtx.Ops, barBG, pill.Op(gtx.Ops))
	defer op.Offset(image.Pt(padX, padY)).Push(gtx.Ops).Pop()
	label.Add(gtx.Ops)
}
