package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/settings"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func TestComposeSideBySide(t *testing.T) {
	pad, labelH := composeMetrics(settings.DefaultGlue())
	// Разная высота: результат берёт самую низкую, без увеличения.
	a := solid(300, 200, color.RGBA{R: 220, G: 60, B: 60, A: 255})
	b := solid(240, 160, color.RGBA{R: 60, G: 170, B: 80, A: 255})
	out, err := compose([]*image.RGBA{a, b}, settings.Default().Glue.Labels(2), annotate.NewShaper(), settings.DefaultGlue())
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	// Высота кадра b = 160, ширина a после масштаба = 300*160/200 = 240.
	wantW := pad*2 + pad + 240 + 240
	wantH := pad*2 + labelH + 160
	if got := out.Bounds().Size(); got != image.Pt(wantW, wantH) {
		t.Fatalf("размер %v, ожидали %v", got, image.Pt(wantW, wantH))
	}
	// В середине первой картинки красный, в середине второй зелёный.
	mid := pad + labelH + 80
	if c := out.RGBAAt(pad+120, mid); c.R < 180 || c.G > 100 {
		t.Errorf("первая картинка не красная: %v", c)
	}
	if c := out.RGBAAt(pad+240+pad+120, mid); c.G < 130 || c.R > 100 {
		t.Errorf("вторая картинка не зелёная: %v", c)
	}
	// Одной картинки мало.
	if _, err := compose([]*image.RGBA{a}, []string{"x"}, annotate.NewShaper(), settings.DefaultGlue()); err == nil {
		t.Error("одна картинка должна давать ошибку")
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".compose.png", data, 0o644)
	}
}

func TestComposeStacked(t *testing.T) {
	pad, labelH := composeMetrics(settings.DefaultGlue())
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
	wantW := pad*2 + 240
	wantH := pad*2 + pad + 2*labelH + 160 + 160
	if got := out.Bounds().Size(); got != image.Pt(wantW, wantH) {
		t.Fatalf("размер %v, ожидали %v", got, image.Pt(wantW, wantH))
	}
	// Красная картинка сверху, зелёная под ней.
	if c := out.RGBAAt(pad+120, pad+labelH+80); c.R < 180 || c.G > 100 {
		t.Errorf("верхняя картинка не красная: %v", c)
	}
	if c := out.RGBAAt(pad+120, pad+labelH+160+pad+labelH+80); c.G < 130 || c.R > 100 {
		t.Errorf("нижняя картинка не зелёная: %v", c)
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".compose-stack.png", data, 0o644)
	}
}

func TestComposeLook(t *testing.T) {
	a := solid(300, 200, color.RGBA{R: 220, G: 60, B: 60, A: 255})
	b := solid(300, 200, color.RGBA{R: 60, G: 170, B: 80, A: 255})
	glue := settings.DefaultGlue()
	glue.Background, glue.Pad, glue.LabelSize = settings.GlueDark, 10, 40
	out, err := compose([]*image.RGBA{a, b}, glue.Labels(2), annotate.NewShaper(), glue)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	pad, labelH := composeMetrics(glue)
	if pad != 10 || labelH != 64 {
		t.Fatalf("отступ %d, строка подписи %d", pad, labelH)
	}
	if got, want := out.Bounds().Size(), image.Pt(pad*3+600, pad*2+labelH+200); got != want {
		t.Fatalf("размер %v, ожидали %v", got, want)
	}
	// Угол и промежуток между картинками закрашены тёмным фоном.
	for _, p := range []image.Point{{2, 2}, {pad + 300 + pad/2, pad + labelH + 100}} {
		if c := out.RGBAAt(p.X, p.Y); c.R != 32 || c.G != 33 || c.B != 36 {
			t.Errorf("в точке %v ожидали тёмный фон, получили %v", p, c)
		}
	}
	if c := out.RGBAAt(pad+150, pad+labelH+100); c.R < 180 {
		t.Errorf("картинка на месте: %v", c)
	}

	// Нулевой отступ: картинки вплотную, склейка всё равно собирается.
	glue.Pad = 0
	flat, err := compose([]*image.RGBA{a, b}, glue.Labels(2), annotate.NewShaper(), glue)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := flat.Bounds().Size(), image.Pt(600, labelH+200); got != want {
		t.Errorf("без отступов %v, ожидали %v", got, want)
	}
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, _ := encodePNG(out)
		os.WriteFile(path+".compose-dark.png", data, 0o644)
	}
}
