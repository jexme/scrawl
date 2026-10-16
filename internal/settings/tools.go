package settings

import (
	"encoding/json"
	"image/color"
	"slices"
)

// ToolNames — имена инструментов в файле настроек, в порядке инструментов
// редактора (annotate.Tool): выделение, перо, линия, стрелка, рамка, маркер,
// размытие, текст.
var ToolNames = []string{"select", "pen", "line", "arrow", "rect", "marker", "blur", "text"}

// Границы толщины инструмента (зеркало annotate.MinWidth/MaxWidth, сверяется тестом).
const (
	MinToolWidth = 1
	MaxToolWidth = 30
)

// ToolStyle — вид одного инструмента по умолчанию. Пустое значение: не задано.
type ToolStyle struct {
	// Color — цвет #RRGGBB. Пусто: не менять цвет при выборе инструмента.
	Color string `json:"color,omitempty"`
	// Width — толщина линии, размер текста или сила размытия. 0: стандартная.
	Width float64 `json:"width,omitempty"`
}

// ToolStyles — вид каждого инструмента по умолчанию.
type ToolStyles struct {
	Pen    ToolStyle `json:"pen"`
	Line   ToolStyle `json:"line"`
	Arrow  ToolStyle `json:"arrow"`
	Rect   ToolStyle `json:"rect"`
	Marker ToolStyle `json:"marker"`
	Blur   ToolStyle `json:"blur"`
	Text   ToolStyle `json:"text"`
}

// Of возвращает указатель на вид инструмента по имени (nil для неизвестного и «select»).
func (t *ToolStyles) Of(name string) *ToolStyle {
	switch name {
	case "pen":
		return &t.Pen
	case "line":
		return &t.Line
	case "arrow":
		return &t.Arrow
	case "rect":
		return &t.Rect
	case "marker":
		return &t.Marker
	case "blur":
		return &t.Blur
	case "text":
		return &t.Text
	}
	return nil
}

// MarshalJSON пишет только те инструменты, у которых что-то задано.
func (t ToolStyles) MarshalJSON() ([]byte, error) {
	out := map[string]ToolStyle{}
	for _, name := range ToolNames {
		if st := t.Of(name); st != nil && *st != (ToolStyle{}) {
			out[name] = *st
		}
	}
	return json.Marshal(out)
}

// normalize сбрасывает неверный цвет и подрезает толщину.
func (t *ToolStyles) normalize() {
	for _, name := range ToolNames {
		st := t.Of(name)
		if st == nil {
			continue
		}
		if c, err := ParseColor(st.Color); err == nil {
			st.Color = FormatColor(c)
		} else {
			st.Color = ""
		}
		if st.Width != 0 {
			st.Width = min(max(st.Width, MinToolWidth), MaxToolWidth)
		}
	}
}

// validTool сообщает, что имя входит в ToolNames.
func validTool(name string) bool { return slices.Contains(ToolNames, name) }

// PaletteSize — число цветов в панели инструментов.
const PaletteSize = 8

// DefaultPalette — цвета панели по умолчанию (сверяется с annotate.Palette тестом).
var DefaultPalette = [PaletteSize]string{
	"#E53935", "#FB8C00", "#FDD835", "#43A047", "#1E88E5", "#8E24AA", "#FFFFFF", "#141414",
}

// normalizePalette заменяет пустые и неверные цвета палитры цветами по умолчанию.
func normalizePalette(p *[PaletteSize]string) {
	for i, hex := range p {
		if c, err := ParseColor(hex); err == nil {
			p[i] = FormatColor(c)
		} else {
			p[i] = DefaultPalette[i]
		}
	}
}

// PaletteColors возвращает цвета панели инструментов.
func (s Settings) PaletteColors() [PaletteSize]color.NRGBA {
	var out [PaletteSize]color.NRGBA
	for i, hex := range s.Palette {
		c, err := ParseColor(hex)
		if err != nil {
			c, _ = ParseColor(DefaultPalette[i])
		}
		out[i] = c
	}
	return out
}
