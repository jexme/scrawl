package settingsui

import (
	"math"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/screen-editor/internal/settings"
)

// slider — ползунок с настоящими границами и шагом: widget.Float хранит только 0..1.
type slider struct {
	f              widget.Float
	min, max, step float64
	format         func(float64) string // подпись значения справа

	// get и set связывают ползунок с настройкой.
	get func(settings.Settings) float64
	set func(*settings.Settings, float64)
}

// Value возвращает значение, округлённое до шага (и до тысячных, чтобы 4.6 не
// превращалось в 4.6000004 и не воспринималось как изменение).
func (sl *slider) Value() float64 {
	v := sl.min + float64(sl.f.Value)*(sl.max-sl.min)
	v = math.Round((v-sl.min)/sl.step)*sl.step + sl.min
	v = math.Round(v*1000) / 1000
	return min(max(v, sl.min), sl.max)
}

// Set ставит значение (подрезается до границ и шага).
func (sl *slider) Set(v float64) {
	v = min(max(v, sl.min), sl.max)
	sl.f.Value = float32((v - sl.min) / (sl.max - sl.min))
	sl.f.Value = float32((sl.Value() - sl.min) / (sl.max - sl.min))
}

// Update обрабатывает перетаскивание и прижимает ползунок к шагу, когда его отпустили.
func (sl *slider) Update(gtx layout.Context) {
	sl.f.Update(gtx)
	if !sl.f.Dragging() {
		sl.Set(sl.Value())
	}
}

func (s *state) slider(sl *slider) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.Slider(s.th, &sl.f).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(64)
				return s.label(material.Body1, sl.format(sl.Value()), fgColor)(gtx)
			}),
		)
	}
}
