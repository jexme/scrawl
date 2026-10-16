package editor

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/annotate"
)

const restoreText = "Восстановить прошлую область  ·  R"

// dock — плавающая панель внизу по центру: подборка («было/стало») и возврат
// прошлой области. Не перекрывает выделение и панель инструментов: если место
// занято, переезжает наверх.
type dock struct {
	add, glue, separate, clear widget.Clickable
	restore                    widget.Clickable
}

// applyLast возвращает область предыдущего скриншота.
func (m *monitor) applyLast() {
	if !m.hasLast || m.hasSel {
		return
	}
	m.claim()
	m.sel, m.hasSel = m.last, true
	m.tool = annotate.ToolSelect
	m.useDefaultTool()
}

// label рендерит текст и возвращает макрос с его размером.
func (m *monitor) label(gtx layout.Context, txt string, sp float32, col color.NRGBA) (op.CallOp, image.Point) {
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := rec.Stop()
	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(1<<12, 1<<12)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{Alignment: text.Start, MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, unit.Sp(sp), txt, mat)
	return rec.Stop(), dims.Size
}

// layoutDock выбирает, что показать: подборку или возврат прошлой области.
func (m *monitor) layoutDock(gtx layout.Context, win image.Point) {
	if m.dragging || m.editing {
		return
	}
	switch {
	case m.hasSel || m.queued > 0:
		m.layoutCollectDock(gtx, win)
	case m.hasLast:
		m.layoutRestoreDock(gtx, win)
	}
}

// placeDock ставит панель размера w×h по центру: внизу, а если там выделение
// или панель инструментов, то наверху.
func (m *monitor) placeDock(gtx layout.Context, win image.Point, w, h int) image.Point {
	x := (win.X - w) / 2
	bottom := image.Pt(x, win.Y-h-gtx.Dp(40))
	top := image.Pt(x, gtx.Dp(28))
	if !m.hasSel {
		return bottom
	}
	margin := gtx.Dp(10)
	free := func(p image.Point) bool {
		r := image.Rectangle{Min: p, Max: p.Add(image.Pt(w, h))}.Inset(-margin)
		return !r.Overlaps(m.selWin) && !r.Overlaps(m.barRect)
	}
	switch {
	case free(bottom):
		return bottom
	case free(top):
		return top
	}
	return bottom
}

// layoutRestoreDock рисует кнопку «Восстановить прошлую область».
func (m *monitor) layoutRestoreDock(gtx layout.Context, win image.Point) {
	if m.dock.restore.Clicked(gtx) {
		m.applyLast()
		return
	}
	txt, size := m.label(gtx, restoreText, 15, iconColor)
	icon := gtx.Dp(20)
	padX, padY, gap := gtx.Dp(14), gtx.Dp(9), gtx.Dp(8)
	w := padX + icon + gap + size.X + padX
	h := max(size.Y, icon) + 2*padY

	defer op.Offset(m.placeDock(gtx, win, w, h)).Push(gtx.Ops).Pop()
	cgtx := gtx
	cgtx.Constraints = layout.Exact(image.Pt(w, h))
	m.dock.restore.Layout(cgtx, func(g layout.Context) layout.Dimensions {
		pill := clip.UniformRRect(image.Rect(0, 0, w, h), h/2)
		paint.FillShape(g.Ops, barBG, pill.Op(g.Ops))
		if m.dock.restore.Hovered() {
			paint.FillShape(g.Ops, accent, clip.Stroke{Path: pill.Path(g.Ops), Width: 2}.Op())
		}
		// Иконка рисуется на iconShare от квадрата: берём квадрат больше и сдвигаем.
		big := int(float32(icon) / iconShare)
		shift := (big - icon) / 2
		st := op.Offset(image.Pt(padX-shift, (h-icon)/2-shift)).Push(g.Ops)
		iconUndo(g.Ops, float32(big), iconColor)
		st.Pop()
		st = op.Offset(image.Pt(padX+icon+gap, (h-size.Y)/2)).Push(g.Ops)
		txt.Add(g.Ops)
		st.Pop()
		area := clip.Rect(image.Rect(0, 0, w, h)).Push(g.Ops)
		pointer.CursorPointer.Add(g.Ops)
		area.Pop()
		return layout.Dimensions{Size: image.Pt(w, h)}
	})
}

type dockItem struct {
	btn     *widget.Clickable
	txt     string
	primary bool
	do      func()
}

// collectItems собирает доступные действия подборки.
func (m *monitor) collectItems() []dockItem {
	var items []dockItem
	total := m.queued + boolInt(m.hasSel)
	if m.hasSel {
		txt := "В подборку  ·  A"
		if m.queued > 0 {
			txt = "Добавить ещё  ·  A"
		}
		items = append(items, dockItem{&m.dock.add, txt, m.queued == 0, m.addToCollection})
	}
	if total >= 2 {
		txt := "Склеить  ·  G"
		if m.hasSel {
			txt = "Склеить с этой областью  ·  G"
		}
		items = append(items, dockItem{&m.dock.glue, txt, m.queued > 0, m.glueCollection})
	}
	if m.queued > 0 {
		items = append(items,
			dockItem{&m.dock.separate, "Отдельно  ·  D", false, m.sendSeparately},
			dockItem{&m.dock.clear, "Очистить", false, m.clearCollection})
	}
	return items
}

// layoutCollectDock рисует панель подборки: счётчик и действия.
func (m *monitor) layoutCollectDock(gtx layout.Context, win image.Point) {
	items := m.collectItems()
	for _, it := range items {
		if it.btn.Clicked(gtx) {
			it.do()
			return
		}
	}

	title := ""
	switch {
	case m.queued > 0 && !m.hasSel:
		title = fmt.Sprintf("Подборка: %d  ·  выделите область", m.queued)
	case m.queued > 0:
		title = fmt.Sprintf("Подборка: %d", m.queued)
	}

	type chip struct {
		macro op.CallOp
		size  image.Point
		w     int
	}
	padX, padY, gap := gtx.Dp(12), gtx.Dp(6), gtx.Dp(8)
	var (
		titleMacro op.CallOp
		titleSize  image.Point
	)
	if title != "" {
		titleMacro, titleSize = m.label(gtx, title, 15, iconColor)
	}
	chips := make([]chip, len(items))
	chipH := max(titleSize.Y, 0) + 2*padY
	total := padX
	if title != "" {
		total += titleSize.X + gap
	}
	for i, it := range items {
		macro, size := m.label(gtx, it.txt, 14, iconColor)
		chips[i] = chip{macro, size, size.X + 2*padX}
		chipH = max(chipH, size.Y+2*padY)
		total += chips[i].w + gap
	}
	total += padX - gap
	h := chipH + 2*gtx.Dp(6)
	if len(items) == 0 && title == "" {
		return
	}

	defer op.Offset(m.placeDock(gtx, win, total, h)).Push(gtx.Ops).Pop()
	pill := clip.UniformRRect(image.Rect(0, 0, total, h), h/2)
	paint.FillShape(gtx.Ops, barBG, pill.Op(gtx.Ops))

	cx := padX
	if title != "" {
		st := op.Offset(image.Pt(cx, (h-titleSize.Y)/2)).Push(gtx.Ops)
		titleMacro.Add(gtx.Ops)
		st.Pop()
		cx += titleSize.X + gap
	}
	for i, it := range items {
		c := chips[i]
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(c.w, chipH))
		st := op.Offset(image.Pt(cx, (h-chipH)/2)).Push(gtx.Ops)
		it.btn.Layout(cgtx, func(g layout.Context) layout.Dimensions {
			r := clip.UniformRRect(image.Rect(0, 0, c.w, chipH), chipH/2)
			bg := color.NRGBA{R: 255, G: 255, B: 255, A: 22}
			if it.primary {
				bg = accent
			}
			paint.FillShape(g.Ops, bg, r.Op(g.Ops))
			if it.btn.Hovered() {
				paint.FillShape(g.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 60}, clip.Stroke{Path: r.Path(g.Ops), Width: 2}.Op())
			}
			lt := op.Offset(image.Pt(padX, (chipH-c.size.Y)/2)).Push(g.Ops)
			c.macro.Add(g.Ops)
			lt.Pop()
			area := clip.Rect(image.Rect(0, 0, c.w, chipH)).Push(g.Ops)
			pointer.CursorPointer.Add(g.Ops)
			area.Pop()
			return layout.Dimensions{Size: image.Pt(c.w, chipH)}
		})
		st.Pop()
		cx += c.w + gap
	}
}
