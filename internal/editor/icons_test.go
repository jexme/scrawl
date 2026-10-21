package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// TestIcons draws all the panel icons on a dark background and checks that each one
// drew something. With SCRAWL_TEST_OUT=path the image is saved.
func TestIcons(t *testing.T) {
	icons := []struct {
		name string
		fn   iconFunc
	}{
		{"select", iconSelect}, {"pen", iconPen}, {"line", iconLine}, {"arrow", iconArrow},
		{"rect", iconRect}, {"marker", iconMarker}, {"blur", iconBlur}, {"text", iconText},
		{"minus", iconMinus}, {"plus", iconPlus}, {"undo", iconUndo}, {"copy", iconCopy},
		{"save", iconSave}, {"close", iconClose},
	}
	const cell = 72
	W, H := cell*len(icons), cell*2
	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()

	bg := color.NRGBA{R: 32, G: 33, B: 36, A: 255}
	fg := color.NRGBA{R: 235, G: 235, B: 235, A: 255}
	var ops op.Ops
	paint.Fill(&ops, bg)
	for i, ic := range icons {
		// Top row: the panel button size (34 px), bottom row: enlarged (72 px).
		st := op.Offset(image.Pt(i*cell+(cell-34)/2, (cell-34)/2)).Push(&ops)
		ic.fn(&ops, 34, fg)
		st.Pop()
		st = op.Offset(image.Pt(i*cell, cell)).Push(&ops)
		ic.fn(&ops, cell, fg)
		st.Pop()
	}
	_ = f32.Pt
	_ = clip.Rect{}
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}

	for i, ic := range icons {
		painted := 0
		for y := cell; y < 2*cell; y++ {
			for x := i * cell; x < (i+1)*cell; x++ {
				if c := out.RGBAAt(x, y); c.R > 100 {
					painted++
				}
			}
		}
		if painted < 40 {
			t.Errorf("иконка %s почти ничего не нарисовала (%d пикселей)", ic.name, painted)
		}
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".icons.png", data, 0o644)
	}
}
