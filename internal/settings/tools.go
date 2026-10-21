package settings

import (
	"encoding/json"
	"image/color"
	"slices"
)

// ToolNames are the tool names in the settings file, in the order of the editor's
// tools (annotate.Tool): select, pen, line, arrow, rectangle, marker,
// blur, text, numbered stamp.
var ToolNames = []string{"select", "pen", "line", "arrow", "rect", "marker", "blur", "text", "stamp"}

// Bounds of the tool thickness (mirror annotate.MinWidth/MaxWidth, checked by a test).
const (
	MinToolWidth = 1
	MaxToolWidth = 30
	// MaxBlurLevel is the highest blur strength level (mirrors annotate.MaxBlurLevel):
	// 1-3 blur, 4-6 mosaic.
	MaxBlurLevel = 6
)

// MaxWidthOf returns the largest thickness of the tool named name.
func MaxWidthOf(name string) float64 {
	if name == "blur" {
		return MaxBlurLevel
	}
	return MaxToolWidth
}

// ToolStyle is the default look of a single tool. The zero value means not set.
type ToolStyle struct {
	// Color is the colour #RRGGBB. Empty means do not change the colour when the tool is selected.
	Color string `json:"color,omitempty"`
	// Width is the line thickness, text size or blur strength. 0 means the standard one.
	Width float64 `json:"width,omitempty"`
}

// ToolStyles is the default look of each tool.
type ToolStyles struct {
	Pen    ToolStyle `json:"pen"`
	Line   ToolStyle `json:"line"`
	Arrow  ToolStyle `json:"arrow"`
	Rect   ToolStyle `json:"rect"`
	Marker ToolStyle `json:"marker"`
	Blur   ToolStyle `json:"blur"`
	Text   ToolStyle `json:"text"`
	Stamp  ToolStyle `json:"stamp"`
}

// Of returns a pointer to the tool's style by name (nil for an unknown name and for "select").
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
	case "stamp":
		return &t.Stamp
	}
	return nil
}

// MarshalJSON writes only the tools that have something set.
func (t ToolStyles) MarshalJSON() ([]byte, error) {
	out := map[string]ToolStyle{}
	for _, name := range ToolNames {
		if st := t.Of(name); st != nil && *st != (ToolStyle{}) {
			out[name] = *st
		}
	}
	return json.Marshal(out)
}

// normalize resets an invalid colour and clamps the thickness.
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
			st.Width = min(max(st.Width, MinToolWidth), MaxWidthOf(name))
		}
	}
}

// validTool reports whether the name is in ToolNames.
func validTool(name string) bool { return slices.Contains(ToolNames, name) }

// PaletteSize is the number of colours in the toolbar.
const PaletteSize = 8

// DefaultPalette is the default toolbar colours (checked against annotate.Palette by a test).
var DefaultPalette = [PaletteSize]string{
	"#E53935", "#FB8C00", "#FDD835", "#43A047", "#1E88E5", "#8E24AA", "#FFFFFF", "#141414",
}

// normalizePalette replaces empty and invalid palette colours with the defaults.
func normalizePalette(p *[PaletteSize]string) {
	for i, hex := range p {
		if c, err := ParseColor(hex); err == nil {
			p[i] = FormatColor(c)
		} else {
			p[i] = DefaultPalette[i]
		}
	}
}

// PaletteColors returns the toolbar colours.
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
