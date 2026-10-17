// Package annotate описывает аннотации поверх скриншота (стрелки, фигуры, текст)
// и рисует их в Gio. Один и тот же код используется для показа на экране и для
// экспорта в PNG, поэтому результат совпадает с тем, что видел пользователь.
//
// Все координаты и размеры задаются в пикселях кадра.
package annotate

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/fonts"
	"github.com/jexme/scrawl/internal/i18n"
)

// Tool — инструмент рисования.
type Tool int

const (
	ToolSelect Tool = iota // выделение области
	ToolPen
	ToolLine
	ToolArrow
	ToolRect
	ToolMarker
	ToolBlur
	ToolText
	ToolStamp // штамп с номером: кружок с цифрой, номера идут по порядку
)

// Palette — цвета в панели инструментов.
var Palette = []color.NRGBA{
	{R: 229, G: 57, B: 53, A: 255},   // красный
	{R: 251, G: 140, B: 0, A: 255},   // оранжевый
	{R: 253, G: 216, B: 53, A: 255},  // жёлтый
	{R: 67, G: 160, B: 71, A: 255},   // зелёный
	{R: 30, G: 136, B: 229, A: 255},  // синий
	{R: 142, G: 36, B: 170, A: 255},  // фиолетовый
	{R: 255, G: 255, B: 255, A: 255}, // белый
	{R: 20, G: 20, B: 20, A: 255},    // чёрный
}

const (
	DefaultWidth = 4
	MinWidth     = 1
	MaxWidth     = 30
)

// DefaultWidthOf возвращает толщину инструмента по умолчанию: у размытия это уровень.
func DefaultWidthOf(t Tool) float32 {
	if t == ToolBlur {
		return DefaultBlurLevel
	}
	return DefaultWidth
}

// MaxWidthOf возвращает наибольшую толщину инструмента: у размытия это последний уровень.
func MaxWidthOf(t Tool) float32 {
	if t == ToolBlur {
		return MaxBlurLevel
	}
	return MaxWidth
}

// Shape — одна аннотация.
type Shape struct {
	Tool  Tool
	Color color.NRGBA
	Width float32     // толщина линии, пиксели кадра
	Pts   []f32.Point // ключевые точки
	Text  string      // только для ToolText и ToolStamp (номер)
	// Alpha — непрозрачность маркера, 1-255; 0 означает DefaultMarkerAlpha.
	Alpha uint8
}

// DefaultMarkerAlpha — непрозрачность маркера, если не задана иначе (43%).
const DefaultMarkerAlpha = 110

// TextSize возвращает размер шрифта текстовой аннотации в пикселях кадра.
func TextSize(width float32) float32 { return 12 + width*3 }

// StampRadius возвращает радиус штампа с номером по «толщине» инструмента.
func StampRadius(width float32) float32 { return 12 + width*2.5 }

// NewShaper создаёт шрифтовой шейпер для текстовых аннотаций.
func NewShaper() *text.Shaper {
	return text.NewShaper(text.WithCollection(fonts.Collection(i18n.Language())))
}

// Font — шрифт текстовых аннотаций.
var Font = gofont.Collection()[0].Font

// Degenerate сообщает, что фигура слишком мала, чтобы её сохранять.
func (s Shape) Degenerate() bool {
	switch s.Tool {
	case ToolText:
		return s.Text == ""
	case ToolBlur:
		if len(s.Pts) < 2 {
			return true
		}
		r := RectOf(s.Pts[0], s.Pts[1])
		return r.Dx() < 4 || r.Dy() < 4
	case ToolPen, ToolStamp:
		return len(s.Pts) < 1
	default:
		if len(s.Pts) < 2 {
			return true
		}
		return dist(s.Pts[0], s.Pts[1]) < 3
	}
}

// Draw рисует фигуру. Координаты фигуры считаются координатами пространства ops.
func (s Shape) Draw(ops *op.Ops, env Env) {
	if len(s.Pts) == 0 {
		return
	}
	switch s.Tool {
	case ToolPen:
		stroke(ops, s.Color, s.Width, s.Pts...)
	case ToolLine:
		if len(s.Pts) >= 2 {
			stroke(ops, s.Color, s.Width, s.Pts[0], s.Pts[1])
		}
	case ToolArrow:
		if len(s.Pts) >= 2 {
			drawArrow(ops, s.Color, s.Width, s.Pts[0], s.Pts[1])
		}
	case ToolRect:
		if len(s.Pts) >= 2 {
			a, b := s.Pts[0], s.Pts[1]
			strokeClosed(ops, s.Color, s.Width,
				f32.Pt(a.X, a.Y), f32.Pt(b.X, a.Y), f32.Pt(b.X, b.Y), f32.Pt(a.X, b.Y))
		}
	case ToolMarker:
		c := s.Color
		c.A = s.Alpha
		if c.A == 0 {
			c.A = DefaultMarkerAlpha
		}
		stroke(ops, c, s.Width*4, s.Pts...)
	case ToolBlur:
		s.drawBlur(ops, env)
	case ToolText:
		drawText(ops, env.Shaper, s.Color, s.Width, s.Pts[0], s.Text)
	case ToolStamp:
		drawStamp(ops, env.Shaper, s.Color, s.Width, s.Pts[0], s.Text)
	}
}

// DrawAll рисует все фигуры по порядку.
func DrawAll(ops *op.Ops, env Env, shapes []Shape) {
	for _, s := range shapes {
		s.Draw(ops, env)
	}
}

func dist(a, b f32.Point) float32 {
	return float32(math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y)))
}

func stroke(ops *op.Ops, c color.NRGBA, width float32, pts ...f32.Point) {
	if len(pts) == 0 {
		return
	}
	if len(pts) == 1 {
		// Точка: короткий отрезок, чтобы штрих был виден.
		pts = []f32.Point{pts[0], pts[0].Add(f32.Pt(0.01, 0))}
	}
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(pts[0])
	for _, pt := range pts[1:] {
		p.LineTo(pt)
	}
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

func strokeClosed(ops *op.Ops, c color.NRGBA, width float32, pts ...f32.Point) {
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(pts[0])
	for _, pt := range pts[1:] {
		p.LineTo(pt)
	}
	p.Close()
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

// drawArrow рисует линию с треугольным наконечником в точке to.
func drawArrow(ops *op.Ops, c color.NRGBA, width float32, from, to f32.Point) {
	d := to.Sub(from)
	length := dist(from, to)
	if length < 1 {
		return
	}
	dir := f32.Pt(d.X/length, d.Y/length)
	normal := f32.Pt(-dir.Y, dir.X)

	head := float32(math.Max(float64(width*4), 14))
	if head > length {
		head = length
	}
	half := head * 0.5
	base := to.Sub(dir.Mul(head))

	// Древко заканчивается внутри наконечника, чтобы не было щели.
	stroke(ops, c, width, from, base.Add(dir.Mul(head*0.3)))

	var p clip.Path
	p.Begin(ops)
	p.MoveTo(to)
	p.LineTo(base.Add(normal.Mul(half)))
	p.LineTo(base.Sub(normal.Mul(half)))
	p.Close()
	paint.FillShape(ops, c, clip.Outline{Path: p.End()}.Op())
}

// drawText рисует текст в пространстве кадра: метрика единичная, поэтому
// размер шрифта задаётся в пикселях кадра независимо от масштаба окна.
func drawText(ops *op.Ops, shaper *text.Shaper, c color.NRGBA, width float32, at f32.Point, txt string) {
	if txt == "" || shaper == nil {
		return
	}
	defer op.Offset(image.Pt(int(at.X), int(at.Y))).Push(ops).Pop()
	gtx := layout.Context{
		Ops:         ops,
		Constraints: layout.Constraints{Max: image.Pt(1<<14, 1<<14)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	rec := op.Record(ops)
	paint.ColorOp{Color: c}.Add(ops)
	mat := rec.Stop()
	widget.Label{}.Layout(gtx, shaper, Font, unit.Sp(TextSize(width)), txt, mat)
}

// onColor возвращает цвет текста, читаемый на фоне c: тёмный на светлом, белый на тёмном.
func onColor(c color.NRGBA) color.NRGBA {
	if lum := 0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B); lum > 150 {
		return color.NRGBA{R: 20, G: 20, B: 20, A: 255}
	}
	return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}

// stampTextSize подбирает размер цифр в штампе: чем длиннее номер, тем мельче.
func stampTextSize(r float32, digits int) float32 {
	switch {
	case digits <= 1:
		return r * 1.25
	case digits == 2:
		return r * 1.0
	case digits == 3:
		return r * 0.8
	}
	return r * 0.64
}

// stampFont — шрифт цифр штампа: жирный, чтобы цифры читались на любом фоне.
var stampFont = font.Font{Typeface: Font.Typeface, Weight: font.Bold}

// drawStamp рисует штамп: цветной круг в ободке цвета цифр (светлом на тёмном, тёмном на
// светлом), мягкая тень снизу и жирный номер строго в середине.
func drawStamp(ops *op.Ops, shaper *text.Shaper, c color.NRGBA, width float32, at f32.Point, num string) {
	r := StampRadius(width)
	fg := onColor(c)

	circle := func(radius, dy float32) clip.Op {
		return clip.Ellipse(image.Rect(
			int(math.Round(float64(at.X-radius))), int(math.Round(float64(at.Y+dy-radius))),
			int(math.Round(float64(at.X+radius))), int(math.Round(float64(at.Y+dy+radius))))).Op(ops)
	}
	// Тень: несколько прозрачных кругов друг в друге, чуть сдвинутых вниз.
	for _, sh := range []struct {
		grow float32
		a    uint8
	}{{0.16, 14}, {0.10, 18}, {0.05, 22}} {
		paint.FillShape(ops, color.NRGBA{A: sh.a}, circle(r*(1+sh.grow), r*0.08))
	}
	paint.FillShape(ops, fg, circle(r, 0))
	paint.FillShape(ops, c, circle(r*0.84, 0))

	if num == "" || shaper == nil {
		return
	}
	gtx := layout.Context{
		Ops:         ops,
		Constraints: layout.Constraints{Max: image.Pt(1<<14, 1<<14)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	size := stampTextSize(r, len(num))
	rec := op.Record(ops)
	paint.ColorOp{Color: fg}.Add(ops)
	mat := rec.Stop()
	rec = op.Record(ops)
	dims := widget.Label{MaxLines: 1}.Layout(gtx, shaper, stampFont, unit.Sp(size), num, mat)
	label := rec.Stop()
	// Цифра стоит на базовой линии: середину цифр (половина её высоты над линией) ставим в центр круга.
	baseline := float32(dims.Size.Y - dims.Baseline)
	top := at.Y - baseline + size*digitHeight/2
	defer op.Offset(image.Pt(int(math.Round(float64(at.X)))-dims.Size.X/2, int(math.Round(float64(top))))).Push(ops).Pop()
	label.Add(ops)
}

// digitHeight — высота цифры шрифта Go в долях размера шрифта.
const digitHeight = 0.72
