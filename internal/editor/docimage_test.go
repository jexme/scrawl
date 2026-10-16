package editor

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/screencast"
)

// fakeDesktop рисует вымышленный экран с окном приложения: настоящие снимки
// пользователя в документацию не попадают.
func fakeDesktop(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(img, r.Intersect(img.Bounds()), image.NewUniform(c), image.Point{}, draw.Src)
	}
	fill(img.Bounds(), color.RGBA{R: 44, G: 62, B: 92, A: 255}) // обои
	win := image.Rect(90, 70, w-90, h-60)
	fill(win, color.RGBA{R: 250, G: 250, B: 252, A: 255})
	fill(image.Rect(win.Min.X, win.Min.Y, win.Max.X, win.Min.Y+34), color.RGBA{R: 226, G: 228, B: 234, A: 255})
	for i, c := range []color.RGBA{{R: 237, G: 106, B: 94, A: 255}, {R: 245, G: 191, B: 79, A: 255}, {R: 97, G: 197, B: 84, A: 255}} {
		fill(image.Rect(win.Min.X+14+i*22, win.Min.Y+11, win.Min.X+26+i*22, win.Min.Y+23), c)
	}
	// Боковая панель, строки текста и карточки.
	fill(image.Rect(win.Min.X, win.Min.Y+34, win.Min.X+190, win.Max.Y), color.RGBA{R: 238, G: 240, B: 245, A: 255})
	for i := 0; i < 8; i++ {
		fill(image.Rect(win.Min.X+20, win.Min.Y+60+i*30, win.Min.X+20+90+(i%3)*25, win.Min.Y+70+i*30), color.RGBA{R: 190, G: 196, B: 210, A: 255})
	}
	for i := 0; i < 3; i++ {
		x := win.Min.X + 220 + i*250
		fill(image.Rect(x, win.Min.Y+70, x+230, win.Min.Y+210), color.RGBA{R: 255, G: 255, B: 255, A: 255})
		fill(image.Rect(x, win.Min.Y+70, x+230, win.Min.Y+74), []color.RGBA{{R: 66, G: 133, B: 244, A: 255}, {R: 52, G: 168, B: 83, A: 255}, {R: 251, G: 188, B: 5, A: 255}}[i])
		for j := 0; j < 4; j++ {
			fill(image.Rect(x+16, win.Min.Y+96+j*24, x+16+120+(j*37)%80, win.Min.Y+106+j*24), color.RGBA{R: 205, G: 210, B: 222, A: 255})
		}
	}
	for i := 0; i < 7; i++ {
		fill(image.Rect(win.Min.X+220, win.Min.Y+250+i*26, win.Min.X+220+300+(i*53)%260, win.Min.Y+260+i*26), color.RGBA{R: 214, G: 218, B: 228, A: 255})
	}
	return img
}

// TestDocOverlayImage рисует настоящий оверлей (панель, рамка, подписи) поверх
// вымышленного экрана для README. Нужна переменная SCRAWL_DOC_IMAGES с папкой.
func TestDocOverlayImage(t *testing.T) {
	dir := os.Getenv("SCRAWL_DOC_IMAGES")
	if dir == "" {
		t.Skip("нужна переменная SCRAWL_DOC_IMAGES")
	}
	defer i18n.SetLanguage(i18n.RU)
	i18n.SetLanguage(i18n.EN)
	sandbox(t)

	const W, H = 1280, 760
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: fakeDesktop(W, H)}, func() {})
	m.sel, m.hasSel = image.Rect(250, 110, 1010, 500), true
	m.tool, m.color = annotate.ToolArrow, annotate.Palette[0]
	m.shapes = []annotate.Shape{
		{Tool: annotate.ToolRect, Color: annotate.Palette[0], Width: 4, Pts: []f32.Point{{X: 300, Y: 160}, {X: 520, Y: 330}}},
		{Tool: annotate.ToolArrow, Color: annotate.Palette[0], Width: 5, Pts: []f32.Point{{X: 760, Y: 420}, {X: 540, Y: 300}}},
		{Tool: annotate.ToolText, Color: annotate.Palette[0], Width: 5, Pts: []f32.Point{{X: 700, Y: 425}}, Text: "Check this card"},
		{Tool: annotate.ToolMarker, Color: annotate.Palette[2], Width: 5, Pts: []f32.Point{{X: 570, Y: 365}, {X: 700, Y: 365}}},
		{Tool: annotate.ToolBlur, Color: annotate.Palette[0], Width: 12, Pts: []f32.Point{{X: 790, Y: 150}, {X: 990, Y: 250}}},
	}

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	m.layout(gtx, image.Pt(W, H))
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "overlay.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		t.Fatal(err)
	}
}
