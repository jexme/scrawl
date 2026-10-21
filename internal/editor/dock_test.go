package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// TestDock checks the contents and position of the dock panel in different states.
func TestDock(t *testing.T) {
	sandbox(t)
	m, _ := newTestMonitor(t)
	m.last, m.hasLast = image.Rect(10, 10, 100, 100), true

	// No collection and no selection: only restoring the previous area.
	if got := m.collectItems(); len(got) != 0 {
		t.Errorf("без выделения и подборки действий быть не должно: %d", len(got))
	}

	// A selection without a collection: the "Add to collection" button, no glue yet.
	m.sel, m.hasSel = image.Rect(20, 20, 80, 80), true
	items := m.collectItems()
	if len(items) != 1 || !items[0].primary {
		t.Errorf("с выделением должна быть одна основная кнопка «В подборку»: %+v", items)
	}

	// A collection of one image and a selection: add another, glue, send separately, clear.
	m.queued = 1
	items = m.collectItems()
	want := []string{"Add another  ·  A", "Glue with this area  ·  G", "As files  ·  D", "Clear"}
	if len(items) != len(want) {
		t.Fatalf("ожидали %d действий, получили %d", len(want), len(items))
	}
	for i, w := range want {
		if items[i].txt != w {
			t.Errorf("действие %d: %q, ожидали %q", i, items[i].txt, w)
		}
	}

	// A collection with no selection and no previous area: a single image cannot be glued.
	m.hasSel = false
	items = m.collectItems()
	for _, it := range items {
		if it.btn == &m.dock.glue {
			t.Error("склеивать одну картинку нельзя")
		}
	}
}

// TestDockPlacement checks that the dock moves to the top when the selection is at the bottom.
func TestDockPlacement(t *testing.T) {
	sandbox(t)
	const W, H = 1000, 600
	m, _ := newTestMonitor(t)
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	win := image.Pt(W, H)

	// Selection at the top: dock at the bottom.
	m.hasSel, m.selWin = true, image.Rect(100, 50, 500, 200)
	if p := m.placeDock(gtx, win, 400, 40); p.Y < H/2 {
		t.Errorf("док должен быть внизу, получили y=%d", p.Y)
	}
	// Selection at the bottom: dock at the top.
	m.selWin = image.Rect(100, 400, 500, 580)
	if p := m.placeDock(gtx, win, 400, 40); p.Y > H/2 {
		t.Errorf("док должен уйти наверх, получили y=%d", p.Y)
	}
	// No selection: always at the bottom.
	m.hasSel = false
	if p := m.placeDock(gtx, win, 400, 40); p.Y < H/2 {
		t.Errorf("без выделения док внизу, получили y=%d", p.Y)
	}
}

// TestDockRender draws the dock for viewing by eye (with SCRAWL_TEST_OUT).
func TestDockRender(t *testing.T) {
	sandbox(t)
	const W, H = 900, 300
	m, _ := newTestMonitor(t)
	m.queued, m.last, m.hasLast = 2, image.Rect(10, 10, 100, 100), true
	m.sel, m.hasSel = image.Rect(150, 40, 750, 160), true
	m.selWin = m.sel

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()

	var ops op.Ops
	paint.Fill(&ops, color.NRGBA{R: 70, G: 76, B: 90, A: 255})
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	m.layoutDock(gtx, image.Pt(W, H))
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".dock.png", data, 0o644)
	}
}
