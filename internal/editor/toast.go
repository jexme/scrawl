package editor

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/i18n"
)

const (
	toastLife = 1300 * time.Millisecond
	toastFade = 350 * time.Millisecond
)

// widthLabel подписывает значение силы для разных инструментов.
func widthLabel(t annotate.Tool) string {
	switch t {
	case annotate.ToolText:
		return i18n.T("Text size")
	case annotate.ToolBlur:
		return i18n.T("Blur strength")
	case annotate.ToolStamp:
		return i18n.T("Stamp size")
	case annotate.ToolMarker:
		return i18n.T("Marker thickness")
	default:
		return i18n.T("Thickness")
	}
}

// showToast показывает на секунду плашку с новым значением силы инструмента.
func (m *monitor) showToast() {
	m.toastMsg = ""
	m.toastTool = m.tool
	m.toastValue = int(m.curWidth())
	m.toastUntil = time.Now().Add(toastLife)
}

// showMessage показывает на пару секунд плашку с готовым текстом, например
// «Скопировано в буфер обмена».
func (m *monitor) showMessage(txt string) {
	m.toastMsg = txt
	m.toastUntil = time.Now().Add(toastLife + 700*time.Millisecond)
}

// layoutToast рисует плашку «Толщина: N» с образцом линии и плавно гасит её.
// Она висит в верхней части выделения и не привязана к курсору.
func (m *monitor) layoutToast(gtx layout.Context, win image.Point, selWin image.Rectangle) {
	remain := time.Until(m.toastUntil)
	if remain <= 0 {
		return
	}
	fade := float32(1)
	if remain < toastFade {
		fade = float32(remain) / float32(toastFade)
		gtx.Execute(op.InvalidateCmd{}) // анимация затухания идёт кадр за кадром
	} else {
		gtx.Execute(op.InvalidateCmd{At: m.toastUntil.Add(-toastFade)})
	}
	scale := func(c color.NRGBA) color.NRGBA {
		c.A = uint8(float32(c.A) * fade)
		return c
	}

	txt := fmt.Sprintf("%s: %d", widthLabel(m.toastTool), m.toastValue)
	if m.toastMsg != "" {
		txt = m.toastMsg
	} else if m.toastTool == annotate.ToolBlur {
		// У размытия сила меняет не только степень, но и вид: мягкое, мозаика, матовое стекло.
		txt += "  ·  " + annotate.ModeOf(float32(m.toastValue)).Label()
	}
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: scale(iconColor)}.Add(gtx.Ops)
	mat := rec.Stop()
	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(1<<12, 1<<12)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, 17, txt, mat)
	label := rec.Stop()

	// Образец: отрезок нужной толщины тем же цветом. Для текста и размытия образца нет.
	sample := m.toastMsg == "" && m.toastTool != annotate.ToolText && m.toastTool != annotate.ToolBlur && m.toastTool != annotate.ToolStamp
	sampleW := 0
	lineW := float32(m.toastValue)
	if m.toastTool == annotate.ToolMarker {
		lineW *= 4
	}
	if sample {
		sampleW = gtx.Dp(54)
	}

	padX, padY, gap := gtx.Dp(16), gtx.Dp(10), gtx.Dp(12)
	contentH := max(dims.Size.Y, int(lineW)+2)
	if sample {
		contentH = max(contentH, gtx.Dp(22))
	}
	w := padX + sampleW + dims.Size.X + padX
	if sample {
		w += gap
	}
	h := contentH + 2*padY

	x := selWin.Min.X + (selWin.Dx()-w)/2
	y := selWin.Min.Y + gtx.Dp(16)
	x = clampInt(x, gtx.Dp(8), max(win.X-w-gtx.Dp(8), gtx.Dp(8)))
	if selWin.Dy() < h+gtx.Dp(32) { // выделение низкое: ставим ниже центра окна
		y = clampInt(selWin.Min.Y, gtx.Dp(8), max(win.Y-h-gtx.Dp(8), gtx.Dp(8)))
	}

	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	pill := clip.UniformRRect(image.Rect(0, 0, w, h), min(h/2, gtx.Dp(14)))
	paint.FillShape(gtx.Ops, scale(barBG), pill.Op(gtx.Ops))

	cursor := padX
	if sample {
		col := m.color
		if m.toastTool == annotate.ToolMarker {
			col.A = m.store.Get().MarkerAlpha()
		}
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(f32.Pt(float32(cursor), float32(h)/2))
		p.LineTo(f32.Pt(float32(cursor+sampleW), float32(h)/2))
		paint.FillShape(gtx.Ops, scale(col), clip.Stroke{Path: p.End(), Width: lineW}.Op())
		cursor += sampleW + gap
	}
	st := op.Offset(image.Pt(cursor, (h-dims.Size.Y)/2)).Push(gtx.Ops)
	label.Add(gtx.Ops)
	st.Pop()
}
