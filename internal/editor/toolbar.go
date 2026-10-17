package editor

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/settings"
)

var (
	barBG     = color.NRGBA{R: 32, G: 33, B: 36, A: 238}
	accent    = color.NRGBA{R: 66, G: 133, B: 244, A: 255}
	iconColor = color.NRGBA{R: 235, G: 235, B: 235, A: 255}
)

// iconFunc рисует иконку в квадрате со стороной s с началом в (0, 0).
type iconFunc func(ops *op.Ops, s float32, c color.NRGBA)

type toolButton struct {
	tool annotate.Tool
	icon iconFunc
	btn  widget.Clickable
}

type actionButton struct {
	icon   iconFunc
	btn    widget.Clickable
	action string // действие с клавишей из настроек: её буква пишется под кнопкой; пусто: клавиши нет
}

// toolbar хранит кнопки панели инструментов. Состояние выбранного инструмента
// и цвета лежит в monitor.
type toolbar struct {
	tools    []*toolButton
	colors   []widget.Clickable
	custom   widget.Clickable // свой цвет: открывает окошко выбора
	wMinus   actionButton
	wPlus    actionButton
	settings actionButton
	add      actionButton
	undo     actionButton
	copy     actionButton
	save     actionButton
	close    actionButton
	area     int // тег области ввода, не пропускающей клики на холст
}

func newToolbar() *toolbar {
	return &toolbar{
		tools: []*toolButton{
			{tool: annotate.ToolSelect, icon: iconSelect},
			{tool: annotate.ToolPen, icon: iconPen},
			{tool: annotate.ToolLine, icon: iconLine},
			{tool: annotate.ToolArrow, icon: iconArrow},
			{tool: annotate.ToolRect, icon: iconRect},
			{tool: annotate.ToolMarker, icon: iconMarker},
			{tool: annotate.ToolBlur, icon: iconBlur},
			{tool: annotate.ToolText, icon: iconText},
			{tool: annotate.ToolStamp, icon: iconStamp},
		},
		colors:   make([]widget.Clickable, len(annotate.Palette)),
		wMinus:   actionButton{icon: iconMinus},
		wPlus:    actionButton{icon: iconPlus},
		settings: actionButton{icon: iconSettings},
		add:      actionButton{icon: iconAdd, action: settings.ActionAdd},
		undo:     actionButton{icon: iconUndo, action: settings.ActionUndo},
		copy:     actionButton{icon: iconCopy, action: settings.ActionCopy},
		save:     actionButton{icon: iconSave, action: settings.ActionSave},
		close:    actionButton{icon: iconClose, action: settings.ActionCancel},
	}
}

// width возвращает ширину панели в пикселях окна.
func (t *toolbar) width(btn, gap, sep, pad int) int {
	// Кнопки: инструменты, цвета и свой цвет, «−», значение, «+», шесть действий. Три разделителя.
	n := len(t.tools) + len(t.colors) + 1 + 3 + 6
	return 2*pad + n*btn + (n-4)*gap + 3*sep
}

// layoutToolbar рисует панель рядом с выделением sel (в пикселях окна) и
// применяет нажатия кнопок к состоянию m.
// toolbarActions — кнопки действий справа на панели и что они делают.
func (m *monitor) toolbarActions() []struct {
	a  *actionButton
	do func()
} {
	t := m.bar
	return []struct {
		a  *actionButton
		do func()
	}{
		{&t.settings, m.openSettingsWindow},
		{&t.add, m.addToCollection},
		{&t.undo, m.undo},
		{&t.copy, func() { m.export(true, false) }},
		{&t.save, func() { m.export(false, true) }},
		{&t.close, m.quit},
	}
}

// toolbarClicks применяет нажатия на кнопки панели. Вызывается до того, как кнопки нарисованы:
// иначе кнопки левее нажатой рисовались бы со старым состоянием (подсвечены оба инструмента), а
// нового кадра никто бы не запросил, пока не придёт следующее событие мыши.
func (m *monitor) toolbarClicks(gtx layout.Context) {
	t := m.bar
	var clicked bool
	for _, tb := range t.tools {
		if tb.btn.Clicked(gtx) {
			m.setTool(tb.tool)
			clicked = true
		}
	}
	for i := range t.colors {
		if t.colors[i].Clicked(gtx) {
			m.color = m.palette[i]
			m.colorPicked = true
			clicked = true
		}
	}
	// Свой цвет: нажатие выбирает его и открывает окошко с палитрой, второе нажатие закрывает.
	if t.custom.Clicked(gtx) {
		clicked = true
		if m.pick.open {
			m.pick.open = false
		} else {
			if !m.hasCustom {
				m.customColor, m.hasCustom = m.color, true
			}
			m.pick.open, m.pick.hsv = true, hsvOf(m.customColor)
			m.color, m.colorPicked = m.customColor, true
		}
	}
	// Толщина: у «выделения» её нет.
	if t.wMinus.btn.Clicked(gtx) {
		m.bumpWidth(-1)
		clicked = true
	}
	if t.wPlus.btn.Clicked(gtx) {
		m.bumpWidth(1)
		clicked = true
	}
	for _, act := range m.toolbarActions() {
		if act.a.btn.Clicked(gtx) {
			act.do()
			clicked = true
		}
	}
	if clicked {
		gtx.Execute(op.InvalidateCmd{}) // кадр с новым состоянием не ждёт следующего движения мыши
	}
}

func (m *monitor) layoutToolbar(gtx layout.Context, win image.Point, sel image.Rectangle) {
	t := m.bar
	m.toolbarClicks(gtx)
	btn := gtx.Dp(34)
	gap := gtx.Dp(4)
	sep := gtx.Dp(10)
	pad := gtx.Dp(6)
	margin := gtx.Dp(8)

	total := t.width(btn, gap, sep, pad)
	capH := gtx.Dp(13) // строка под кнопками: клавиша быстрого вызова
	height := btn + capH + 2*pad

	x := sel.Max.X - total
	if x < margin {
		x = margin
	}
	if x+total > win.X-margin {
		x = win.X - margin - total
	}
	y := sel.Max.Y + margin
	if y+height > win.Y-margin {
		y = sel.Min.Y - margin - height
		if y < margin {
			y = sel.Max.Y - height - margin
		}
	}

	m.barRect = image.Rect(x, y, x+total, y+height)
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()

	// Фон и область ввода, чтобы клики по панели не попадали на холст.
	bg := image.Rect(0, 0, total, height)
	rr := clip.UniformRRect(bg, gtx.Dp(8))
	paint.FillShape(gtx.Ops, barBG, rr.Op(gtx.Ops))
	area := clip.Rect(bg).Push(gtx.Ops)
	event.Op(gtx.Ops, &t.area)
	area.Pop()

	cursor := pad
	place := func(w func(gtx layout.Context) layout.Dimensions) {
		st := op.Offset(image.Pt(cursor, pad)).Push(gtx.Ops)
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(btn, btn))
		w(cgtx)
		st.Pop()
		cursor += btn + gap
	}

	for i, tb := range t.tools {
		active := m.tool == tb.tool
		x0 := cursor
		place(func(g layout.Context) layout.Dimensions {
			return tb.btn.Layout(g, func(g layout.Context) layout.Dimensions {
				drawButton(g, active, tb.btn.Hovered(), tb.icon)
				handCursor(g)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
		m.drawKeyCaption(gtx, image.Pt(x0+btn/2, pad+btn), keyCaption(m.store.Get().Keys.Of(settings.ToolAction(settings.ToolNames[i]))))
	}
	cursor += sep - gap

	for i := range t.colors {
		c := m.palette[i]
		active := m.color == c
		place(func(g layout.Context) layout.Dimensions {
			return t.colors[i].Layout(g, func(g layout.Context) layout.Dimensions {
				drawSwatch(g, c, active)
				handCursor(g)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
	}

	customActive := m.hasCustom && m.color == m.customColor
	m.pickAnchor = x + cursor + btn/2
	place(func(g layout.Context) layout.Dimensions {
		return t.custom.Layout(g, func(g layout.Context) layout.Dimensions {
			drawCustomSwatch(g, m.customColor, m.hasCustom, customActive || m.pick.open)
			handCursor(g)
			return layout.Dimensions{Size: g.Constraints.Min}
		})
	})
	cursor += sep - gap

	// Толщина линии, размер текста, сила размытия: «−», текущее значение, «+».
	enabled := m.tool != annotate.ToolSelect
	ic := iconColor
	if !enabled {
		ic.A = 90
	}
	capKey := func(action string) string { return keyCaption(m.store.Get().Keys.Of(action)) }
	x0 := cursor
	capA := uint8(200)
	if !enabled {
		capA = 90
	}
	m.drawKeyCaptionA(gtx, image.Pt(x0+btn/2, pad+btn), capKey(settings.ActionWidthDown), capA)
	place(func(g layout.Context) layout.Dimensions {
		return t.wMinus.btn.Layout(g, func(g layout.Context) layout.Dimensions {
			drawButtonColored(g, false, enabled && t.wMinus.btn.Hovered(), t.wMinus.icon, ic)
			if enabled {
				handCursor(g)
			}
			return layout.Dimensions{Size: g.Constraints.Min}
		})
	})
	place(func(g layout.Context) layout.Dimensions {
		m.drawWidthValue(g, enabled)
		return layout.Dimensions{Size: g.Constraints.Min}
	})
	m.drawKeyCaptionA(gtx, image.Pt(cursor+btn/2, pad+btn), capKey(settings.ActionWidthUp), capA)
	place(func(g layout.Context) layout.Dimensions {
		return t.wPlus.btn.Layout(g, func(g layout.Context) layout.Dimensions {
			drawButtonColored(g, false, enabled && t.wPlus.btn.Hovered(), t.wPlus.icon, ic)
			if enabled {
				handCursor(g)
			}
			return layout.Dimensions{Size: g.Constraints.Min}
		})
	})
	cursor += sep - gap

	for _, act := range m.toolbarActions() {
		x0 := cursor
		place(func(g layout.Context) layout.Dimensions {
			return act.a.btn.Layout(g, func(g layout.Context) layout.Dimensions {
				drawButton(g, false, act.a.btn.Hovered(), act.a.icon)
				handCursor(g)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
		if act.a.action != "" {
			m.drawKeyCaption(gtx, image.Pt(x0+btn/2, pad+btn), keyCaption(m.store.Get().Keys.Of(act.a.action)))
		}
	}

	// Поглощаем события панели, чтобы они не уходили на холст.
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &t.area, Kinds: pointer.Press | pointer.Release | pointer.Drag}); !ok {
			break
		}
	}
}

// handCursor делает курсор «рукой» над текущей кнопкой: вызывается в содержимом кнопки.
func handCursor(gtx layout.Context) {
	area := clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	area.Pop()
}

// drawKeyCaption пишет клавишу быстрого вызова мелко под кнопкой: center это
// середина кнопки по горизонтали и нижний край кнопки.
func (m *monitor) drawKeyCaption(gtx layout.Context, center image.Point, txt string) {
	m.drawKeyCaptionA(gtx, center, txt, 200)
}

// drawKeyCaptionA то же с заданной яркостью: у недоступной кнопки подпись тусклее.
func (m *monitor) drawKeyCaptionA(gtx layout.Context, center image.Point, txt string, alpha uint8) {
	if txt == "" {
		return
	}
	col := iconColor
	col.A = alpha
	label, size := m.label(gtx, txt, 10, col)
	defer op.Offset(image.Pt(center.X-size.X/2, center.Y+gtx.Dp(1))).Push(gtx.Ops).Pop()
	label.Add(gtx.Ops)
}

func drawButton(gtx layout.Context, active, hovered bool, icon iconFunc) {
	drawButtonColored(gtx, active, hovered, icon, iconColor)
}

// drawButtonColored рисует кнопку: нажатая (активный инструмент) закрашена акцентом, под мышью
// слегка подсвечена, чтобы было видно, что на неё можно нажать.
func drawButtonColored(gtx layout.Context, active, hovered bool, icon iconFunc, c color.NRGBA) {
	s := gtx.Constraints.Min.X
	switch {
	case active:
		rr := clip.UniformRRect(image.Rect(0, 0, s, s), gtx.Dp(6))
		paint.FillShape(gtx.Ops, accent, rr.Op(gtx.Ops))
	case hovered:
		rr := clip.UniformRRect(image.Rect(0, 0, s, s), gtx.Dp(6))
		paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 30}, rr.Op(gtx.Ops))
	}
	icon(gtx.Ops, float32(s), c)
}

// drawWidthValue показывает число: толщину текущего инструмента (для размытия силу).
func (m *monitor) drawWidthValue(gtx layout.Context, enabled bool) {
	s := gtx.Constraints.Min.X
	txt := "–"
	col := iconColor
	if enabled {
		txt = strconv.Itoa(int(m.curWidth()))
	} else {
		col.A = 90
	}

	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := rec.Stop()
	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(s, s)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{Alignment: text.Middle, MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, 15, txt, mat)
	label := rec.Stop()

	st := op.Offset(image.Pt(0, (s-dims.Size.Y)/2)).Push(gtx.Ops)
	label.Add(gtx.Ops)
	st.Pop()
}

// drawCustomSwatch рисует кнопку своего цвета: радужное кольцо, а в середине выбранный цвет
// (пока он не выбран, тёмная точка).
func drawCustomSwatch(gtx layout.Context, c color.NRGBA, has, active bool) {
	s := gtx.Constraints.Min.X
	inset := int(float32(s) * 0.22)
	if active {
		ring := clip.Ellipse(image.Rect(inset-3, inset-3, s-inset+3, s-inset+3))
		paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, ring.Op(gtx.Ops))
	}
	outer := image.Rect(inset, inset, s-inset, s-inset)
	func() {
		defer clip.Ellipse(outer).Push(gtx.Ops).Pop()
		seg := float32(outer.Dx()) / 6
		for i := 0; i < 6; i++ {
			x0 := float32(outer.Min.X) + float32(i)*seg
			part := clip.Rect(image.Rect(int(x0), outer.Min.Y, int(x0+seg)+1, outer.Max.Y)).Push(gtx.Ops)
			paint.LinearGradientOp{Stop1: f32.Pt(x0, 0), Color1: hueStops[i], Stop2: f32.Pt(x0+seg, 0), Color2: hueStops[i+1]}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			part.Pop()
		}
	}()
	dot := int(float32(outer.Dx()) * 0.28)
	fill := barBG
	if has {
		fill = c
	}
	paint.FillShape(gtx.Ops, fill, clip.Ellipse(image.Rect(outer.Min.X+dot, outer.Min.Y+dot, outer.Max.X-dot, outer.Max.Y-dot)).Op(gtx.Ops))
}

func drawSwatch(gtx layout.Context, c color.NRGBA, active bool) {
	s := gtx.Constraints.Min.X
	inset := int(float32(s) * 0.22)
	if active {
		ring := clip.Ellipse(image.Rect(inset-3, inset-3, s-inset+3, s-inset+3))
		paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, ring.Op(gtx.Ops))
	}
	dot := clip.Ellipse(image.Rect(inset, inset, s-inset, s-inset))
	paint.FillShape(gtx.Ops, c, dot.Op(gtx.Ops))
}

// Иконки в стиле Tabler Icons (MIT): сетка 24×24, линия 2, скруглённые концы.
// Рисуются на 60% размера кнопки по центру.
const iconShare = 0.6

func tabler(ops *op.Ops, s float32, c color.NRGBA, paths ...string) {
	size := s * iconShare
	off := f32.Pt((s-size)/2, (s-size)/2)
	scale := size / 24
	for _, d := range paths {
		strokeSVG(ops, c, d, off, scale, 2*scale)
	}
}

func iconSelect(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M8 5v10a1 1 0 0 0 1 1h10", "M5 8h10a1 1 0 0 1 1 1v10")
}

func iconPen(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M4 20h4l10.5 -10.5a2.828 2.828 0 1 0 -4 -4l-10.5 10.5v4", "M13.5 6.5l4 4")
}

func iconLine(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M4 20l16 -16")
}

func iconArrow(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M17 7l-10 10", "M8 7l9 0l0 9")
}

func iconRect(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M3 3m0 2a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v14a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2z")
}

func iconMarker(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M3 19h4l10.5 -10.5a1.5 1.5 0 0 0 -4 -4l-10.5 10.5v4", "M12.5 5.5l4 4",
		"M4.5 13.5l4 4", "M21 15v4h-8l4 -4z")
}

// blurDots — растр из точек: радиус тает слева направо и сверху вниз.
var blurDots = [][3]float32{
	{8, 8, 1.9}, {8, 12, 1.5}, {8, 16, 1.1},
	{12.5, 8, 1.5}, {12.5, 12, 1.1}, {12.5, 16, 0.75},
	{16.5, 8, 1.1}, {16.5, 12, 0.75}, {16.5, 16, 0.45},
}

func iconBlur(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M3 3m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v12a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z")
	size := s * iconShare
	off := f32.Pt((s-size)/2, (s-size)/2)
	scale := size / 24
	for _, d := range blurDots {
		cx, cy, r := d[0], d[1], d[2]
		circle := fmt.Sprintf("M%g %gm%g 0a%g %g 0 1 0 %g 0a%g %g 0 1 0 %g 0z", cx, cy, -r, r, r, 2*r, r, r, -2*r)
		fillSVG(ops, c, circle, off, scale)
	}
}

func iconMinus(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M5 12l14 0")
}

func iconPlus(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M12 5l0 14", "M5 12l14 0")
}

func iconText(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M4 20l3 0", "M14 20l7 0", "M6.8 15l7.4 0", "M5 20l5.5 -16", "M16 20l-5.5 -16")
}

// iconStamp: кружок с единицей (Tabler circle-number-1), штамп с номером.
func iconStamp(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0", "M10 10l2 -2v8")
}

func iconUndo(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M9 14l-4 -4l4 -4", "M5 10h11a4 4 0 1 1 0 8h-1")
}

func iconCopy(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c,
		"M7 7m0 2.667a2.667 2.667 0 0 1 2.667 -2.667h8.666a2.667 2.667 0 0 1 2.667 2.667v8.666a2.667 2.667 0 0 1 -2.667 2.667h-8.666a2.667 2.667 0 0 1 -2.667 -2.667z",
		"M4.012 16.737a2.005 2.005 0 0 1 -1.012 -1.737v-10c0 -1.1 .9 -2 2 -2h10c.75 0 1.158 .385 1.5 1")
}

func iconSave(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M6 4h10l4 4v10a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-12a2 2 0 0 1 2 -2",
		"M12 14m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0", "M14 4l0 4l-6 0l0 -4")
}

// iconAdd: квадрат с плюсом (Tabler square-plus), «добавить в подборку».
func iconAdd(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M3 3m0 2a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v14a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2z",
		"M9 12h6", "M12 9v6")
}

// iconSettings: шестерёнка (Tabler settings).
func iconSettings(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c,
		"M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z",
		"M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0")
}

func iconClose(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M18 6l-12 12", "M6 6l12 12")
}
