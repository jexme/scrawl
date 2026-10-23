package editor

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/testenv"
)

// A toolbar button gets a tooltip with its name and key once the cursor has rested on it, and
// the tooltip goes away when the cursor leaves.
func TestToolbarTooltip(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 800, 600
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, W, H))}, func() {})
	m.sel, m.hasSel = image.Rect(50, 50, 500, 300), true
	size := image.Pt(W, H)
	var r input.Router
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size), Source: r.Source()}
		m.layout(gtx, size)
		r.Frame(&ops)
	}
	frame()
	frame()
	if m.tipText != "" {
		t.Fatalf("no tooltip before the cursor is on a button: %q", m.tipText)
	}
	// The second button of the main row is the pen: the row starts at mainRow.Min, the padding
	// is 6 and a button 34 wide with a gap of 4.
	over := f32.Pt(float32(m.mainRow.Min.X+6+34+4+17), float32(m.mainRow.Min.Y+6+17))
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: over})
	frame()
	frame()
	if m.tipCand == "" || m.tipText == "" {
		t.Fatalf("the hovered button must offer a tooltip: cand=%q shown=%q", m.tipCand, m.tipText)
	}
	if want := tipText("Pen", "P"); m.tipText != want {
		t.Errorf("tooltip %q, expected %q", m.tipText, want)
	}
	// After the delay the tooltip is drawn (this must not panic and must ask for no more frames).
	m.tipSince = time.Now().Add(-time.Second)
	frame()
	// The cursor moves away from the toolbar.
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(300, 100)})
	frame()
	frame()
	if m.tipText != "" {
		t.Errorf("the tooltip must go away with the cursor: %q", m.tipText)
	}
}

// With the whole screen selected the toolbar is inside the selection, and the "To collection"
// panel must not lie over its buttons.
func TestDockDoesNotCoverToolbarOnFullScreenSelection(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 800, 600
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, W, H))}, func() {})
	m.selectAll()
	size := image.Pt(W, H)
	var r input.Router
	var ops op.Ops
	gtx := func() layout.Context {
		ops.Reset()
		return layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size), Source: r.Source()}
	}
	g := gtx()
	m.layout(g, size)
	r.Frame(&ops)
	g = gtx()
	m.layout(g, size)
	for _, w := range []int{260, 300} {
		p := m.placeDock(g, size, w, 40)
		if got := (image.Rectangle{Min: p, Max: p.Add(image.Pt(w, 40))}); got.Overlaps(m.barSpace) {
			t.Errorf("the dock %v lies over the toolbar %v", got, m.barSpace)
		}
	}
}
