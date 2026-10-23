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
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/settings"
)

// withKey appends the action's key from the settings to the button label.
func (m *monitor) withKey(label, action string) string {
	return label + "  ·  " + m.store.Get().Keys.Of(action)
}

// dock is a floating panel at the bottom centre: the collection ("before/after") and
// restoring the previous area. It does not cover the selection or the toolbar: if the
// place is taken, it moves to the top.
type dock struct {
	add, glue, separate, clear widget.Clickable
	restore                    widget.Clickable
}

// applyLast restores the area of the previous screenshot.
func (m *monitor) applyLast() {
	if !m.hasLast || m.hasSel {
		return
	}
	m.claim()
	m.sel, m.hasSel = m.last, true
	m.tool = annotate.ToolSelect
	m.useDefaultTool()
}

// label renders text and returns a macro with its size.
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

// layoutDock chooses what to show: the collection or the restore-previous-area button.
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

// placeDock places a panel of size w×h at the centre: at the bottom, or at the top if the
// selection or the toolbar is there.
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
		return !r.Overlaps(m.selWin) && !r.Overlaps(m.barSpace)
	}
	switch {
	case free(bottom):
		return bottom
	case free(top):
		return top
	}
	// The whole screen is selected and the toolbar sits inside it: the panel goes just above the
	// toolbar, where it covers only the picture, instead of lying over the buttons.
	above := image.Pt(x, m.barSpace.Min.Y-h-margin)
	if r := (image.Rectangle{Min: above, Max: above.Add(image.Pt(w, h))}); above.Y >= margin && !r.Overlaps(m.barSpace) {
		return above
	}
	return bottom
}

// layoutRestoreDock draws the "Restore the previous area" button.
func (m *monitor) layoutRestoreDock(gtx layout.Context, win image.Point) {
	if m.dock.restore.Clicked(gtx) {
		m.applyLast()
		gtx.Execute(op.InvalidateCmd{}) // the panel is no longer needed: the next frame without it does not wait for mouse movement
		return
	}
	txt, size := m.label(gtx, m.withKey(i18n.T("Restore the previous area"), settings.ActionRestore), 15, iconColor)
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
		// The icon is drawn at iconShare of the square: take a bigger square and shift it.
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

// collectItems gathers the available collection actions.
func (m *monitor) collectItems() []dockItem {
	var items []dockItem
	total := m.queued + boolInt(m.hasSel)
	if m.hasSel {
		txt := m.withKey(i18n.T("To collection"), settings.ActionAdd)
		if m.queued > 0 {
			txt = m.withKey(i18n.T("Add another"), settings.ActionAdd)
		}
		items = append(items, dockItem{&m.dock.add, txt, m.queued == 0, m.addToCollection})
	}
	if total >= 2 {
		txt := m.withKey(i18n.T("Glue"), settings.ActionGlue)
		if m.hasSel {
			txt = m.withKey(i18n.T("Glue with this area"), settings.ActionGlue)
		}
		items = append(items, dockItem{&m.dock.glue, txt, m.queued > 0, m.glueCollection})
	}
	if m.queued > 0 {
		items = append(items,
			dockItem{&m.dock.separate, m.withKey(i18n.T("As files"), settings.ActionSend), false, m.sendSeparately},
			dockItem{&m.dock.clear, i18n.T("Clear"), false, m.clearCollection})
	}
	return items
}

// layoutCollectDock draws the collection panel: the counter and the actions.
func (m *monitor) layoutCollectDock(gtx layout.Context, win image.Point) {
	items := m.collectItems()
	for _, it := range items {
		if it.btn.Clicked(gtx) {
			it.do()
			gtx.Execute(op.InvalidateCmd{}) // this frame is skipped, the panel with the new state is drawn right after
			return
		}
	}

	title := ""
	switch {
	case m.queued > 0 && !m.hasSel:
		title = fmt.Sprintf(i18n.T("Collection: %d  ·  select an area"), m.queued)
	case m.queued > 0:
		title = fmt.Sprintf(i18n.T("Collection: %d"), m.queued)
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
