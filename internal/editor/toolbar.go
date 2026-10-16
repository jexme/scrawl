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
	icon iconFunc
	btn  widget.Clickable
}

// toolbar хранит кнопки панели инструментов. Состояние выбранного инструмента
// и цвета лежит в monitor.
type toolbar struct {
	tools    []*toolButton
	colors   []widget.Clickable
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
		},
		colors:   make([]widget.Clickable, len(annotate.Palette)),
		wMinus:   actionButton{icon: iconMinus},
		wPlus:    actionButton{icon: iconPlus},
		settings: actionButton{icon: iconSettings},
		add:      actionButton{icon: iconAdd},
		undo:     actionButton{icon: iconUndo},
		copy:     actionButton{icon: iconCopy},
		save:     actionButton{icon: iconSave},
		close:    actionButton{icon: iconClose},
	}
}

// width возвращает ширину панели в пикселях окна.
func (t *toolbar) width(btn, gap, sep, pad int) int {
	// Кнопки: инструменты, цвета, «−», значение, «+», шесть действий. Три разделителя.
	n := len(t.tools) + len(t.colors) + 3 + 6
	return 2*pad + n*btn + (n-4)*gap + 3*sep
}

// layoutToolbar рисует панель рядом с выделением sel (в пикселях окна) и
// применяет нажатия кнопок к состоянию m.
func (m *monitor) layoutToolbar(gtx layout.Context, win image.Point, sel image.Rectangle) {
	t := m.bar
	btn := gtx.Dp(34)
	gap := gtx.Dp(4)
	sep := gtx.Dp(10)
	pad := gtx.Dp(6)
	margin := gtx.Dp(8)

	total := t.width(btn, gap, sep, pad)
	height := btn + 2*pad

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

	for _, tb := range t.tools {
		if tb.btn.Clicked(gtx) {
			m.setTool(tb.tool)
		}
		active := m.tool == tb.tool
		place(func(g layout.Context) layout.Dimensions {
			return tb.btn.Layout(g, func(g layout.Context) layout.Dimensions {
				drawButton(g, active, tb.icon)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
	}
	cursor += sep - gap

	for i := range t.colors {
		if t.colors[i].Clicked(gtx) {
			m.color = m.palette[i]
			m.colorPicked = true
		}
		c := m.palette[i]
		active := m.color == c
		place(func(g layout.Context) layout.Dimensions {
			return t.colors[i].Layout(g, func(g layout.Context) layout.Dimensions {
				drawSwatch(g, c, active)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
	}
	cursor += sep - gap

	// Толщина линии, размер текста, сила размытия: «−», текущее значение, «+».
	enabled := m.tool != annotate.ToolSelect
	ic := iconColor
	if !enabled {
		ic.A = 90
	}
	if t.wMinus.btn.Clicked(gtx) && enabled {
		m.bumpWidth(-1)
	}
	place(func(g layout.Context) layout.Dimensions {
		return t.wMinus.btn.Layout(g, func(g layout.Context) layout.Dimensions {
			drawButtonColored(g, false, t.wMinus.icon, ic)
			return layout.Dimensions{Size: g.Constraints.Min}
		})
	})
	place(func(g layout.Context) layout.Dimensions {
		m.drawWidthValue(g, enabled)
		return layout.Dimensions{Size: g.Constraints.Min}
	})
	if t.wPlus.btn.Clicked(gtx) && enabled {
		m.bumpWidth(1)
	}
	place(func(g layout.Context) layout.Dimensions {
		return t.wPlus.btn.Layout(g, func(g layout.Context) layout.Dimensions {
			drawButtonColored(g, false, t.wPlus.icon, ic)
			return layout.Dimensions{Size: g.Constraints.Min}
		})
	})
	cursor += sep - gap

	actions := []struct {
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
	for _, act := range actions {
		if act.a.btn.Clicked(gtx) {
			act.do()
		}
		place(func(g layout.Context) layout.Dimensions {
			return act.a.btn.Layout(g, func(g layout.Context) layout.Dimensions {
				drawButton(g, false, act.a.icon)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
	}

	// Поглощаем события панели, чтобы они не уходили на холст.
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &t.area, Kinds: pointer.Press | pointer.Release | pointer.Drag}); !ok {
			break
		}
	}
}

func drawButton(gtx layout.Context, active bool, icon iconFunc) {
	drawButtonColored(gtx, active, icon, iconColor)
}

func drawButtonColored(gtx layout.Context, active bool, icon iconFunc, c color.NRGBA) {
	s := gtx.Constraints.Min.X
	if active {
		rr := clip.UniformRRect(image.Rect(0, 0, s, s), gtx.Dp(6))
		paint.FillShape(gtx.Ops, accent, rr.Op(gtx.Ops))
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
