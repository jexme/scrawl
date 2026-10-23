// Package editor shows a frozen frame across the whole monitor, lets the user select a
// region, draw annotations and save the result.
package editor

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
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
	"github.com/jexme/scrawl/internal/windowerr"
)

// Config holds the editor launch parameters.
type Config struct {
	Frames  []screencast.Frame
	Finish  func()    // releases resources and exits the process
	T0      time.Time // the moment of the "hotkey press", for timing measurements
	Verbose bool
	Fresh   bool // do not suggest the region of the previous screenshot
	Hint    bool // show the first-launch hint while nothing is selected
}

var (
	handleFg  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	textFrame = color.NRGBA{R: 255, G: 255, B: 255, A: 160}
	identity  = unit.Metric{PxPerDp: 1, PxPerSp: 1}
)

// shadeColor returns the dimming colour outside the selection for a percentage of 0-100.
func shadeColor(percent int) color.NRGBA {
	return color.NRGBA{A: uint8(math.Round(float64(min(max(percent, 0), 100)) * 255 / 100))}
}

// Editing modes for a finished region.
const (
	adjustNone = iota
	adjustResize
	adjustMove
)

// Run opens one fullscreen window per frame. The caller must call app.Main() afterwards.
//
// It returns a function that closes all windows the same way the user closes them.
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
	store := settings.NewStore() // settings shared by all windows
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

// monitor is the overlay window of one monitor with its own editing state.
type monitor struct {
	cfg    Config
	frame  *screencast.Frame
	finish func()
	shaper *text.Shaper
	env    annotate.Env // font and blur for drawing annotations
	bar    *toolbar

	// peers are all overlay windows (including this one). There is one active region across
	// all monitors: when a selection starts in one window, the others are reset.
	peers []*monitor
	alive *atomic.Int32 // how many windows are not closed yet
	win   atomic.Pointer[app.Window]
	reset atomic.Bool // another window took the selection, so this one must reset its own

	imageOp paint.ImageOp
	loupeOp paint.ImageOp // the same frame with nearest-pixel sampling, created when the magnifier is first shown
	canvas  int           // canvas tag

	// Selection in frame pixels.
	sel      image.Rectangle
	hasSel   bool
	dragging bool
	selStart f32.Point

	// Resizing and moving the finished region.
	adjust      int
	adjEdge     edge
	adjStartSel image.Rectangle
	adjStartPt  f32.Point
	hover       f32.Point // last cursor position, in frame pixels

	// Accumulated wheel or touchpad scroll and the time of the last thickness step.
	scrollAcc  float32
	scrollSeen time.Time
	scrollStep time.Time
	clock      func() time.Time // time source, replaced in tests
	hoverIn    bool             // the cursor is over this window now (the magnifier is not needed on another monitor)

	// Settings (shared between windows) and double click.
	store        *settings.Store
	lastClickAt  time.Time
	lastClickPos f32.Point

	// Collection of screenshots ("before/after").
	queued int // how many images have accumulated on disk
	dock   dock

	// Positions of elements in window pixels: needed so the dock overlaps neither the
	// selection nor the toolbar.
	selWin   image.Rectangle
	barRect  image.Rectangle // the whole toolbar (both rows if the second one is visible)
	mainRow  image.Rectangle // main row
	barSpace image.Rectangle // room for both rows even if the second is hidden: the dock does not overlap it
	barBelow bool            // toolbar is below the selection (second row below the main one), otherwise above it

	// Menu "⋯" on the toolbar.
	menuOpen   bool
	menuAnchor int // centre of the "⋯" button in window pixels

	// Toast with the new value of the tool strength.
	toastUntil time.Time
	toastTool  annotate.Tool
	toastValue int
	toastMsg   string // ready-made toast text instead of "Thickness: N"

	// Region of the previous screenshot: it is not applied automatically, it comes back with a
	// button or the R key.
	last    image.Rectangle
	hasLast bool
	lastBtn widget.Clickable
	tol     float32 // hit tolerance for a handle, in frame pixels

	tool  annotate.Tool
	color color.NRGBA
	// Custom colour from the toolbar palette: remembered in settings. pick is its chooser
	// popup.
	customColor color.NRGBA
	hasCustom   bool
	pick        picker
	pickAnchor  int                               // centre of the custom colour button in window pixels
	keys        []keyBinding                      // overlay keys from settings
	palette     [settings.PaletteSize]color.NRGBA // toolbar colours from settings
	// colorPicked: the colour was chosen manually in the toolbar, the tool colours from
	// settings no longer apply.
	colorPicked bool
	widths      map[annotate.Tool]float32 // thickness or strength for each tool separately
	shapes      []annotate.Shape
	cur         *annotate.Shape
	stroke      []f32.Point // the raw path of the pen or marker stroke being drawn
	// before is the selection and annotations that a new selection drag replaced: a click
	// without dragging brings them back instead of leaving the user with nothing.
	before struct {
		sel    image.Rectangle
		has    bool
		shapes []annotate.Shape
	}
	shift bool // Shift is held (the last pointer event): straight lines and squares

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
	if c, err := settings.ParseColor(m.store.Get().CustomColor); err == nil {
		m.customColor, m.hasCustom = c, true
	}
	if !cfg.Fresh && m.store.Get().RememberRegion {
		m.last, m.hasLast = loadLastRegion(f)
		// If a collection ("before/after") is being assembled, the "after" shot has to be taken in
		// the same place: the previous region is applied right away.
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
			windowerr.Report(e.Err) // e.Err != nil: the window failed; do not exit silently
			// The window was closed (by us or by the compositor): remember the region, close the other
			// windows and exit once all of them are closed.
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
					fs := m.frameSize()
					fmt.Printf("monitor (%d,%d) window %dx%d, frame %dx%d: picture on screen %s after the hotkey\n",
						m.frame.X, m.frame.Y, e.Size.X, e.Size.Y, fs.X, fs.Y, time.Since(m.cfg.T0).Round(time.Millisecond))
				}
			}
		}
	}
}

func (m *monitor) frameSize() image.Point {
	b := m.frame.Img.Bounds()
	return image.Pt(b.Dx(), b.Dy())
}

// layout handles input and draws one frame of the window. Order matters: input areas
// registered later lie above (the text editor and the toolbar are above the canvas).
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

	// Canvas: an input area covering the whole window.
	area := clip.Rect(image.Rectangle{Max: win}).Push(gtx.Ops)
	event.Op(gtx.Ops, &m.canvas)
	m.cursor().Add(gtx.Ops)
	area.Pop()

	if !m.editing && m.focusCanvas {
		gtx.Execute(key.FocusCmd{Tag: &m.canvas})
		m.focusCanvas = false
	}

	// The scene is in frame space and is scaled to the window.
	scale := op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(sx, sy))).Push(gtx.Ops)
	m.drawScene(gtx, fs, sx)
	m.layoutEditor(gtx)
	scale.Pop()

	m.selWin, m.barRect, m.mainRow, m.barSpace = image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, image.Rectangle{}
	if m.hasSel {
		m.selWin = image.Rect(
			int(float32(m.sel.Min.X)*sx), int(float32(m.sel.Min.Y)*sy),
			int(float32(m.sel.Max.X)*sx), int(float32(m.sel.Max.Y)*sy),
		)
		m.layoutToolbar(gtx, win, m.selWin)
		m.layoutPicker(gtx, win, m.barRect, m.pickAnchor)
		if m.store.Get().ShowSize {
			m.layoutSizeLabel(gtx, m.selWin)
		}
		m.layoutToast(gtx, win, m.selWin)
	}
	if m.cfg.Hint && !m.hasSel && !m.dragging && m.alone() {
		m.layoutHint(gtx, win)
	}
	m.layoutDock(gtx, win)
	m.layoutLoupe(gtx, win, sx, sy)
}

// cursor chooses the cursor by the mouse position: border handles, move, crosshair.
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

// layoutSizeLabel shows the selection size next to the top left corner.
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

	// Annotations are visible only inside the selection.
	cl := clip.Rect(r).Push(gtx.Ops)
	annotate.DrawAll(gtx.Ops, m.env, m.shapes)
	if m.cur != nil {
		m.cur.Draw(gtx.Ops, m.env)
	}
	cl.Pop()

	// Dashed border: thickness, dash and gap are in screen pixels and stay the same at any
	// window scale.
	dashedRect(gtx.Ops, r, cfg.BorderNRGBA(), float32(cfg.BorderWidth)/sx, float32(cfg.BorderDash)/sx, float32(cfg.BorderGap)/sx)

	// Handles only at the corners: the sides of the border are dragged with the "Select" tool.
	if hr := float32(cfg.HandleRadius) / sx; hr > 0 {
		for _, c := range cornerPoints(r) {
			dot := clip.Ellipse(image.Rect(int(c.X-hr), int(c.Y-hr), int(c.X+hr), int(c.Y+hr)))
			paint.FillShape(gtx.Ops, handleFg, dot.Op(gtx.Ops))
		}
	}
}

// layoutEditor shows the text editor at position editAt. Sizes are computed in frame
// pixels, like those of a finished annotation.
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

	// Border around the text being edited.
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
				m.cancelText() // Esc drops the text being typed, not the whole overlay
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

// keyBinding is an overlay key and the action it triggers.
type keyBinding struct {
	action  string
	binding keymap.Binding
	run     func()
}

// initKeys builds the key table from settings. Keys are assigned at overlay launch:
// settings do not change while it is open.
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
	// Tool keys work when a region is selected: otherwise the choice would reset to the
	// default tool at the moment of selection.
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

// keyFilters returns event filters for all overlay keys. They are not bound to focus:
// after a click on a toolbar button focus leaves the canvas, but the keys must always
// work.
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
			pointer.Filter{Target: &m.canvas, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Move | pointer.Enter | pointer.Leave | pointer.Cancel},
			pointer.Filter{Target: &m.canvas, Kinds: pointer.Scroll, ScrollY: pointer.ScrollRange{Min: -100, Max: 100}},
		)
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Kind == pointer.Cancel {
			// The cursor left the window (for example, onto the overlay of another monitor): Cancel
			// has no position, so we do not overwrite the last one, but hide the magnifier.
			m.hoverIn = false
			continue
		}
		p := toFrame(e.Position)
		m.hover = p
		m.hoverIn = e.Kind != pointer.Leave
		m.shift = e.Modifiers.Contain(key.ModShift)
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

// claim takes the single active region for itself: the selection and drawings of the other
// windows are reset.
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

// resetState clears the selection, the drawings and text editing.
func (m *monitor) resetState() {
	m.pick.open, m.menuOpen = false, false
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
	// The tool colour from settings applies until a colour is chosen manually in the toolbar.
	if st := m.toolStyle(t); st != nil && !m.colorPicked {
		if c, err := settings.ParseColor(st.Color); err == nil {
			m.color = c
		}
	}
}

// toolName returns the tool name in settings.
func toolName(t annotate.Tool) string {
	if int(t) < len(settings.ToolNames) {
		return settings.ToolNames[t]
	}
	return ""
}

// toolStyle returns the default tool style from settings (nil for "Select").
func (m *monitor) toolStyle(t annotate.Tool) *settings.ToolStyle {
	tools := m.store.Get().Tools
	return tools.Of(toolName(t))
}

// useDefaultTool enables the default tool from settings: it is called when a region is
// selected and drawing is possible.
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
	if m.pick.open || m.menuOpen {
		m.pick.open, m.menuOpen = false, false // a click outside the colour popup or menu closes it
		return
	}
	if m.editing {
		m.commitText()
	}
	if m.isDoubleClick(p) && m.doubleClickAction(p) {
		return
	}
	if m.hasSel {
		// Border handles take priority over any tool.
		if e, ok := hitHandle(m.sel, p, m.tol, m.tool == annotate.ToolSelect); ok {
			m.claim()
			m.adjust, m.adjEdge, m.adjStartSel = adjustResize, e, m.sel
			return
		}
		// Dragging the finished region by its inner part.
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
		m.before.sel, m.before.has, m.before.shapes = m.sel, m.hasSel, m.shapes
		m.shapes, m.cur = nil, nil // annotations belong to the old area, a new area starts clean
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
		m.stroke = []f32.Point{p}
		m.cur = &annotate.Shape{Tool: m.tool, Color: m.color, Width: m.curWidth(), Pts: m.stroke, Alpha: m.store.Get().MarkerAlpha()}
	case annotate.ToolStamp:
		// The stamp is placed under the cursor immediately; while the button is held, it can be
		// moved.
		m.cur = &annotate.Shape{Tool: m.tool, Color: m.color, Width: m.curWidth(), Pts: []f32.Point{p}, Text: strconv.Itoa(m.nextStamp())}
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
		if t := m.cur.Tool; t == annotate.ToolPen || t == annotate.ToolMarker {
			m.stroke = append(m.stroke, p)
		}
		m.applyModifiers(p)
	}
}

// applyModifiers sets the shape being drawn from its anchor and the cursor position p,
// taking Shift into account: a straight line for the pen and the marker, a 45-degree
// snap for a line and an arrow, a square for a box and a blur.
func (m *monitor) applyModifiers(p f32.Point) {
	p = m.clamp(p)
	switch m.cur.Tool {
	case annotate.ToolPen, annotate.ToolMarker:
		if len(m.stroke) < 2 {
			return
		}
		if m.shift {
			// The raw path is kept, so releasing Shift brings the freehand stroke back.
			m.cur.Pts = []f32.Point{m.stroke[0], snapAngle(m.stroke[0], m.stroke[len(m.stroke)-1], m.sel)}
		} else {
			m.cur.Pts = m.stroke
		}
	case annotate.ToolStamp:
		m.cur.Pts[0] = p
	case annotate.ToolLine, annotate.ToolArrow:
		if m.shift {
			p = snapAngle(m.cur.Pts[0], p, m.sel)
		}
		m.cur.Pts[1] = p
	default: // box and blur
		if m.shift {
			p = squareTo(m.cur.Pts[0], p, m.sel)
		}
		m.cur.Pts[1] = p
	}
}

func (m *monitor) release(p f32.Point) {
	switch {
	case m.adjust != adjustNone:
		m.adjust = adjustNone
	case m.dragging:
		m.dragging = false
		m.sel = rectOf(m.selStart, p).Intersect(m.frame.Img.Bounds())
		// A plain click without dragging is not counted as a region.
		m.hasSel = m.sel.Dx() >= 4 && m.sel.Dy() >= 4
		switch {
		case m.hasSel:
			m.useDefaultTool()
		case m.before.has:
			// A click without dragging: keep the previous selection and its annotations.
			m.sel, m.hasSel, m.shapes = m.before.sel, true, m.before.shapes
		default:
			m.sel = image.Rectangle{}
		}
		m.before.has, m.before.shapes = false, nil
	case m.cur != nil:
		m.applyModifiers(p) // Shift pressed or released without moving the mouse
		if !m.cur.Degenerate() {
			m.shapes = append(m.shapes, *m.cur)
		}
		m.cur = nil
	}
}

// Scrolling changes the thickness in steps. A wheel sends one event per click, while a
// touchpad sends dozens of small ones per second, plus inertia after the finger is lifted:
// if the thickness changed on every event, it would fly from 1 to 6 in a single gesture.
const (
	scrollNotch     = 8                      // an event at least this large counts as a wheel click: step immediately
	scrollUnit      = 40                     // the chunk of small events (touchpad) that makes one step
	scrollIdle      = 300 * time.Millisecond // after such a pause the accumulated amount is forgotten
	scrollGapFine   = 50 * time.Millisecond  // the shortest gap between steps for tools with a fine scale
	scrollGapCoarse = 250 * time.Millisecond // the same for blur: it has only six levels
)

// scroll accumulates the scroll dy and moves the thickness by one step per chunk. A wheel
// click (one large event) is a whole chunk, small touchpad events accumulate up to a
// chunk. There is a pause between steps; the excess during the pause is not accumulated,
// and the remainder after a step is discarded: touchpad inertia does not produce extra
// steps.
func (m *monitor) scroll(dy float32) {
	now := m.now()
	if now.Sub(m.scrollSeen) > scrollIdle || m.scrollAcc*dy < 0 { // pause or change of direction
		m.scrollAcc = 0
	}
	m.scrollSeen = now
	if dy <= -scrollNotch || dy >= scrollNotch {
		m.scrollAcc = float32(math.Copysign(scrollUnit, float64(dy)))
	} else {
		m.scrollAcc += dy
	}
	if m.scrollAcc > -scrollUnit && m.scrollAcc < scrollUnit {
		return
	}
	gap := scrollGapFine
	if annotate.MaxWidthOf(m.tool) <= annotate.MaxBlurLevel {
		gap = scrollGapCoarse
	}
	if now.Sub(m.scrollStep) < gap {
		m.scrollAcc = float32(math.Copysign(scrollUnit, float64(m.scrollAcc))) // the chunk is ready, waiting for the pause
		return
	}
	if m.scrollAcc < 0 {
		m.bumpWidth(1)
	} else {
		m.bumpWidth(-1)
	}
	m.scrollAcc, m.scrollStep = 0, now
}

// now returns the current time; it is replaced in tests.
func (m *monitor) now() time.Time {
	if m.clock != nil {
		return m.clock()
	}
	return time.Now()
}

// curWidth returns the thickness (the strength for blur) of the current tool.
func (m *monitor) curWidth() float32 { return m.widthOf(m.tool) }

// widthOf returns the thickness of the tool or the default value.
func (m *monitor) widthOf(t annotate.Tool) float32 {
	if w, ok := m.widths[t]; ok {
		return w
	}
	if st := m.toolStyle(t); st != nil && st.Width > 0 {
		return min(float32(st.Width), annotate.MaxWidthOf(t))
	}
	return annotate.DefaultWidthOf(t)
}

// bumpWidth changes the thickness of the current tool. The shape being drawn right now and
// the text being edited are updated immediately.
func (m *monitor) bumpWidth(delta float32) {
	if m.tool == annotate.ToolSelect {
		return
	}
	w := float32(math.Min(math.Max(float64(m.curWidth()+delta), annotate.MinWidth), float64(annotate.MaxWidthOf(m.tool))))
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

// nextStamp returns the number of the new stamp: one more than already placed. After the
// last stamp is undone, its number is freed.
func (m *monitor) nextStamp() int {
	n := 1
	for _, s := range m.shapes {
		if s.Tool == annotate.ToolStamp {
			n++
		}
	}
	return n
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

// export renders the selection with annotations, saves it to a file if needed and copies
// it to the clipboard, then exits (or stays in the editor if "don't close after copying"
// is enabled in settings).
func (m *monitor) export(toClipboard, toFile bool) {
	if m.editing {
		m.commitText()
	}
	if !m.hasSel {
		return
	}
	img, err := render(m.frame.Img, m.sel, m.shapes, m.shaper)
	if err != nil {
		fail(i18n.T("render"), err)
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
			fail(i18n.T("saving"), err)
			return
		}
		msg, done = path, i18n.T("Saved: ")+filepath.Base(path)
	}
	clipboardFailed := false
	if toClipboard {
		if err := copyPNG(data); err != nil {
			fail(i18n.T("clipboard"), err)
			// Notifications may be unavailable too (no notify-send), and a window that closes
			// would hide the failure: say it on the screen and keep the editor open.
			m.showError(err.Error())
			clipboardFailed = true
		} else if msg == "" {
			msg, done = i18n.T("copied to the clipboard"), i18n.T("Copied to the clipboard")
		} else {
			msg += i18n.T(" (and to the clipboard)")
			done += i18n.T(", copied")
		}
	}
	if clipboardFailed {
		return
	}
	m.inform(i18n.T("Screenshot ready"), msg)
	if m.store.Get().KeepOpen {
		m.showMessage(done) // we stay in the editor: confirm the action on screen
		return
	}
	m.quit()
}

// quit remembers the region (if there is one) and exits. The region is remembered on any
// close, not only after a successful screenshot.
func (m *monitor) quit() {
	m.rememberRegion()
	m.closeWindows()
}

// closeWindows closes the windows gracefully and exits once they have closed: if the
// process exits while fullscreen windows are open, GNOME freezes for a second or two,
// which is noticeable in games. If the windows have not closed within 800 ms, we exit
// forcibly.
func (m *monitor) closeWindows() {
	closed := 0
	for _, p := range m.peers {
		if w := p.win.Load(); w != nil {
			w.Perform(system.ActionClose)
			closed++
		}
	}
	if closed == 0 {
		m.finish() // no windows (for example, in tests): nothing to close
		return
	}
	time.AfterFunc(800*time.Millisecond, m.finish)
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	notify(i18n.T("Error: ")+what, err.Error())
}

func rectOf(a, b f32.Point) image.Rectangle {
	// The corner that is further right and down is rounded up: the pixel under the cursor
	// belongs to the area, and a drag to the last pixel of the screen reaches the edge of the
	// frame instead of stopping one pixel short.
	return image.Rect(
		int(math.Floor(math.Min(float64(a.X), float64(b.X)))), int(math.Floor(math.Min(float64(a.Y), float64(b.Y)))),
		int(math.Ceil(math.Max(float64(a.X), float64(b.X)))), int(math.Ceil(math.Max(float64(a.Y), float64(b.Y)))),
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

// alone reports that this is the only overlay or the first one: the hint is drawn once, not
// on every monitor.
func (m *monitor) alone() bool { return len(m.peers) <= 1 || m.peers[0] == m }
