package editor

import (
	"image"
	"runtime"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/testenv"
)

// The selection must survive releasing the left button: events go through a real
// input.Router, as in the running program.
func TestSelectionSurvivesRelease(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 800, 600
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, W, H))}, func() {})
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

	send := func(e pointer.Event) {
		e.Source = pointer.Mouse
		r.Queue(e)
		frame()
	}
	send(pointer.Event{Kind: pointer.Move, Position: f32.Pt(100, 100)})
	send(pointer.Event{Kind: pointer.Press, Buttons: pointer.ButtonPrimary, Position: f32.Pt(100, 100)})
	send(pointer.Event{Kind: pointer.Move, Buttons: pointer.ButtonPrimary, Position: f32.Pt(200, 150)})
	send(pointer.Event{Kind: pointer.Move, Buttons: pointer.ButtonPrimary, Position: f32.Pt(300, 250)})
	if !m.dragging || m.sel.Dx() < 150 {
		t.Fatalf("во время перетаскивания: dragging=%v sel=%v", m.dragging, m.sel)
	}
	send(pointer.Event{Kind: pointer.Release, Position: f32.Pt(300, 250)})
	if !m.hasSel || m.dragging {
		t.Fatalf("после отпускания кнопки: hasSel=%v dragging=%v sel=%v", m.hasSel, m.dragging, m.sel)
	}
	if want := image.Rect(100, 100, 300, 250); m.sel != want {
		t.Errorf("область %v, ждали %v", m.sel, want)
	}
	frame()
	if !m.hasSel {
		t.Errorf("выделение пропало на следующем кадре: sel=%v", m.sel)
	}
}

// Shift while drawing with the pen makes a straight line from the start of the stroke;
// without Shift the stroke stays freehand.
func TestShiftStraightPenLine(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 800, 600
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, W, H))}, func() {})
	m.sel, m.hasSel = image.Rect(50, 50, 700, 500), true
	m.setTool(annotate.ToolPen)
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
	send := func(kind pointer.Kind, x, y float32, mods key.Modifiers) {
		r.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(x, y), Modifiers: mods})
		frame()
	}

	// A wavy stroke with Shift held at the end: a straight horizontal line.
	send(pointer.Press, 100, 200, 0)
	send(pointer.Move, 150, 230, 0)
	send(pointer.Move, 200, 190, 0)
	if n := len(m.cur.Pts); n != 3 {
		t.Fatalf("freehand: %d points, expected 3", n)
	}
	send(pointer.Move, 400, 207, key.ModShift)
	if n := len(m.cur.Pts); n != 2 || m.cur.Pts[0] != f32.Pt(100, 200) || m.cur.Pts[1].Y != 200 || m.cur.Pts[1].X < 299 {
		t.Fatalf("with Shift the line must be straight and horizontal: %v", m.cur.Pts)
	}
	// Shift released: the freehand path comes back, no point was lost.
	send(pointer.Move, 420, 215, 0)
	if n := len(m.cur.Pts); n != 5 {
		t.Fatalf("after releasing Shift: %d points, expected 5", n)
	}
	send(pointer.Move, 500, 260, key.ModShift)
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(500, 260), Modifiers: key.ModShift})
	frame()
	if len(m.shapes) != 1 || len(m.shapes[0].Pts) != 2 {
		t.Fatalf("the finished stroke must be one straight segment: %+v", m.shapes)
	}
}

// With Shift a box becomes a square, a line snaps to 45 degrees.
func TestShiftSquareAndLine(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 800, 600
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, W, H))}, func() {})
	m.sel, m.hasSel = image.Rect(50, 50, 700, 500), true
	for _, c := range []struct {
		tool annotate.Tool
		end  f32.Point
		want f32.Point
	}{
		{annotate.ToolRect, f32.Pt(300, 180), f32.Pt(250, 50)},  // 200 wide, but only 150 px of room above
		{annotate.ToolBlur, f32.Pt(150, 400), f32.Pt(300, 400)}, // 200 tall, so 200 wide
	} {
		m.setTool(c.tool)
		m.shift = true
		m.press(f32.Pt(100, 200))
		m.drag(c.end)
		if got := m.cur.Pts[1]; got != c.want {
			t.Errorf("tool %v: end point %v, expected %v", c.tool, got, c.want)
		}
		m.release(c.end)
	}
	m.setTool(annotate.ToolArrow)
	m.shift = true
	m.press(f32.Pt(100, 200))
	m.drag(f32.Pt(300, 215))
	if got := m.cur.Pts[1]; got.Y != 200 || got.X < 299 {
		t.Errorf("arrow with Shift: %v, expected a horizontal line", got)
	}
}

// Starting a new selection drops the annotations of the old one (they used to stay hidden in
// the frame and ended up in the next shot); a click without dragging keeps the old selection
// and its annotations instead of leaving the user with nothing.
func TestNewSelectionDropsAnnotationsButClickKeepsThem(t *testing.T) {
	testenv.Config(t, t.TempDir())
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, 800, 600))}, func() {})
	m.sel, m.hasSel = image.Rect(100, 100, 600, 500), true
	m.shapes = []annotate.Shape{{Tool: annotate.ToolRect, Pts: []f32.Point{f32.Pt(200, 200), f32.Pt(400, 300)}}}
	m.setTool(annotate.ToolSelect)

	// A stray click outside the selection: nothing is lost.
	m.press(f32.Pt(700, 550))
	m.release(f32.Pt(700, 550))
	if !m.hasSel || m.sel != image.Rect(100, 100, 600, 500) || len(m.shapes) != 1 {
		t.Fatalf("a click must keep the selection and the annotations: hasSel=%v sel=%v shapes=%d", m.hasSel, m.sel, len(m.shapes))
	}

	// A real new selection starts clean.
	m.press(f32.Pt(650, 20))
	m.drag(f32.Pt(780, 90))
	m.release(f32.Pt(780, 90))
	if !m.hasSel || m.sel != image.Rect(650, 20, 780, 90) {
		t.Fatalf("new selection: hasSel=%v sel=%v", m.hasSel, m.sel)
	}
	if len(m.shapes) != 0 {
		t.Errorf("annotations of the old area must be dropped, %d left", len(m.shapes))
	}
}

// A drag to the last pixel of the screen reaches the edge of the frame.
func TestSelectionReachesFrameEdge(t *testing.T) {
	if got := rectOf(f32.Pt(0, 0), f32.Pt(1279.4, 799.2)); got != image.Rect(0, 0, 1280, 800) {
		t.Errorf("rectOf rounded the far corner down: %v", got)
	}
	if got := rectOf(f32.Pt(10.7, 20.2), f32.Pt(30, 40)); got != image.Rect(10, 20, 30, 40) {
		t.Errorf("the near corner is rounded down, whole numbers stay: %v", got)
	}
}

// Shift pressed or released without moving the mouse is taken into account on release.
func TestShiftOnReleaseWithoutMoving(t *testing.T) {
	testenv.Config(t, t.TempDir())
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, 800, 600))}, func() {})
	m.sel, m.hasSel = image.Rect(50, 50, 700, 500), true
	m.setTool(annotate.ToolLine)
	m.press(f32.Pt(100, 200))
	m.drag(f32.Pt(300, 230)) // Shift was not held while moving
	m.shift = true           // Shift pressed, the mouse does not move, the button is released
	m.release(f32.Pt(300, 230))
	if len(m.shapes) != 1 || m.shapes[0].Pts[1].Y != 200 {
		t.Fatalf("the line must snap on release: %+v", m.shapes)
	}
}

// A failed copy to the clipboard is shown on the screen and the editor stays open: with no
// notify-send either, the user would otherwise see only an overlay that closes with nothing
// copied.
func TestClipboardFailureStaysVisible(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows copies through its own clipboard API; there is no external tool that can be missing")
	}
	testenv.Config(t, t.TempDir())
	empty := t.TempDir()
	t.Setenv("PATH", empty) // neither wl-copy, xclip nor notify-send
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")
	m, finished := newTestMonitor(t)
	m.sel, m.hasSel = image.Rect(10, 10, 200, 150), true
	m.export(true, false)
	if *finished {
		t.Error("the editor must stay open when the clipboard failed")
	}
	if m.toastMsg == "" || time.Until(m.toastUntil) < 3*time.Second {
		t.Errorf("the error must be shown on the screen for a while: %q for %v", m.toastMsg, time.Until(m.toastUntil))
	}
}

// Esc while typing a text drops the text only: the overlay and the other annotations stay.
func TestEscDuringTextKeepsOverlay(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 800, 600
	finished := false
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: image.NewRGBA(image.Rect(0, 0, W, H))}, func() { finished = true })
	m.sel, m.hasSel = image.Rect(50, 50, 700, 500), true
	m.shapes = []annotate.Shape{{Tool: annotate.ToolRect, Pts: []f32.Point{f32.Pt(100, 100), f32.Pt(200, 200)}}}
	m.setTool(annotate.ToolText)
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
	m.press(f32.Pt(300, 300)) // starts typing
	if !m.editing {
		t.Fatal("the text tool must start editing")
	}
	frame()
	frame()
	r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	frame()
	frame()
	if m.editing {
		t.Error("Esc must cancel the text being typed")
	}
	if finished || len(m.shapes) != 1 {
		t.Errorf("Esc while typing must not close the overlay or drop annotations: finished=%v shapes=%d", finished, len(m.shapes))
	}
}
