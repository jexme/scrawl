package editor

import (
	"github.com/jexme/scrawl/internal/testenv"
	"image"
	"os"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
)

// TestToolbarKeyCaptions draws the toolbar and checks that the keys are written under the
// buttons with a quick call: under the tools and actions in the main row, under "−" and
// "+" in the second one, and there are no captions under the colours. With
// SCRAWL_TEST_OUT=path.png the result is saved.
func TestToolbarKeyCaptions(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 1100, 300
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = 60, 70, 90, 255
	}
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(100, 20, 500, 120), true
	m.tool = annotate.ToolPen

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(image.Pt(W, H))}
	m.layoutToolbar(gtx, image.Pt(W, H), m.sel)
	if err := win.Frame(&ops); err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}

	main := m.mainRow
	if main.Dy() < 34+13 {
		t.Fatalf("ряд %v слишком низкий для подписей", main)
	}
	light := func(r image.Rectangle) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if c := out.RGBAAt(x, y); c.R > 150 && c.G > 150 && c.B > 150 {
					n++
				}
			}
		}
		return n
	}
	// Main row: captions under the tools and under undo, copy, save and close.
	caps := func(row image.Rectangle) image.Rectangle {
		return image.Rect(row.Min.X, row.Max.Y-13, row.Max.X, row.Max.Y)
	}
	toolsEnd := main.Min.X + 6 + len(m.bar.tools)*38
	if n := light(image.Rect(main.Min.X, caps(main).Min.Y, toolsEnd, main.Max.Y)); n < 60 {
		t.Errorf("под инструментами нет подписей клавиш: найдено %d светлых пикселей", n)
	}
	if n := light(image.Rect(toolsEnd, caps(main).Min.Y, main.Max.X, main.Max.Y)); n < 40 {
		t.Errorf("под кнопками действий нет подписей клавиш: найдено %d светлых пикселей", n)
	}

	// The second row lies below the main one, it is lower: it has no captions.
	if m.barRect.Dy() <= main.Dy() {
		t.Fatalf("при рисующем инструменте ожидали второй ряд под основным: панель %v, основной ряд %v", m.barRect, main)
	}
	style := image.Rect(main.Min.X, main.Max.Y+4, main.Min.X+m.bar.styleWidth(34, 4, 10, 6), m.barRect.Max.Y)
	if style.Dy() != 34+2*6 {
		t.Errorf("высота второго ряда %d, ожидали %d (кнопка и поля, без строки подписей)", style.Dy(), 34+2*6)
	}
	// There are no hints under the "−" and "+" buttons: below the buttons there is nothing at
	// all lighter than the row background.
	if n := light(image.Rect(style.Min.X, style.Max.Y, style.Max.X, style.Max.Y+13)); n != 0 {
		t.Errorf("под вторым рядом что-то нарисовано: %d светлых пикселей", n)
	}

	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, err := encodePNG(out)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// "Select" has no second row.
	m.tool = annotate.ToolSelect
	ops.Reset()
	m.layoutToolbar(gtx, image.Pt(W, H), m.sel)
	if !m.barRect.Eq(m.mainRow) {
		t.Errorf("у выделения панель должна быть одним рядом: %v и %v", m.barRect, m.mainRow)
	}
	if m.barSpace.Dy() <= m.mainRow.Dy() {
		t.Errorf("место под второй ряд должно быть занято и у выделения: %v", m.barSpace)
	}
}

// TestToolKeys: a tool key enables the tool, but only when a region is selected.
func TestToolKeys(t *testing.T) {
	testenv.Config(t, t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	run := func(action string) {
		t.Helper()
		for _, kb := range m.keys {
			if kb.action == action {
				kb.run()
				return
			}
		}
		t.Fatalf("действия %q нет среди клавиш", action)
	}

	m.tool = annotate.ToolSelect
	run(settings.ToolAction("pen"))
	if m.tool != annotate.ToolSelect {
		t.Errorf("без области клавиша инструмента не должна его включать: %v", m.tool)
	}
	m.sel, m.hasSel = image.Rect(10, 10, 50, 50), true
	for t2, name := range settings.ToolNames {
		run(settings.ToolAction(name))
		if m.tool != annotate.Tool(t2) {
			t.Errorf("клавиша %q: инструмент %v, ожидали %v", name, m.tool, annotate.Tool(t2))
		}
	}
}

// TestWidthKeys: plus and minus change the thickness of the current tool, and do nothing
// for "Select".
func TestWidthKeys(t *testing.T) {
	testenv.Config(t, t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(10, 10, 50, 50), true
	run := func(action string) {
		t.Helper()
		for _, kb := range m.keys {
			if kb.action == action {
				kb.run()
				return
			}
		}
		t.Fatalf("действия %q нет среди клавиш", action)
	}

	m.tool = annotate.ToolSelect
	run(settings.ActionWidthUp)
	if _, ok := m.widths[annotate.ToolSelect]; ok {
		t.Error("у «выделения» толщины нет")
	}
	m.tool = annotate.ToolPen
	before := m.curWidth()
	run(settings.ActionWidthUp)
	if m.curWidth() != before+1 {
		t.Errorf("плюс: %v -> %v", before, m.curWidth())
	}
	run(settings.ActionWidthDown)
	run(settings.ActionWidthDown)
	if m.curWidth() != before-1 {
		t.Errorf("два минуса: %v -> %v", before, m.curWidth())
	}
}

// TestToolbarClickRedrawsImmediately: a click on a tool to the right of the active one.
// The buttons to the left of the pressed one must not stay highlighted (previously the
// click was handled in the middle of drawing, and both tools were highlighted until the
// next mouse move), and one more frame must be requested.
func TestToolbarClickRedrawsImmediately(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 1200, 300
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(100, 20, 1100, 120), true
	m.tool = annotate.ToolPen // the pen is active, we press the arrow, which is to the right of it

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()
	size := image.Pt(W, H)
	var r input.Router
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(size), Source: r.Source()}
		m.layoutToolbar(gtx, size, m.sel)
		r.Frame(&ops)
	}
	frame()
	frame()

	center := func(i int) f32.Point { // centre of the button of the i-th tool
		return f32.Pt(float32(m.mainRow.Min.X+6+i*38+17), float32(m.mainRow.Min.Y+6+17))
	}
	arrow := int(annotate.ToolArrow)
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: center(arrow)})
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: center(arrow), Buttons: pointer.ButtonPrimary})
	frame()
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: center(arrow)})
	r.WakeupTime() // discard earlier frame requests
	frame()        // the click is handled in this frame
	if m.tool != annotate.ToolArrow {
		t.Fatalf("после клика инструмент %v, ожидали стрелку", m.tool)
	}

	// The frame after the click: only the arrow is highlighted.
	if err := win.Frame(&ops); err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	highlighted := func(i int) bool { // the corner of the button is filled with the accent colour
		c := out.RGBAAt(m.mainRow.Min.X+6+i*38+4, m.mainRow.Min.Y+6+17)
		return c.B > 200 && c.R < 100
	}
	if !highlighted(arrow) {
		t.Error("нажатая кнопка не подсвечена")
	}
	if highlighted(int(annotate.ToolPen)) {
		t.Error("прежний инструмент остался подсвеченным в том же кадре")
	}
	if _, ok := r.WakeupTime(); !ok {
		t.Error("после клика не запрошен новый кадр")
	}
}

// TestToolbarHover: the button under the mouse is slightly highlighted, the others are
// not.
func TestToolbarHover(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 1200, 300
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(100, 20, 1100, 120), true
	m.tool = annotate.ToolPen

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()
	size := image.Pt(W, H)
	var r input.Router
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(size), Source: r.Source()}
		m.layoutToolbar(gtx, size, m.sel)
		r.Frame(&ops)
	}
	frame()
	frame()
	line := int(annotate.ToolLine)
	at := func(i int) (int, int) { return m.mainRow.Min.X + 6 + i*38 + 4, m.mainRow.Min.Y + 6 + 17 }
	x, y := at(line)
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(x+13), float32(y))})
	frame()
	if err := win.Frame(&ops); err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	bg := out.RGBAAt(m.mainRow.Min.X+2, m.mainRow.Min.Y+2)
	hov := out.RGBAAt(x, y)
	other := out.RGBAAt(at(int(annotate.ToolRect)))
	if hov.R <= bg.R+15 || hov.B > 150 {
		t.Errorf("кнопка под мышью не подсвечена: %v на фоне %v", hov, bg)
	}
	if other.R > bg.R+5 {
		t.Errorf("кнопка без мыши подсвечена: %v на фоне %v", other, bg)
	}
}

// TestToolbarOverflowMenu: "⋯" opens the menu; a second click, a click on another button
// and a click on the canvas close it; the menu is drawn on top and does not go outside the
// window.
func TestToolbarOverflowMenu(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 1100, 400
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(100, 20, 1000, 120), true
	m.tool = annotate.ToolPen

	size := image.Pt(W, H)
	var r input.Router
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(size), Source: r.Source()}
		m.layoutToolbar(gtx, size, m.sel)
		r.Frame(&ops)
	}
	click := func(p f32.Point) {
		t.Helper()
		r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p})
		r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary})
		frame()
		r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
		frame()
		frame()
	}
	frame()
	frame()

	more := f32.Pt(float32(m.menuAnchor), float32(m.mainRow.Min.Y+6+17))
	click(more)
	if !m.menuOpen {
		t.Fatal("клик по «⋯» не открыл меню")
	}
	click(more)
	if m.menuOpen {
		t.Error("повторный клик по «⋯» не закрыл меню")
	}
	click(more)
	// A click on a tool closes the menu and enables the tool.
	click(f32.Pt(float32(m.mainRow.Min.X+6+int(annotate.ToolArrow)*38+17), float32(m.mainRow.Min.Y+6+17)))
	if m.menuOpen || m.tool != annotate.ToolArrow {
		t.Errorf("клик по инструменту: меню открыто=%v, инструмент %v", m.menuOpen, m.tool)
	}
	// A click on the canvas closes the menu and draws nothing.
	click(more)
	m.press(f32.Pt(10, 10))
	if m.menuOpen {
		t.Error("клик по холсту не закрыл меню")
	}

	// The menu is entirely inside the window and below the toolbar.
	m.menuOpen = true
	var hw *headless.Window
	hw, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer hw.Release()
	ops.Reset()
	gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(size), Source: r.Source()}
	m.layoutToolbar(gtx, size, m.sel)
	if err := hw.Frame(&ops); err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := hw.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	// Below the toolbar (12 px below its edge) in the "⋯" column there must be the menu
	// background, not an empty frame.
	probe := out.RGBAAt(m.menuAnchor-5, m.barRect.Max.Y+14)
	if probe.R > 80 || probe.G > 80 {
		t.Errorf("под панелью не нарисовано меню: %v", probe)
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".menu.png", data, 0o644)
	}
}

// TestWidthValueCentered: the thickness number sits in the centre of the button both
// horizontally and vertically, for any number of digits.
func TestWidthValueCentered(t *testing.T) {
	testenv.Config(t, t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.tool = annotate.ToolPen
	hw, err := headless.NewWindow(120, 120)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer hw.Release()
	for _, w := range []float32{1, 4, 8, 10, 22, 30} {
		m.widths[annotate.ToolPen] = w
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(image.Pt(34, 34))}
		paint.FillShape(&ops, barBG, clip.Rect(image.Rect(40, 40, 74, 74)).Op())
		st := op.Offset(image.Pt(40, 40)).Push(&ops)
		m.drawWidthValue(gtx, true)
		st.Pop()
		if err := hw.Frame(&ops); err != nil {
			t.Skipf("headless рендер недоступен: %v", err)
		}
		out := image.NewRGBA(image.Rect(0, 0, 120, 120))
		if err := hw.Screenshot(out); err != nil {
			t.Fatal(err)
		}
		minX, minY, maxX, maxY := 1<<20, 1<<20, -1, -1
		for y := 0; y < 120; y++ {
			for x := 0; x < 120; x++ {
				if c := out.RGBAAt(x, y); c.R > 150 && c.G > 150 && c.B > 150 { // light digits on a dark button
					minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
				}
			}
		}
		if maxX < 0 {
			t.Errorf("толщина %v: числа не видно", w)
			continue
		}
		cx, cy := float32(minX+maxX+1)/2, float32(minY+maxY+1)/2
		if dx, dy := cx-57, cy-57; dx < -1.5 || dx > 1.5 || dy < -1.5 || dy > 1.5 {
			t.Errorf("толщина %v: середина числа (%.1f, %.1f), центр кнопки (57, 57)", w, cx, cy)
		}
	}
}
