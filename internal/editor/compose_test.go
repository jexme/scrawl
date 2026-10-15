package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/jexme/screen-editor/internal/annotate"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func TestComposeLabels(t *testing.T) {
	if got := composeLabels(2); got[0] != "Было" || got[1] != "Стало" {
		t.Errorf("для двух: %v", got)
	}
	if got := composeLabels(3); got[0] != "Шаг 1" || got[2] != "Шаг 3" {
		t.Errorf("для трёх: %v", got)
	}
}

func TestComposeSideBySide(t *testing.T) {
	// Разная высота: результат берёт самую низкую, без увеличения.
	a := solid(300, 200, color.RGBA{R: 220, G: 60, B: 60, A: 255})
	b := solid(240, 160, color.RGBA{R: 60, G: 170, B: 80, A: 255})
	out, err := compose([]*image.RGBA{a, b}, composeLabels(2), annotate.NewShaper())
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	// Высота кадра b = 160, ширина a после масштаба = 300*160/200 = 240.
	wantW := composePad*2 + composeGap + 240 + 240
	wantH := composePad*2 + composeLabelH + 160
	if got := out.Bounds().Size(); got != image.Pt(wantW, wantH) {
		t.Fatalf("размер %v, ожидали %v", got, image.Pt(wantW, wantH))
	}
	// В середине первой картинки красный, в середине второй зелёный.
	mid := composePad + composeLabelH + 80
	if c := out.RGBAAt(composePad+120, mid); c.R < 180 || c.G > 100 {
		t.Errorf("первая картинка не красная: %v", c)
	}
	if c := out.RGBAAt(composePad+240+composeGap+120, mid); c.G < 130 || c.R > 100 {
		t.Errorf("вторая картинка не зелёная: %v", c)
	}
	// Одной картинки мало.
	if _, err := compose([]*image.RGBA{a}, []string{"x"}, annotate.NewShaper()); err == nil {
		t.Error("одна картинка должна давать ошибку")
	}
	if path := os.Getenv("SCREEN_EDITOR_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".compose.png", data, 0o644)
	}
}
