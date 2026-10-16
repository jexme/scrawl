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

// TestDock проверяет содержимое и положение док-панели в разных состояниях.
func TestDock(t *testing.T) {
	sandbox(t)
	m, _ := newTestMonitor(t)
	m.last, m.hasLast = image.Rect(10, 10, 100, 100), true

	// Без подборки и выделения: только возврат прошлой области.
	if got := m.collectItems(); len(got) != 0 {
		t.Errorf("без выделения и подборки действий быть не должно: %d", len(got))
	}

	// Выделение без подборки: кнопка «В подборку», склейки пока нет.
	m.sel, m.hasSel = image.Rect(20, 20, 80, 80), true
	items := m.collectItems()
	if len(items) != 1 || !items[0].primary {
		t.Errorf("с выделением должна быть одна основная кнопка «В подборку»: %+v", items)
	}

	// Подборка из одной картинки и выделение: добавить ещё, склеить, отдельно, очистить.
	m.queued = 1
	items = m.collectItems()
	want := []string{"Добавить ещё  ·  A", "Склеить с этой областью  ·  G", "Отдельно  ·  D", "Очистить"}
	if len(items) != len(want) {
		t.Fatalf("ожидали %d действий, получили %d", len(want), len(items))
	}
	for i, w := range want {
		if items[i].txt != w {
			t.Errorf("действие %d: %q, ожидали %q", i, items[i].txt, w)
		}
	}

	// Подборка без выделения и без прошлой области: склеить одну нельзя.
	m.hasSel = false
	items = m.collectItems()
	for _, it := range items {
		if it.btn == &m.dock.glue {
			t.Error("склеивать одну картинку нельзя")
		}
	}
}

// TestDockPlacement проверяет, что док уходит наверх, когда внизу выделение.
func TestDockPlacement(t *testing.T) {
	sandbox(t)
	const W, H = 1000, 600
	m, _ := newTestMonitor(t)
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	win := image.Pt(W, H)

	// Выделение сверху: док внизу.
	m.hasSel, m.selWin = true, image.Rect(100, 50, 500, 200)
	if p := m.placeDock(gtx, win, 400, 40); p.Y < H/2 {
		t.Errorf("док должен быть внизу, получили y=%d", p.Y)
	}
	// Выделение внизу: док наверху.
	m.selWin = image.Rect(100, 400, 500, 580)
	if p := m.placeDock(gtx, win, 400, 40); p.Y > H/2 {
		t.Errorf("док должен уйти наверх, получили y=%d", p.Y)
	}
	// Выделения нет: всегда внизу.
	m.hasSel = false
	if p := m.placeDock(gtx, win, 400, 40); p.Y < H/2 {
		t.Errorf("без выделения док внизу, получили y=%d", p.Y)
	}
}

// TestDockRender рисует док для просмотра глазами (с SCRAWL_TEST_OUT).
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
