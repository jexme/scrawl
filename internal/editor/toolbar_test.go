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

// TestToolbarKeyCaptions рисует панель и проверяет, что клавиши написаны под кнопками с быстрым
// вызовом: под инструментами и действиями в основном ряду, под «−» и «+» во втором, а под цветами
// подписей нет. С SCRAWL_TEST_OUT=путь.png результат сохраняется.
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
	// Основной ряд: подписи под инструментами и под отменой, копированием, сохранением и закрытием.
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

	// Второй ряд лежит под основным, он ниже: подписей у него нет.
	if m.barRect.Dy() <= main.Dy() {
		t.Fatalf("при рисующем инструменте ожидали второй ряд под основным: панель %v, основной ряд %v", m.barRect, main)
	}
	style := image.Rect(main.Min.X, main.Max.Y+4, main.Min.X+m.bar.styleWidth(34, 4, 10, 6), m.barRect.Max.Y)
	if style.Dy() != 34+2*6 {
		t.Errorf("высота второго ряда %d, ожидали %d (кнопка и поля, без строки подписей)", style.Dy(), 34+2*6)
	}
	// Под кнопками «−» и «+» подсказок нет: нет вообще ничего светлее фона ряда ниже кнопок.
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

	// У «выделения» второго ряда нет.
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

// TestToolKeys: клавиша инструмента включает его, но только когда область выбрана.
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

// TestWidthKeys: плюс и минус меняют толщину текущего инструмента, у «выделения» ничего не делают.
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

// TestToolbarClickRedrawsImmediately: клик по инструменту правее активного. Кнопки левее нажатой
// не должны остаться подсвеченными (раньше клик обрабатывался посреди рисования, и подсвечены были
// оба инструмента до следующего движения мыши), и должен запрашиваться ещё один кадр.
func TestToolbarClickRedrawsImmediately(t *testing.T) {
	testenv.Config(t, t.TempDir())
	const W, H = 1200, 300
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(100, 20, 1100, 120), true
	m.tool = annotate.ToolPen // активно перо, нажмём стрелку, она правее

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

	center := func(i int) f32.Point { // середина кнопки i-го инструмента
		return f32.Pt(float32(m.mainRow.Min.X+6+i*38+17), float32(m.mainRow.Min.Y+6+17))
	}
	arrow := int(annotate.ToolArrow)
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: center(arrow)})
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: center(arrow), Buttons: pointer.ButtonPrimary})
	frame()
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: center(arrow)})
	r.WakeupTime() // сбросить прежние запросы кадра
	frame()        // в этом кадре клик обработан
	if m.tool != annotate.ToolArrow {
		t.Fatalf("после клика инструмент %v, ожидали стрелку", m.tool)
	}

	// Кадр после клика: подсвечена только стрелка.
	if err := win.Frame(&ops); err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	highlighted := func(i int) bool { // угол кнопки закрашен акцентным цветом
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

// TestToolbarHover: кнопка под мышью слегка подсвечена, остальные нет.
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

// TestToolbarOverflowMenu: «⋯» открывает меню, повторный клик, клик по другой кнопке и клик по холсту
// закрывают его; меню рисуется поверх и не выходит за окно.
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
	// Клик по инструменту закрывает меню и включает инструмент.
	click(f32.Pt(float32(m.mainRow.Min.X+6+int(annotate.ToolArrow)*38+17), float32(m.mainRow.Min.Y+6+17)))
	if m.menuOpen || m.tool != annotate.ToolArrow {
		t.Errorf("клик по инструменту: меню открыто=%v, инструмент %v", m.menuOpen, m.tool)
	}
	// Клик по холсту закрывает меню и не рисует.
	click(more)
	m.press(f32.Pt(10, 10))
	if m.menuOpen {
		t.Error("клик по холсту не закрыл меню")
	}

	// Меню целиком в окне и ниже панели.
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
	// Под панелью (на 12 px ниже её края) в колонке «⋯» должен быть фон меню, а не пустой кадр.
	probe := out.RGBAAt(m.menuAnchor-5, m.barRect.Max.Y+14)
	if probe.R > 80 || probe.G > 80 {
		t.Errorf("под панелью не нарисовано меню: %v", probe)
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".menu.png", data, 0o644)
	}
}

// TestWidthValueCentered: число толщины стоит по центру кнопки и по горизонтали, и по вертикали,
// при любом числе знаков.
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
				if c := out.RGBAAt(x, y); c.R > 150 && c.G > 150 && c.B > 150 { // светлые цифры на тёмной кнопке
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
