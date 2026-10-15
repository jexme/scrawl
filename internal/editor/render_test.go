package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/f32"

	"github.com/jexme/screen-editor/internal/annotate"
)

// TestRenderAnnotations рендерит аннотации поверх однотонного кадра и проверяет
// ключевые пиксели. С SCREEN_EDITOR_TEST_OUT=путь.png результат сохраняется для просмотра.
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

	// Координаты в результате сдвинуты на sel.Min.
	at := func(x, y int) color.RGBA { return out.RGBAAt(x-sel.Min.X, y-sel.Min.Y) }
	if c := at(110, 30); c != bg {
		t.Errorf("фон в углу: %v, ожидали %v", c, bg)
	}
	// Середина древка стрелки (275, 225) должна быть красной.
	if c := at(275, 225); c.R < 200 || c.G > 90 {
		t.Errorf("древко стрелки не красное: %v", c)
	}
	// Верхняя сторона жёлтой рамки.
	if c := at(190, 120); c.R < 200 || c.G < 180 {
		t.Errorf("рамка не жёлтая: %v", c)
	}

	if path := os.Getenv("SCREEN_EDITOR_TEST_OUT"); path != "" {
		data, err := encodePNG(out)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
