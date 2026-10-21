package editor

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/i18n"
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
	more     actionButton // меню «⋯»: настройки и «в подборку»
	// Теги областей ввода, не пропускающих клики на холст: основной ряд, ряд цветов и меню.
	area, area2, menuArea int
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
		more:     actionButton{icon: iconMore},
	}
}

// rowWidth считает ширину ряда из n кнопок с seps разделителями: между кнопками зазор gap,
// на месте разделителя sep.
func rowWidth(n, seps, btn, gap, sep, pad int) int {
	return 2*pad + n*btn + (n-1-seps)*gap + seps*sep
}

// mainWidth возвращает ширину основного ряда: инструменты | отмена, копировать, сохранить | «⋯», закрыть.
func (t *toolbar) mainWidth(btn, gap, sep, pad int) int {
	return rowWidth(len(t.tools)+3+2, 2, btn, gap, sep, pad)
}

// styleWidth возвращает ширину ряда цветов и толщины: цвета и свой цвет | «−», значение, «+».
func (t *toolbar) styleWidth(btn, gap, sep, pad int) int {
	return rowWidth(len(t.colors)+1+3, 1, btn, gap, sep, pad)
}

// rowPlacer раскладывает кнопки ряда слева направо в координатах ряда.
type rowPlacer struct {
	gtx                layout.Context
	cursor             int
	btn, gap, pad, sep int
}

func (p *rowPlacer) place(w func(gtx layout.Context) layout.Dimensions) {
	st := op.Offset(image.Pt(p.cursor, p.pad)).Push(p.gtx.Ops)
	cgtx := p.gtx
	cgtx.Constraints = layout.Exact(image.Pt(p.btn, p.btn))
	w(cgtx)
	st.Pop()
	p.cursor += p.btn + p.gap
}

// divider ставит разделитель вместо обычного зазора.
func (p *rowPlacer) divider() { p.cursor += p.sep - p.gap }

// center возвращает середину очередной кнопки по горизонтали (в координатах ряда).
func (p *rowPlacer) center() int { return p.cursor + p.btn/2 }

// toolbarActions — кнопки действий основного ряда и что они делают: отмена, копировать,
// сохранить и закрыть. Настройки и «в подборку» лежат в меню «⋯», см. menuItems.
func (m *monitor) toolbarActions() []struct {
	a  *actionButton
	do func()
} {
	t := m.bar
	return []struct {
		a  *actionButton
		do func()
	}{
		{&t.undo, m.undo},
		{&t.copy, func() { m.export(true, false) }},
		{&t.save, func() { m.export(false, true) }},
		{&t.close, m.quit},
	}
}

// menuItems — пункты меню «⋯».
func (m *monitor) menuItems() []struct {
	a     *actionButton
	label string
	do    func()
} {
	t := m.bar
	return []struct {
		a     *actionButton
		label string
		do    func()
	}{
		{&t.settings, i18n.T("Settings"), m.openSettingsWindow},
		{&t.add, m.withKey(i18n.T("To collection"), settings.ActionAdd), m.addToCollection},
	}
}

// toolbarClicks применяет нажатия на кнопки панели. Вызывается до того, как кнопки нарисованы:
// иначе кнопки левее нажатой рисовались бы со старым состоянием (подсвечены оба инструмента), а
// нового кадра никто бы не запросил, пока не придёт следующее событие мыши.
func (m *monitor) toolbarClicks(gtx layout.Context) {
	t := m.bar
	var clicked, toggledMenu bool
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
	if t.more.btn.Clicked(gtx) {
		m.menuOpen = !m.menuOpen
		clicked, toggledMenu = true, true
	}
	for _, it := range m.menuItems() {
		if it.a.btn.Clicked(gtx) {
			m.menuOpen = false
			it.do()
			clicked = true
		}
	}
	if clicked {
		if !toggledMenu {
			m.menuOpen = false // любое другое нажатие на панель закрывает меню
		}
		gtx.Execute(op.InvalidateCmd{}) // кадр с новым состоянием не ждёт следующего движения мыши
	}
}

// layoutToolbar рисует панель рядом с выделением sel (в пикселях окна) и применяет нажатия
// кнопок к состоянию m. Основной ряд (инструменты, отмена, копировать, сохранить, меню «⋯»,
// закрыть) виден всегда, ряд цветов и толщины под ним только при рисующем инструменте.
func (m *monitor) layoutToolbar(gtx layout.Context, win image.Point, sel image.Rectangle) {
	t := m.bar
	m.toolbarClicks(gtx)
	btn := gtx.Dp(34)
	gap := gtx.Dp(4)
	sep := gtx.Dp(10)
	pad := gtx.Dp(6)
	margin := gtx.Dp(8)
	capH := gtx.Dp(13)         // строка под кнопками: клавиша быстрого вызова
	rowH := btn + capH + 2*pad // основной ряд: под кнопками подписаны клавиши
	rowH2 := btn + 2*pad       // ряд цветов и толщины без подписей
	vGap := gtx.Dp(4)

	w1, w2 := t.mainWidth(btn, gap, sep, pad), t.styleWidth(btn, gap, sep, pad)
	showStyle := m.tool != annotate.ToolSelect
	if !showStyle {
		m.pick.open = false // у «выделения» нет цвета, окошко цвета не нужно
	}

	x1 := sel.Max.X - w1
	if x1 < margin {
		x1 = margin
	}
	if x1+w1 > win.X-margin {
		x1 = win.X - margin - w1
	}
	x2 := x1
	if x2+w2 > win.X-margin {
		x2 = max(win.X-margin-w2, margin)
	}

	// Место считается под оба ряда, чтобы панель не прыгала, когда второй ряд появляется.
	reserve := rowH + vGap + rowH2
	below := true
	y1 := sel.Max.Y + margin
	if y1+reserve > win.Y-margin {
		if sel.Min.Y-margin-reserve >= margin {
			below = false
			y1 = sel.Min.Y - margin - rowH // основной ряд у выделения, второй дальше от него
		} else {
			y1 = sel.Max.Y - reserve - margin
		}
	}
	y2 := y1 + rowH + vGap
	if !below {
		y2 = y1 - vGap - rowH2
	}
	r1 := image.Rect(x1, y1, x1+w1, y1+rowH)
	r2 := image.Rect(x2, y2, x2+w2, y2+rowH2)
	m.mainRow = r1
	m.barSpace = r1.Union(r2)
	m.barRect = r1
	if showStyle {
		m.barRect = r1.Union(r2)
	}
	m.barBelow = below
	capKey := func(action string) string { return keyCaption(m.store.Get().Keys.Of(action)) }

	// Основной ряд.
	func() {
		defer op.Offset(r1.Min).Push(gtx.Ops).Pop()
		bg := image.Rect(0, 0, w1, rowH)
		paint.FillShape(gtx.Ops, barBG, clip.UniformRRect(bg, gtx.Dp(8)).Op(gtx.Ops))
		area := clip.Rect(bg).Push(gtx.Ops)
		event.Op(gtx.Ops, &t.area)
		area.Pop()

		p := &rowPlacer{gtx: gtx, cursor: pad, btn: btn, gap: gap, pad: pad, sep: sep}
		for i, tb := range t.tools {
			x0 := p.center()
			p.place(func(g layout.Context) layout.Dimensions {
				return tb.btn.Layout(g, func(g layout.Context) layout.Dimensions {
					drawButton(g, m.tool == tb.tool, tb.btn.Hovered(), tb.icon)
					handCursor(g)
					return layout.Dimensions{Size: g.Constraints.Min}
				})
			})
			m.drawKeyCaption(gtx, image.Pt(x0, pad+btn), capKey(settings.ToolAction(settings.ToolNames[i])))
		}
		p.divider()

		acts := m.toolbarActions()
		drawAct := func(a *actionButton) {
			x0 := p.center()
			p.place(func(g layout.Context) layout.Dimensions {
				return a.btn.Layout(g, func(g layout.Context) layout.Dimensions {
					drawButton(g, false, a.btn.Hovered(), a.icon)
					handCursor(g)
					return layout.Dimensions{Size: g.Constraints.Min}
				})
			})
			if a.action != "" {
				m.drawKeyCaption(gtx, image.Pt(x0, pad+btn), capKey(a.action))
			}
		}
		for _, act := range acts[:3] {
			drawAct(act.a)
		}
		p.divider()
		m.menuAnchor = r1.Min.X + p.center()
		p.place(func(g layout.Context) layout.Dimensions {
			return t.more.btn.Layout(g, func(g layout.Context) layout.Dimensions {
				drawButton(g, m.menuOpen, t.more.btn.Hovered(), t.more.icon)
				handCursor(g)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
		})
		drawAct(acts[3].a)

		// Поглощаем события панели, чтобы они не уходили на холст.
		for {
			if _, ok := gtx.Event(pointer.Filter{Target: &t.area, Kinds: pointer.Press | pointer.Release | pointer.Drag}); !ok {
				break
			}
		}
	}()

	// Ряд цветов и толщины.
	if showStyle {
		func() {
			defer op.Offset(r2.Min).Push(gtx.Ops).Pop()
			bg := image.Rect(0, 0, w2, rowH2)
			paint.FillShape(gtx.Ops, barBG, clip.UniformRRect(bg, gtx.Dp(8)).Op(gtx.Ops))
			area := clip.Rect(bg).Push(gtx.Ops)
			event.Op(gtx.Ops, &t.area2)
			area.Pop()

			p := &rowPlacer{gtx: gtx, cursor: pad, btn: btn, gap: gap, pad: pad, sep: sep}
			for i := range t.colors {
				c := m.palette[i]
				active := m.color == c
				p.place(func(g layout.Context) layout.Dimensions {
					return t.colors[i].Layout(g, func(g layout.Context) layout.Dimensions {
						drawSwatch(g, c, active)
						handCursor(g)
						return layout.Dimensions{Size: g.Constraints.Min}
					})
				})
			}
			customActive := m.hasCustom && m.color == m.customColor
			m.pickAnchor = r2.Min.X + p.center()
			p.place(func(g layout.Context) layout.Dimensions {
				return t.custom.Layout(g, func(g layout.Context) layout.Dimensions {
					drawCustomSwatch(g, m.customColor, m.hasCustom, customActive || m.pick.open)
					handCursor(g)
					return layout.Dimensions{Size: g.Constraints.Min}
				})
			})
			p.divider()

			// Толщина линии, размер текста, сила размытия: «−», текущее значение, «+».
			ic := iconColor
			p.place(func(g layout.Context) layout.Dimensions {
				return t.wMinus.btn.Layout(g, func(g layout.Context) layout.Dimensions {
					drawButtonColored(g, false, t.wMinus.btn.Hovered(), t.wMinus.icon, ic)
					handCursor(g)
					return layout.Dimensions{Size: g.Constraints.Min}
				})
			})
			p.place(func(g layout.Context) layout.Dimensions {
				m.drawWidthValue(g, true)
				return layout.Dimensions{Size: g.Constraints.Min}
			})
			p.place(func(g layout.Context) layout.Dimensions {
				return t.wPlus.btn.Layout(g, func(g layout.Context) layout.Dimensions {
					drawButtonColored(g, false, t.wPlus.btn.Hovered(), t.wPlus.icon, ic)
					handCursor(g)
					return layout.Dimensions{Size: g.Constraints.Min}
				})
			})

			for {
				if _, ok := gtx.Event(pointer.Filter{Target: &t.area2, Kinds: pointer.Press | pointer.Release | pointer.Drag}); !ok {
					break
				}
			}
		}()
	}

	m.layoutOverflowMenu(gtx, win)
}

// layoutOverflowMenu рисует меню «⋯» под панелью (над ней, если панель над выделением): пункты
// «Настройки» и «В подборку» с клавишей.
func (m *monitor) layoutOverflowMenu(gtx layout.Context, win image.Point) {
	if !m.menuOpen {
		return
	}
	t := m.bar
	items := m.menuItems()
	padX, padY := gtx.Dp(12), gtx.Dp(6)
	itemH, icon, gap := gtx.Dp(38), gtx.Dp(20), gtx.Dp(10)
	margin := gtx.Dp(8)

	type lab struct {
		macro op.CallOp
		size  image.Point
	}
	labs := make([]lab, len(items))
	textW := 0
	for i, it := range items {
		macro, size := m.label(gtx, it.label, 14, iconColor)
		labs[i] = lab{macro, size}
		textW = max(textW, size.X)
	}
	w := 2*padX + icon + gap + textW
	h := 2*padY + itemH*len(items)
	x := clampInt(m.menuAnchor+gtx.Dp(17)-w, margin, max(win.X-w-margin, margin))
	y := m.barRect.Max.Y + margin
	if !m.barBelow {
		y = m.barRect.Min.Y - margin - h
	}
	y = clampInt(y, margin, max(win.Y-h-margin, margin))
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()

	bg := image.Rect(0, 0, w, h)
	paint.FillShape(gtx.Ops, barBG, clip.UniformRRect(bg, gtx.Dp(8)).Op(gtx.Ops))
	area := clip.Rect(bg).Push(gtx.Ops)
	event.Op(gtx.Ops, &t.menuArea)
	area.Pop()

	big := int(float32(icon) / iconShare) // иконка рисуется на iconShare от квадрата: берём квадрат больше
	shift := (big - icon) / 2
	for i, it := range items {
		st := op.Offset(image.Pt(0, padY+i*itemH)).Push(gtx.Ops)
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(w, itemH))
		it.a.btn.Layout(cgtx, func(g layout.Context) layout.Dimensions {
			if it.a.btn.Hovered() {
				row := clip.UniformRRect(image.Rect(gtx.Dp(4), 0, w-gtx.Dp(4), itemH), gtx.Dp(6))
				paint.FillShape(g.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 30}, row.Op(g.Ops))
			}
			ist := op.Offset(image.Pt(padX-shift, (itemH-icon)/2-shift)).Push(g.Ops)
			it.a.icon(g.Ops, float32(big), iconColor)
			ist.Pop()
			lt := op.Offset(image.Pt(padX+icon+gap, (itemH-labs[i].size.Y)/2)).Push(g.Ops)
			labs[i].macro.Add(g.Ops)
			lt.Pop()
			handCursor(g)
			return layout.Dimensions{Size: image.Pt(w, itemH)}
		})
		st.Pop()
	}
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &t.menuArea, Kinds: pointer.Press | pointer.Release | pointer.Drag}); !ok {
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
	if txt == "" {
		return
	}
	col := iconColor
	col.A = 200
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
	const size float64 = 15
	dims := widget.Label{MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, unit.Sp(size), txt, mat)
	label := rec.Stop()

	// По горизонтали по ширине числа, по вертикали по середине цифр: цифра стоит на базовой линии
	// и занимает около 0,72 размера шрифта, а не всю высоту строки.
	baseline := dims.Size.Y - dims.Baseline
	top := s/2 - baseline + int(math.Round(size*0.72/2))
	st := op.Offset(image.Pt((s-dims.Size.X)/2, top)).Push(gtx.Ops)
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

// iconMore: три точки (Tabler dots), меню «ещё».
func iconMore(ops *op.Ops, s float32, c color.NRGBA) {
	tabler(ops, s, c, "M5 12m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0", "M12 12m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0",
		"M19 12m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0")
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
