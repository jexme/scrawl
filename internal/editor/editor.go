// Package editor показывает замороженный кадр на весь монитор, даёт выделить
// область, нарисовать аннотации и сохранить результат.
package editor

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/collection"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/keymap"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
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
	handleFg  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	textFrame = color.NRGBA{R: 255, G: 255, B: 255, A: 160}
	identity  = unit.Metric{PxPerDp: 1, PxPerSp: 1}
)

// shadeColor возвращает цвет затемнения вне области для процента 0-100.
func shadeColor(percent int) color.NRGBA {
	return color.NRGBA{A: uint8(math.Round(float64(min(max(percent, 0), 100)) * 255 / 100))}
}

// Режимы правки готовой области.
const (
	adjustNone = iota
	adjustResize
	adjustMove
)

// Run открывает по fullscreen-окну на каждый кадр. Вызывающий должен после
// этого вызвать app.Main().
//
// Возвращает функцию, которая закрывает все окна так же, как их закрывает пользователь.
func Run(cfg Config) (closeAll func()) {
	app.ID = screencast.AppID
	var once sync.Once
	finish := func() { once.Do(cfg.Finish) }
	monitors := make([]*monitor, len(cfg.Frames))
	for i := range cfg.Frames {
		monitors[i] = newMonitor(cfg, &cfg.Frames[i], finish)
	}
	alive := new(atomic.Int32)
	alive.Store(int32(len(monitors)))
	store := settings.NewStore() // общие настройки для всех окон
	for _, m := range monitors {
		m.peers = monitors
		m.alive = alive
		m.store = store
		go m.loop()
	}
	if len(monitors) == 0 {
		return finish
	}
	return monitors[0].closeWindows
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
	alive *atomic.Int32 // сколько окон ещё не закрыто
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

	// Настройки (общие для окон) и двойной клик.
	store        *settings.Store
	lastClickAt  time.Time
	lastClickPos f32.Point

	// Подборка скриншотов («было/стало»).
	queued int // сколько картинок накоплено на диске
	dock   dock

	// Положение элементов в пикселях окна: нужно, чтобы док не перекрывал выделение и панель.
	selWin  image.Rectangle
	barRect image.Rectangle

	// Плашка с новым значением силы инструмента.
	toastUntil time.Time
	toastTool  annotate.Tool
	toastValue int
	toastMsg   string // готовый текст плашки вместо «Толщина: N»

	// Область предыдущего скриншота: не подставляется сама, возвращается по кнопке или клавише R.
	last    image.Rectangle
	hasLast bool
	lastBtn widget.Clickable
	tol     float32 // допуск попадания в ручку, пиксели кадра

	tool    annotate.Tool
	color   color.NRGBA
	keys    []keyBinding                      // клавиши оверлея по настройкам
	palette [settings.PaletteSize]color.NRGBA // цвета панели из настроек
	// colorPicked: цвет выбран в панели вручную, цвета инструментов из настроек больше не применяются.
	colorPicked bool
	widths      map[annotate.Tool]float32 // толщина или сила для каждого инструмента отдельно
	shapes      []annotate.Shape
	cur         *annotate.Shape

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
		widths:      make(map[annotate.Tool]float32),
		focusCanvas: true,
	}
	m.ed.Submit = true
	m.env = annotate.Env{Shaper: m.shaper, Blur: annotate.NewBlurrer(f.Img)}
	m.queued = collection.Count()
	m.store = settings.NewStore()
	m.initKeys()
	m.palette = m.store.Get().PaletteColors()
	m.color = m.palette[0]
	if !cfg.Fresh && m.store.Get().RememberRegion {
		m.last, m.hasLast = loadLastRegion(f)
		// Если собирают подборку («было/стало»), снимать «стало» нужно в том же
		// месте: прошлая область выставляется сразу.
		if m.hasLast && m.queued > 0 {
			m.sel, m.hasSel = m.last, true
			m.useDefaultTool()
		}
	}
	return m
}

func (m *monitor) loop() {
	w := new(app.Window)
	m.win.Store(w)
	w.Option(app.Title("scrawl"), app.FullscreenAt(m.frame.X, m.frame.Y))
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			// Окно закрыто (нами или композитором): запоминаем область, закрываем
			// остальные окна и завершаем работу, когда закрыты все.
			m.rememberRegion()
			if m.alive.Add(-1) <= 0 {
				m.finish()
			} else {
				m.closeWindows()
			}
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

	m.selWin, m.barRect = image.Rectangle{}, image.Rectangle{}
	if m.hasSel {
		m.selWin = image.Rect(
			int(float32(m.sel.Min.X)*sx), int(float32(m.sel.Min.Y)*sy),
			int(float32(m.sel.Max.X)*sx), int(float32(m.sel.Max.Y)*sy),
		)
		m.layoutToolbar(gtx, win, m.selWin)
		if m.store.Get().ShowSize {
			m.layoutSizeLabel(gtx, m.selWin)
		}
		m.layoutToast(gtx, win, m.selWin)
	}
	m.layoutDock(gtx, win)
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
	cfg := m.store.Get()
	shade := shadeColor(cfg.DimPercent)

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

	// Рамка пунктиром: толщина, штрих и пауза в пикселях экрана, при любом масштабе
	// окна остаются теми же.
	dashedRect(gtx.Ops, r, cfg.BorderNRGBA(), float32(cfg.BorderWidth)/sx, float32(cfg.BorderDash)/sx, float32(cfg.BorderGap)/sx)

	// Ручки только в углах: за стороны рамку тянут инструментом «выделение».
	if hr := float32(cfg.HandleRadius) / sx; hr > 0 {
		for _, c := range cornerPoints(r) {
			dot := clip.Ellipse(image.Rect(int(c.X-hr), int(c.Y-hr), int(c.X+hr), int(c.Y+hr)))
			paint.FillShape(gtx.Ops, handleFg, dot.Op(gtx.Ops))
		}
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
		ev, ok := gtx.Event(m.keyFilters()...)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		for _, kb := range m.keys {
			if kb.binding.Matches(e) {
				kb.run()
				break
			}
		}
	}
}

// keyBinding — клавиша оверлея и действие, которое она запускает.
type keyBinding struct {
	action  string
	binding keymap.Binding
	run     func()
}

// initKeys составляет таблицу клавиш по настройкам. Клавиши назначены на
// запуск оверлея: пока он открыт, настройки не меняются.
func (m *monitor) initKeys() {
	keys := m.store.Get().Keys
	actions := map[string]func(){
		settings.ActionConfirm: func() { m.export(m.store.Get().Enter.Targets()) },
		settings.ActionCancel:  m.quit,
		settings.ActionRestore: m.applyLast,
		settings.ActionAdd:     m.addToCollection,
		settings.ActionGlue: func() {
			if m.queued+boolInt(m.hasSel) >= 2 {
				m.glueCollection()
			}
		},
		settings.ActionSend: func() {
			if m.queued+boolInt(m.hasSel) >= 1 {
				m.sendSeparately()
			}
		},
		settings.ActionUndo: m.undo,
		settings.ActionCopy: func() { m.export(true, false) },
		settings.ActionSave: func() { m.export(false, true) },
	}
	actions[settings.ActionWidthUp] = func() { m.bumpWidth(1) }
	actions[settings.ActionWidthDown] = func() { m.bumpWidth(-1) }
	// Клавиши инструментов работают, когда область выбрана: иначе выбор сбросился бы на
	// инструмент по умолчанию в момент выделения.
	for t, name := range settings.ToolNames {
		tool := annotate.Tool(t)
		actions[settings.ToolAction(name)] = func() {
			if m.hasSel {
				m.setTool(tool)
			}
		}
	}
	m.keys = nil
	for _, action := range settings.KeyActions {
		b, ok := keymap.Parse(keys.Of(action))
		if !ok {
			b, _ = keymap.Parse(settings.DefaultKeys[action])
		}
		m.keys = append(m.keys, keyBinding{action: action, binding: b, run: actions[action]})
	}
}

// keyFilters возвращает фильтры событий для всех клавиш оверлея. Без привязки к
// фокусу: после клика по кнопке панели фокус уходит с холста, а клавиши должны
// работать всегда.
func (m *monitor) keyFilters() []event.Filter {
	var fs []event.Filter
	for _, kb := range m.keys {
		for _, f := range kb.binding.Filters() {
			fs = append(fs, f)
		}
	}
	return fs
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
	// Цвет инструмента из настроек действует, пока цвет не выбрали в панели вручную.
	if st := m.toolStyle(t); st != nil && !m.colorPicked {
		if c, err := settings.ParseColor(st.Color); err == nil {
			m.color = c
		}
	}
}

// toolName возвращает имя инструмента в настройках.
func toolName(t annotate.Tool) string {
	if int(t) < len(settings.ToolNames) {
		return settings.ToolNames[t]
	}
	return ""
}

// toolStyle возвращает вид инструмента по умолчанию из настроек (nil для «выделения»).
func (m *monitor) toolStyle(t annotate.Tool) *settings.ToolStyle {
	tools := m.store.Get().Tools
	return tools.Of(toolName(t))
}

// useDefaultTool включает инструмент по умолчанию из настроек: вызывается,
// когда область выбрана и можно рисовать.
func (m *monitor) useDefaultTool() {
	want := m.store.Get().DefaultTool
	for t, name := range settings.ToolNames {
		if name == want {
			m.setTool(annotate.Tool(t))
			return
		}
	}
}

func (m *monitor) press(p f32.Point) {
	if m.editing {
		m.commitText()
	}
	if m.isDoubleClick(p) && m.doubleClickAction(p) {
		return
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
		m.cur = &annotate.Shape{Tool: m.tool, Color: m.color, Width: m.curWidth(), Pts: []f32.Point{p}, Alpha: m.store.Get().MarkerAlpha()}
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
		} else {
			m.useDefaultTool()
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
	if st := m.toolStyle(t); st != nil && st.Width > 0 {
		return float32(st.Width)
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
// и копирует в буфер обмена, затем завершает работу (или остаётся в редакторе,
// если в настройках включено «не закрывать после копирования»).
func (m *monitor) export(toClipboard, toFile bool) {
	if m.editing {
		m.commitText()
	}
	if !m.hasSel {
		return
	}
	img, err := render(m.frame.Img, m.sel, m.shapes, m.shaper)
	if err != nil {
		fail(i18n.T("рендер"), err)
		return
	}
	data, err := encodePNG(img)
	if err != nil {
		fail("PNG", err)
		return
	}

	msg, done := "", ""
	if toFile {
		path, err := saveFile(m.store.Get(), data)
		if err != nil {
			fail(i18n.T("сохранение"), err)
			return
		}
		msg, done = path, i18n.T("Сохранено: ")+filepath.Base(path)
	}
	if toClipboard {
		if err := copyPNG(data); err != nil {
			fail(i18n.T("буфер обмена"), err)
		} else if msg == "" {
			msg, done = i18n.T("скопирован в буфер обмена"), i18n.T("Скопировано в буфер обмена")
		} else {
			msg += i18n.T(" (и в буфер обмена)")
			done += i18n.T(", скопировано")
		}
	}
	m.inform(i18n.T("Скриншот готов"), msg)
	if m.store.Get().KeepOpen {
		m.showMessage(done) // остаёмся в редакторе: подтверждаем действие на экране
		return
	}
	m.quit()
}

// quit запоминает область (если она есть) и завершает работу. Область
// запоминается при любом закрытии, а не только после успешного скриншота.
func (m *monitor) quit() {
	m.rememberRegion()
	m.closeWindows()
}

// closeWindows закрывает окна по-человечески и завершает работу, когда они
// закрылись: если выйти из процесса при открытых полноэкранных окнах, GNOME
// подвисает на секунду-две, что заметно в играх. Если окна не закрылись за
// 800 мс, завершаем принудительно.
func (m *monitor) closeWindows() {
	closed := 0
	for _, p := range m.peers {
		if w := p.win.Load(); w != nil {
			w.Perform(system.ActionClose)
			closed++
		}
	}
	if closed == 0 {
		m.finish() // окон нет (например, в тестах): закрывать нечего
		return
	}
	time.AfterFunc(800*time.Millisecond, m.finish)
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	notify(i18n.T("Ошибка: ")+what, err.Error())
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

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
