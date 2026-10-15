// Package editor показывает замороженный кадр на весь монитор, даёт выделить
// область, нарисовать аннотации и сохранить результат.
package editor

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/jexme/screen-editor/internal/annotate"
	"github.com/jexme/screen-editor/internal/screencast"
)

// Config — параметры запуска редактора.
type Config struct {
	Frames  []screencast.Frame
	Finish  func()    // освобождает ресурсы и завершает процесс
	T0      time.Time // момент «нажатия хоткея» для замеров
	Verbose bool
	Fresh   bool // не предлагать область предыдущего скриншота
}

var (
	shade     = color.NRGBA{A: 185}
	selBorder = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	handleFg  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	handleBg  = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
	textFrame = color.NRGBA{R: 255, G: 255, B: 255, A: 160}
	identity  = unit.Metric{PxPerDp: 1, PxPerSp: 1}
)

// Режимы правки готовой области.
const (
	adjustNone = iota
	adjustResize
	adjustMove
)

// Run открывает по fullscreen-окну на каждый кадр. Вызывающий должен после
// этого вызвать app.Main().
func Run(cfg Config) {
	app.ID = screencast.AppID
	var once sync.Once
	finish := func() { once.Do(cfg.Finish) }
	monitors := make([]*monitor, len(cfg.Frames))
	for i := range cfg.Frames {
		monitors[i] = newMonitor(cfg, &cfg.Frames[i], finish)
	}
	for _, m := range monitors {
		m.peers = monitors
		go m.loop()
	}
}

// monitor — окно-оверлей одного монитора со своим состоянием редактирования.
type monitor struct {
	cfg    Config
	frame  *screencast.Frame
	finish func()
	shaper *text.Shaper
	env    annotate.Env // шрифт и размытие для отрисовки аннотаций
	bar    *toolbar

	// peers — все окна-оверлеи (включая это). Активна одна область на все мониторы:
	// когда выделение начинается в одном окне, остальные сбрасываются.
	peers []*monitor
	win   atomic.Pointer[app.Window]
	reset atomic.Bool // другое окно забрало выделение, нужно сбросить своё

	imageOp paint.ImageOp
	canvas  int // тег холста

	// Выделение в пикселях кадра.
	sel      image.Rectangle
	hasSel   bool
	dragging bool
	selStart f32.Point

	// Изменение размера и перемещение готовой области.
	adjust      int
	adjEdge     edge
	adjStartSel image.Rectangle
	adjStartPt  f32.Point
	hover       f32.Point // последнее положение курсора, пиксели кадра

	// Плашка с новым значением силы инструмента.
	toastUntil time.Time
	toastTool  annotate.Tool
	toastValue int

	// Область предыдущего скриншота: не подставляется сама, возвращается по кнопке или клавише R.
	last    image.Rectangle
	hasLast bool
	lastBtn widget.Clickable
	tol     float32 // допуск попадания в ручку, пиксели кадра

	tool   annotate.Tool
	color  color.NRGBA
	widths map[annotate.Tool]float32 // толщина или сила для каждого инструмента отдельно
	shapes []annotate.Shape
	cur    *annotate.Shape

	editing bool
	editAt  f32.Point
	ed      widget.Editor

	focusCanvas bool
	focusEditor bool
	reported    bool
}

func newMonitor(cfg Config, f *screencast.Frame, finish func()) *monitor {
	m := &monitor{
		cfg:         cfg,
		frame:       f,
		finish:      finish,
		shaper:      annotate.NewShaper(),
		bar:         newToolbar(),
		imageOp:     paint.NewImageOp(f.Img),
		color:       annotate.Palette[0],
		widths:      make(map[annotate.Tool]float32),
		focusCanvas: true,
	}
	m.ed.Submit = true
	m.env = annotate.Env{Shaper: m.shaper, Blur: annotate.NewBlurrer(f.Img)}
	if !cfg.Fresh {
		m.last, m.hasLast = loadLastRegion(f)
	}
	return m
}

func (m *monitor) loop() {
	w := new(app.Window)
	m.win.Store(w)
	w.Option(app.Title("screen-editor"), app.FullscreenAt(m.frame.X, m.frame.Y))
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			m.quit()
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			m.layout(gtx, e.Size)
			e.Frame(gtx.Ops)
			if !m.reported {
				m.reported = true
				if m.cfg.Verbose {
					fmt.Printf("монитор (%d,%d) %dx%d: картинка на экране через %s от «хоткея»\n",
						m.frame.X, m.frame.Y, e.Size.X, e.Size.Y, time.Since(m.cfg.T0).Round(time.Millisecond))
				}
			}
		}
	}
}

func (m *monitor) frameSize() image.Point {
	b := m.frame.Img.Bounds()
	return image.Pt(b.Dx(), b.Dy())
}

// layout обрабатывает ввод и рисует один кадр окна. Порядок важен: области
// ввода, зарегистрированные позже, лежат выше (редактор текста и панель
// инструментов выше холста).
func (m *monitor) layout(gtx layout.Context, win image.Point) {
	fs := m.frameSize()
	sx := float32(win.X) / float32(fs.X)
	sy := float32(win.Y) / float32(fs.Y)
	toFrame := func(p f32.Point) f32.Point { return f32.Pt(p.X/sx, p.Y/sy) }

	m.tol = float32(gtx.Dp(8)) / sx
	if m.reset.Swap(false) {
		m.resetState()
	}
	m.handleKeys(gtx)
	m.handlePointer(gtx, toFrame)

	// Холст: область ввода на всё окно.
	area := clip.Rect(image.Rectangle{Max: win}).Push(gtx.Ops)
	event.Op(gtx.Ops, &m.canvas)
	m.cursor().Add(gtx.Ops)
	area.Pop()

	if !m.editing && m.focusCanvas {
		gtx.Execute(key.FocusCmd{Tag: &m.canvas})
		m.focusCanvas = false
	}

	// Сцена в пространстве кадра, масштабируется на окно.
	scale := op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(sx, sy))).Push(gtx.Ops)
	m.drawScene(gtx, fs, sx)
	m.layoutEditor(gtx)
	scale.Pop()

	m.layoutLastButton(gtx, win)

	if m.hasSel {
		selWin := image.Rect(
			int(float32(m.sel.Min.X)*sx), int(float32(m.sel.Min.Y)*sy),
			int(float32(m.sel.Max.X)*sx), int(float32(m.sel.Max.Y)*sy),
		)
		m.layoutToolbar(gtx, win, selWin)
		m.layoutSizeLabel(gtx, selWin)
		m.layoutToast(gtx, win, selWin)
	}
}

// cursor выбирает курсор по положению мыши: ручки рамки, перемещение, прицел.
func (m *monitor) cursor() pointer.Cursor {
	if m.adjust == adjustMove {
		return pointer.CursorGrabbing
	}
	if m.adjust == adjustResize {
		return cursorFor(m.adjEdge)
	}
	if m.hasSel && !m.editing {
		if e, ok := hitHandle(m.sel, m.hover, m.tol, m.tool == annotate.ToolSelect); ok {
			return cursorFor(e)
		}
		if m.tool == annotate.ToolSelect && inRect(m.hover, m.sel) {
			return pointer.CursorGrab
		}
	}
	return pointer.CursorCrosshair
}

// layoutSizeLabel показывает размер выделения рядом с левым верхним углом.
func (m *monitor) layoutSizeLabel(gtx layout.Context, selWin image.Rectangle) {
	txt := fmt.Sprintf("%d × %d", m.sel.Dx(), m.sel.Dy())
	pad := gtx.Dp(6)

	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: iconColor}.Add(gtx.Ops)
	mat := rec.Stop()

	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(1<<12, 1<<12)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, 13, txt, mat)
	label := rec.Stop()

	w, h := dims.Size.X+2*pad, dims.Size.Y+pad
	x, y := selWin.Min.X, selWin.Min.Y-h-gtx.Dp(4)
	if y < 0 {
		y = selWin.Min.Y + gtx.Dp(4)
		x += gtx.Dp(4)
	}
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	bg := clip.UniformRRect(image.Rect(0, 0, w, h), gtx.Dp(4))
	paint.FillShape(gtx.Ops, barBG, bg.Op(gtx.Ops))
	st := op.Offset(image.Pt(pad, pad/2)).Push(gtx.Ops)
	label.Add(gtx.Ops)
	st.Pop()
}

func (m *monitor) drawScene(gtx layout.Context, fs image.Point, sx float32) {
	full := image.Rectangle{Max: fs}

	clipImg := clip.Rect(full).Push(gtx.Ops)
	m.imageOp.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	clipImg.Pop()

	if !m.hasSel && !m.dragging {
		paint.FillShape(gtx.Ops, shade, clip.Rect(full).Op())
		return
	}

	r := m.sel
	for _, side := range []image.Rectangle{
		{Min: full.Min, Max: image.Pt(full.Max.X, r.Min.Y)},
		{Min: image.Pt(full.Min.X, r.Max.Y), Max: full.Max},
		{Min: image.Pt(full.Min.X, r.Min.Y), Max: image.Pt(r.Min.X, r.Max.Y)},
		{Min: image.Pt(r.Max.X, r.Min.Y), Max: image.Pt(full.Max.X, r.Max.Y)},
	} {
		paint.FillShape(gtx.Ops, shade, clip.Rect(side).Op())
	}

	// Аннотации видны только внутри выделения.
	cl := clip.Rect(r).Push(gtx.Ops)
	annotate.DrawAll(gtx.Ops, m.env, m.shapes)
	if m.cur != nil {
		m.cur.Draw(gtx.Ops, m.env)
	}
	cl.Pop()

	bw := 2 / sx
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Max.Y)))
	p.LineTo(f32.Pt(float32(r.Min.X), float32(r.Max.Y)))
	p.Close()
	paint.FillShape(gtx.Ops, selBorder, clip.Stroke{Path: p.End(), Width: bw}.Op())

	// Ручки для изменения размера: 8 квадратиков на рамке.
	hs := float32(5) / sx
	for _, c := range handlePoints(r) {
		outer := image.Rect(int(c.X-hs-1/sx), int(c.Y-hs-1/sx), int(c.X+hs+1/sx), int(c.Y+hs+1/sx))
		inner := image.Rect(int(c.X-hs), int(c.Y-hs), int(c.X+hs), int(c.Y+hs))
		paint.FillShape(gtx.Ops, handleBg, clip.Rect(outer).Op())
		paint.FillShape(gtx.Ops, handleFg, clip.Rect(inner).Op())
	}
}

// layoutEditor показывает редактор текста в позиции editAt. Размеры считаются
// в пикселях кадра, как и у готовой аннотации.
func (m *monitor) layoutEditor(gtx layout.Context) {
	if !m.editing {
		return
	}
	defer op.Offset(image.Pt(int(m.editAt.X), int(m.editAt.Y))).Push(gtx.Ops).Pop()

	size := annotate.TextSize(m.curWidth())
	egtx := gtx
	egtx.Metric = identity
	egtx.Constraints = layout.Constraints{
		Min: image.Pt(int(size*3), int(size*1.4)),
		Max: image.Pt(1<<14, 1<<14),
	}

	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: m.color}.Add(gtx.Ops)
	mat := rec.Stop()
	rec = op.Record(gtx.Ops)
	paint.ColorOp{Color: color.NRGBA{R: 66, G: 133, B: 244, A: 120}}.Add(gtx.Ops)
	selMat := rec.Stop()

	dims := m.ed.Layout(egtx, m.shaper, annotate.Font, unit.Sp(size), mat, selMat)

	// Рамка вокруг редактируемого текста.
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(float32(dims.Size.X), 0))
	p.LineTo(f32.Pt(float32(dims.Size.X), float32(dims.Size.Y)))
	p.LineTo(f32.Pt(0, float32(dims.Size.Y)))
	p.Close()
	paint.FillShape(gtx.Ops, textFrame, clip.Stroke{Path: p.End(), Width: 1}.Op())

	if m.focusEditor {
		gtx.Execute(key.FocusCmd{Tag: &m.ed})
		m.focusEditor = false
	}
}

func (m *monitor) handleKeys(gtx layout.Context) {
	if m.editing {
		for {
			ev, ok := gtx.Event(key.Filter{Focus: &m.ed, Name: key.NameEscape})
			if !ok {
				break
			}
			if e, ok := ev.(key.Event); ok && e.State == key.Press {
				m.quit()
			}
		}
		for {
			ev, ok := m.ed.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				m.commitText()
			}
		}
		return
	}

	for {
		ev, ok := gtx.Event(
			// Без привязки к фокусу: после клика по кнопке панели фокус уходит
			// с холста, а Esc и Enter должны работать всегда.
			key.Filter{Name: key.NameEscape},
			key.Filter{Name: key.NameReturn},
			key.Filter{Name: key.NameEnter},
			key.Filter{Name: "R"},
			key.Filter{Name: "Z", Required: key.ModCtrl},
			key.Filter{Name: "C", Required: key.ModCtrl},
			key.Filter{Name: "S", Required: key.ModCtrl},
		)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		ctrl := e.Modifiers.Contain(key.ModCtrl)
		switch {
		case e.Name == key.NameEscape:
			m.quit()
		case e.Name == key.NameReturn || e.Name == key.NameEnter:
			m.export(true, true)
		case !ctrl && e.Name == "R":
			m.applyLast()
		case ctrl && e.Name == "Z":
			m.undo()
		case ctrl && e.Name == "C":
			m.export(true, false)
		case ctrl && e.Name == "S":
			m.export(false, true)
		}
	}
}

func (m *monitor) handlePointer(gtx layout.Context, toFrame func(f32.Point) f32.Point) {
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: &m.canvas, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Move},
			pointer.Filter{Target: &m.canvas, Kinds: pointer.Scroll, ScrollY: pointer.ScrollRange{Min: -100, Max: 100}},
		)
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		p := toFrame(e.Position)
		m.hover = p
		switch e.Kind {
		case pointer.Press:
			if e.Buttons.Contain(pointer.ButtonPrimary) {
				m.press(p)
			}
		case pointer.Drag:
			m.drag(p)
		case pointer.Release:
			m.release(p)
		case pointer.Scroll:
			m.scroll(e.Scroll.Y)
		}
	}
}

// claim забирает единственную активную область себе: у остальных окон
// выделение и рисунки сбрасываются.
func (m *monitor) claim() {
	for _, o := range m.peers {
		if o == m {
			continue
		}
		o.reset.Store(true)
		if w := o.win.Load(); w != nil {
			w.Invalidate()
		}
	}
}

// resetState убирает выделение, рисунки и редактирование текста.
func (m *monitor) resetState() {
	m.dragging = false
	m.adjust = adjustNone
	m.hasSel = false
	m.sel = image.Rectangle{}
	m.shapes = nil
	m.cur = nil
	m.tool = annotate.ToolSelect
	if m.editing {
		m.cancelText()
	}
}

func (m *monitor) setTool(t annotate.Tool) {
	if m.editing {
		m.commitText()
	}
	m.tool = t
}

func (m *monitor) press(p f32.Point) {
	if m.editing {
		m.commitText()
	}
	if m.hasSel {
		// Ручки рамки важнее любого инструмента.
		if e, ok := hitHandle(m.sel, p, m.tol, m.tool == annotate.ToolSelect); ok {
			m.claim()
			m.adjust, m.adjEdge, m.adjStartSel = adjustResize, e, m.sel
			return
		}
		// Перетаскивание готовой области за её внутреннюю часть.
		if m.tool == annotate.ToolSelect && inRect(p, m.sel) {
			m.claim()
			m.adjust, m.adjStartSel, m.adjStartPt = adjustMove, m.sel, p
			return
		}
	}
	if !m.hasSel || m.tool == annotate.ToolSelect {
		m.claim()
		m.dragging = true
		m.selStart = p
		m.sel = rectOf(p, p)
		m.hasSel = false
		return
	}
	if !inRect(p, m.sel) {
		return
	}
	switch m.tool {
	case annotate.ToolText:
		m.editing = true
		m.editAt = p
		m.ed.SetText("")
		m.focusEditor = true
	case annotate.ToolPen, annotate.ToolMarker:
		m.cur = &annotate.Shape{Tool: m.tool, Color: m.color, Width: m.curWidth(), Pts: []f32.Point{p}}
	default:
		m.cur = &annotate.Shape{Tool: m.tool, Color: m.color, Width: m.curWidth(), Pts: []f32.Point{p, p}}
	}
}

func (m *monitor) drag(p f32.Point) {
	switch {
	case m.adjust == adjustResize:
		m.sel = applyResize(m.adjStartSel, m.adjEdge, p, m.frame.Img.Bounds())
	case m.adjust == adjustMove:
		m.sel = applyMove(m.adjStartSel, int(p.X-m.adjStartPt.X), int(p.Y-m.adjStartPt.Y), m.frame.Img.Bounds())
	case m.dragging:
		m.sel = rectOf(m.selStart, p)
	case m.cur != nil:
		p = m.clamp(p)
		switch m.cur.Tool {
		case annotate.ToolPen, annotate.ToolMarker:
			m.cur.Pts = append(m.cur.Pts, p)
		default:
			m.cur.Pts[1] = p
		}
	}
}

func (m *monitor) release(p f32.Point) {
	switch {
	case m.adjust != adjustNone:
		m.adjust = adjustNone
	case m.dragging:
		m.dragging = false
		m.sel = rectOf(m.selStart, p).Intersect(m.frame.Img.Bounds())
		// Простой клик без выделения не считаем областью.
		m.hasSel = m.sel.Dx() >= 4 && m.sel.Dy() >= 4
		if !m.hasSel {
			m.sel = image.Rectangle{}
		}
	case m.cur != nil:
		if !m.cur.Degenerate() {
			m.shapes = append(m.shapes, *m.cur)
		}
		m.cur = nil
	}
}

func (m *monitor) scroll(dy float32) {
	switch {
	case dy < 0:
		m.bumpWidth(1)
	case dy > 0:
		m.bumpWidth(-1)
	}
}

// curWidth возвращает толщину (для размытия силу) текущего инструмента.
func (m *monitor) curWidth() float32 { return m.widthOf(m.tool) }

// widthOf возвращает толщину инструмента или значение по умолчанию.
func (m *monitor) widthOf(t annotate.Tool) float32 {
	if w, ok := m.widths[t]; ok {
		return w
	}
	return annotate.DefaultWidth
}

// bumpWidth меняет толщину текущего инструмента. Фигура, которую рисуют
// прямо сейчас, и редактируемый текст обновляются сразу.
func (m *monitor) bumpWidth(delta float32) {
	if m.tool == annotate.ToolSelect {
		return
	}
	w := float32(math.Min(math.Max(float64(m.curWidth()+delta), annotate.MinWidth), annotate.MaxWidth))
	m.widths[m.tool] = w
	if m.cur != nil {
		m.cur.Width = w
	}
	m.showToast()
}

func (m *monitor) clamp(p f32.Point) f32.Point {
	r := m.sel
	p.X = float32(math.Min(math.Max(float64(p.X), float64(r.Min.X)), float64(r.Max.X)))
	p.Y = float32(math.Min(math.Max(float64(p.Y), float64(r.Min.Y)), float64(r.Max.Y)))
	return p
}

func (m *monitor) undo() {
	if m.editing {
		m.cancelText()
		return
	}
	if n := len(m.shapes); n > 0 {
		m.shapes = m.shapes[:n-1]
	}
}

func (m *monitor) commitText() {
	txt := strings.TrimSpace(m.ed.Text())
	if txt != "" {
		m.shapes = append(m.shapes, annotate.Shape{
			Tool: annotate.ToolText, Color: m.color, Width: m.widthOf(annotate.ToolText),
			Pts: []f32.Point{m.editAt}, Text: txt,
		})
	}
	m.cancelText()
}

func (m *monitor) cancelText() {
	m.editing = false
	m.ed.SetText("")
	m.focusCanvas = true
}

// export рендерит выделение с аннотациями, при необходимости сохраняет в файл
// и копирует в буфер обмена, затем завершает работу.
func (m *monitor) export(toClipboard, toFile bool) {
	if m.editing {
		m.commitText()
	}
	if !m.hasSel {
		return
	}
	img, err := render(m.frame.Img, m.sel, m.shapes, m.shaper)
	if err != nil {
		fail("рендер", err)
		return
	}
	data, err := encodePNG(img)
	if err != nil {
		fail("PNG", err)
		return
	}

	msg := ""
	if toFile {
		path, err := saveFile(data)
		if err != nil {
			fail("сохранение", err)
			return
		}
		msg = path
	}
	if toClipboard {
		if err := copyPNG(data); err != nil {
			fail("буфер обмена", err)
		} else if msg == "" {
			msg = "скопирован в буфер обмена"
		} else {
			msg += " (и в буфер обмена)"
		}
	}
	notify("Скриншот готов", msg)
	m.quit()
}

// quit запоминает область (если она есть) и завершает работу. Область
// запоминается при любом закрытии, а не только после успешного скриншота.
func (m *monitor) quit() {
	if m.hasSel {
		saveLastRegion(m.frame, m.sel)
	}
	m.finish()
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	notify("Ошибка: "+what, err.Error())
}

func rectOf(a, b f32.Point) image.Rectangle {
	return image.Rect(
		int(math.Min(float64(a.X), float64(b.X))), int(math.Min(float64(a.Y), float64(b.Y))),
		int(math.Max(float64(a.X), float64(b.X))), int(math.Max(float64(a.Y), float64(b.Y))),
	)
}

func inRect(p f32.Point, r image.Rectangle) bool {
	return p.X >= float32(r.Min.X) && p.X < float32(r.Max.X) &&
		p.Y >= float32(r.Min.Y) && p.Y < float32(r.Max.Y)
}
