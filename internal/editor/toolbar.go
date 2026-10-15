package editor

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/jexme/screen-editor/internal/annotate"
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
	tools  []*toolButton
	colors []widget.Clickable
	undo   actionButton
	copy   actionButton
	save   actionButton
	close  actionButton
	area   int // тег области ввода, не пропускающей клики на холст
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
		colors: make([]widget.Clickable, len(annotate.Palette)),
		undo:   actionButton{icon: iconUndo},
		copy:   actionButton{icon: iconCopy},
		save:   actionButton{icon: iconSave},
		close:  actionButton{icon: iconClose},
	}
}

// width возвращает ширину панели в пикселях окна.
func (t *toolbar) width(btn, gap, sep, pad int) int {
	n := len(t.tools) + len(t.colors) + 4
	return 2*pad + n*btn + (n-3)*gap + 2*sep
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
			m.color = annotate.Palette[i]
		}
		c := annotate.Palette[i]
		active := m.color == c
		place(func(g layout.Context) layout.Dimensions {
			return t.colors[i].Layout(g, func(g layout.Context) layout.Dimensions {
				drawSwatch(g, c, active)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
	}
	cursor += sep - gap

	actions := []struct {
		a  *actionButton
		do func()
	}{
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
	s := gtx.Constraints.Min.X
	if active {
		rr := clip.UniformRRect(image.Rect(0, 0, s, s), gtx.Dp(6))
		paint.FillShape(gtx.Ops, accent, rr.Op(gtx.Ops))
	}
	icon(gtx.Ops, float32(s), iconColor)
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

// line рисует ломаную в нормированных координатах иконки (0..1).
func line(ops *op.Ops, s float32, c color.NRGBA, w float32, pts ...[2]float32) {
	var p clip.Path
	p.Begin(ops)
	for i, pt := range pts {
		fp := f32.Pt(pt[0]*s, pt[1]*s)
		if i == 0 {
			p.MoveTo(fp)
		} else {
			p.LineTo(fp)
		}
	}
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: w}.Op())
}

func iconW(s float32) float32 { return s * 0.07 }

func iconSelect(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s)
	line(ops, s, c, w, [2]float32{0.2, 0.4}, [2]float32{0.2, 0.2}, [2]float32{0.4, 0.2})
	line(ops, s, c, w, [2]float32{0.6, 0.2}, [2]float32{0.8, 0.2}, [2]float32{0.8, 0.4})
	line(ops, s, c, w, [2]float32{0.8, 0.6}, [2]float32{0.8, 0.8}, [2]float32{0.6, 0.8})
	line(ops, s, c, w, [2]float32{0.4, 0.8}, [2]float32{0.2, 0.8}, [2]float32{0.2, 0.6})
}

func iconPen(ops *op.Ops, s float32, c color.NRGBA) {
	line(ops, s, c, iconW(s), [2]float32{0.2, 0.7}, [2]float32{0.35, 0.35}, [2]float32{0.5, 0.65},
		[2]float32{0.65, 0.3}, [2]float32{0.8, 0.5})
}

func iconLine(ops *op.Ops, s float32, c color.NRGBA) {
	line(ops, s, c, iconW(s), [2]float32{0.2, 0.8}, [2]float32{0.8, 0.2})
}

func iconArrow(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s)
	line(ops, s, c, w, [2]float32{0.2, 0.8}, [2]float32{0.75, 0.25})
	line(ops, s, c, w, [2]float32{0.45, 0.2}, [2]float32{0.8, 0.2}, [2]float32{0.8, 0.55})
}

func iconRect(ops *op.Ops, s float32, c color.NRGBA) {
	line(ops, s, c, iconW(s), [2]float32{0.2, 0.28}, [2]float32{0.8, 0.28}, [2]float32{0.8, 0.72},
		[2]float32{0.2, 0.72}, [2]float32{0.2, 0.28})
}

func iconMarker(ops *op.Ops, s float32, c color.NRGBA) {
	hl := c
	hl.A = 130
	line(ops, s, hl, s*0.26, [2]float32{0.28, 0.68}, [2]float32{0.72, 0.32})
	line(ops, s, c, iconW(s), [2]float32{0.2, 0.84}, [2]float32{0.5, 0.84})
}

// iconBlur: рамка с полосами, которые по мере движения вправо «растворяются».
func iconBlur(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s)
	line(ops, s, c, w, [2]float32{0.2, 0.25}, [2]float32{0.8, 0.25}, [2]float32{0.8, 0.75},
		[2]float32{0.2, 0.75}, [2]float32{0.2, 0.25})
	for i, y := range []float32{0.4, 0.5, 0.6} {
		fade := c
		fade.A = uint8(255 - 70*i)
		line(ops, s, fade, w*0.9, [2]float32{0.3, y}, [2]float32{0.7, y})
	}
}

func iconText(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s) * 1.2
	line(ops, s, c, w, [2]float32{0.25, 0.25}, [2]float32{0.75, 0.25})
	line(ops, s, c, w, [2]float32{0.5, 0.25}, [2]float32{0.5, 0.8})
}

func iconUndo(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s)
	line(ops, s, c, w, [2]float32{0.32, 0.36}, [2]float32{0.62, 0.36}, [2]float32{0.78, 0.5},
		[2]float32{0.62, 0.68}, [2]float32{0.34, 0.68})
	line(ops, s, c, w, [2]float32{0.44, 0.22}, [2]float32{0.3, 0.36}, [2]float32{0.44, 0.5})
}

func iconCopy(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s)
	line(ops, s, c, w, [2]float32{0.38, 0.2}, [2]float32{0.8, 0.2}, [2]float32{0.8, 0.62},
		[2]float32{0.38, 0.62}, [2]float32{0.38, 0.2})
	line(ops, s, c, w, [2]float32{0.2, 0.38}, [2]float32{0.62, 0.38}, [2]float32{0.62, 0.8},
		[2]float32{0.2, 0.8}, [2]float32{0.2, 0.38})
}

func iconSave(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s)
	line(ops, s, c, w, [2]float32{0.5, 0.2}, [2]float32{0.5, 0.6})
	line(ops, s, c, w, [2]float32{0.32, 0.45}, [2]float32{0.5, 0.63}, [2]float32{0.68, 0.45})
	line(ops, s, c, w, [2]float32{0.2, 0.64}, [2]float32{0.2, 0.8}, [2]float32{0.8, 0.8}, [2]float32{0.8, 0.64})
}

func iconClose(ops *op.Ops, s float32, c color.NRGBA) {
	w := iconW(s)
	line(ops, s, c, w, [2]float32{0.26, 0.26}, [2]float32{0.74, 0.74})
	line(ops, s, c, w, [2]float32{0.74, 0.26}, [2]float32{0.26, 0.74})
}
