package settingsui

import (
	"math"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/scrawl/internal/settings"
)

// slider is a slider with real bounds and step: widget.Float stores only 0..1.
type slider struct {
	f              widget.Float
	min, max, step float64
	format         func(float64) string // value label on the right

	// get and set connect the slider to a setting.
	get func(settings.Settings) float64
	set func(*settings.Settings, float64)
}

// Value returns the value rounded to the step (and to thousandths, so that 4.6 does not
// turn into 4.6000004 and get treated as a change).
func (sl *slider) Value() float64 {
	v := sl.min + float64(sl.f.Value)*(sl.max-sl.min)
	v = math.Round((v-sl.min)/sl.step)*sl.step + sl.min
	v = math.Round(v*1000) / 1000
	return min(max(v, sl.min), sl.max)
}

// Set sets the value (clamped to the bounds and step).
func (sl *slider) Set(v float64) {
	v = min(max(v, sl.min), sl.max)
	sl.f.Value = float32((v - sl.min) / (sl.max - sl.min))
	sl.f.Value = float32((sl.Value() - sl.min) / (sl.max - sl.min))
}

// Update handles dragging and snaps the slider to the step when it is released.
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
