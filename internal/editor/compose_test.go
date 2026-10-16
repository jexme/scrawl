package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/jexme/screen-editor/internal/annotate"
	"github.com/jexme/screen-editor/internal/settings"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func TestComposeSideBySide(t *testing.T) {
	// Разная высота: результат берёт самую низкую, без увеличения.
	a := solid(300, 200, color.RGBA{R: 220, G: 60, B: 60, A: 255})
	b := solid(240, 160, color.RGBA{R: 60, G: 170, B: 80, A: 255})
	out, err := compose([]*image.RGBA{a, b}, settings.Default().Glue.Labels(2), annotate.NewShaper(), settings.DefaultGlue())
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
	if _, err := compose([]*image.RGBA{a}, []string{"x"}, annotate.NewShaper(), settings.DefaultGlue()); err == nil {
		t.Error("одна картинка должна давать ошибку")
	}
	if path := os.Getenv("SCREEN_EDITOR_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".compose.png", data, 0o644)
	}
}

func TestComposeStacked(t *testing.T) {
	// Разная ширина: результат берёт самую узкую, без увеличения.
	a := solid(300, 200, color.RGBA{R: 220, G: 60, B: 60, A: 255})
	b := solid(240, 160, color.RGBA{R: 60, G: 170, B: 80, A: 255})
	glue := settings.DefaultGlue()
	glue.Layout = settings.GlueStack
	out, err := compose([]*image.RGBA{a, b}, glue.Labels(2), annotate.NewShaper(), glue)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	// Ширина b = 240; высота a после масштаба = 200*240/300 = 160.
	wantW := composePad*2 + 240
	wantH := composePad*2 + composeGap + 2*composeLabelH + 160 + 160
	if got := out.Bounds().Size(); got != image.Pt(wantW, wantH) {
		t.Fatalf("размер %v, ожидали %v", got, image.Pt(wantW, wantH))
	}
	// Красная картинка сверху, зелёная под ней.
	if c := out.RGBAAt(composePad+120, composePad+composeLabelH+80); c.R < 180 || c.G > 100 {
		t.Errorf("верхняя картинка не красная: %v", c)
	}
	if c := out.RGBAAt(composePad+120, composePad+composeLabelH+160+composeGap+composeLabelH+80); c.G < 130 || c.R > 100 {
		t.Errorf("нижняя картинка не зелёная: %v", c)
	}
	if path := os.Getenv("SCREEN_EDITOR_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".compose-stack.png", data, 0o644)
	}
}
