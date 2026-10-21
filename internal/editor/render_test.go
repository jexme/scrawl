package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/f32"

	"github.com/jexme/scrawl/internal/annotate"
)

// TestRenderAnnotations renders annotations over a solid frame and checks the key
// pixels. With SCRAWL_TEST_OUT=path.png the result is saved for inspection.
func TestRenderAnnotations(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 600, 400))
	bg := color.RGBA{R: 40, G: 60, B: 90, A: 255}
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = bg.R, bg.G, bg.B, bg.A
	}

	red := annotate.Palette[0]
	yellow := annotate.Palette[2]
	white := annotate.Palette[6]
	shapes := []annotate.Shape{
		{Tool: annotate.ToolArrow, Color: red, Width: 6, Pts: []f32.Point{{X: 150, Y: 300}, {X: 400, Y: 150}}},
		{Tool: annotate.ToolRect, Color: yellow, Width: 4, Pts: []f32.Point{{X: 120, Y: 120}, {X: 260, Y: 220}}},
		{Tool: annotate.ToolMarker, Color: yellow, Width: 4, Pts: []f32.Point{{X: 300, Y: 320}, {X: 520, Y: 320}}},
		{Tool: annotate.ToolText, Color: white, Width: 4, Pts: []f32.Point{{X: 140, Y: 40}}, Text: "Привет, Hello"},
	}

	sel := image.Rect(100, 20, 580, 380)
	out, err := render(src, sel, shapes, annotate.NewShaper())
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	if got := out.Bounds().Size(); got != sel.Size() {
		t.Fatalf("размер результата %v, ожидали %v", got, sel.Size())
	}

	// Coordinates in the result are shifted by sel.Min.
	at := func(x, y int) color.RGBA { return out.RGBAAt(x-sel.Min.X, y-sel.Min.Y) }
	if c := at(110, 30); c != bg {
		t.Errorf("фон в углу: %v, ожидали %v", c, bg)
	}
	// The middle of the arrow shaft (275, 225) must be red.
	if c := at(275, 225); c.R < 200 || c.G > 90 {
		t.Errorf("древко стрелки не красное: %v", c)
	}
	// The top side of the yellow rectangle.
	if c := at(190, 120); c.R < 200 || c.G < 180 {
		t.Errorf("рамка не жёлтая: %v", c)
	}

	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, err := encodePNG(out)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestRenderBlur checks that the blur tool wipes out small details inside the
// rectangle and leaves the rest alone. The frame: black and white vertical stripes 2 px wide.
func TestRenderBlur(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 400; x++ {
			v := uint8(0)
			if (x/2)%2 == 0 {
				v = 255
			}
			src.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	shapes := []annotate.Shape{
		{Tool: annotate.ToolBlur, Width: annotate.DefaultBlurLevel, Pts: []f32.Point{{X: 100, Y: 60}, {X: 300, Y: 180}}},
	}
	sel := image.Rect(0, 0, 400, 240)
	out, err := render(src, sel, shapes, annotate.NewShaper())
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}

	// The spread of values in a row: stripes outside the rectangle, almost flat grey inside.
	spread := func(y, x0, x1 int) int {
		lo, hi := 255, 0
		for x := x0; x < x1; x++ {
			v := int(out.RGBAAt(x, y).R)
			lo, hi = min(lo, v), max(hi, v)
		}
		return hi - lo
	}
	if s := spread(120, 20, 80); s < 200 {
		t.Errorf("снаружи рамки полосы должны остаться: размах %d", s)
	}
	if s := spread(120, 140, 260); s > 60 {
		t.Errorf("внутри рамки должно быть размыто: размах %d", s)
	}
	if mid := int(out.RGBAAt(200, 120).R); mid < 90 || mid > 170 {
		t.Errorf("центр размытой области должен быть серым: %d", mid)
	}

	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".blur.png", data, 0o644)
	}
}

func TestShadeColor(t *testing.T) {
	for percent, want := range map[int]uint8{0: 0, 86: 219, 100: 255, -10: 0, 500: 255, 50: 128} {
		if got := shadeColor(percent).A; got != want {
			t.Errorf("затемнение %d%%: альфа %d, ожидали %d", percent, got, want)
		}
	}
}
