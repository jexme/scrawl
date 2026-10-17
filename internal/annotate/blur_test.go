package annotate

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"
)

func TestBlurModes(t *testing.T) {
	want := []BlurMode{BlurSoft, BlurSoft, BlurSoft, BlurMosaic, BlurMosaic, BlurMosaic}
	for level := 1; level <= MaxBlurLevel; level++ {
		if got := ModeOf(float32(level)); got != want[level-1] {
			t.Errorf("уровень %d: режим %v, ожидали %v", level, got, want[level-1])
		}
	}
	if ModeOf(DefaultBlurLevel) != BlurSoft || DefaultWidthOf(ToolBlur) != DefaultBlurLevel || DefaultWidthOf(ToolPen) != DefaultWidth {
		t.Error("по умолчанию размытие мягкое, а у остальных инструментов прежняя толщина")
	}
	// Вне границ уровень подрезается, а не ломает режимы и не выходит за таблицы.
	if ModeOf(-5) != BlurSoft || ModeOf(500) != BlurMosaic {
		t.Error("уровень вне 1-6 должен подрезаться")
	}
	if gaussSigma(-5) != softSigma[0] || gaussSigma(500) != softSigma[softLevels-1] || mosaicBlock(-5) != mosaicSizes[0] || mosaicBlock(500) != mosaicSizes[len(mosaicSizes)-1] {
		t.Error("параметры за границами уровней должны подрезаться")
	}
	if MaxWidthOf(ToolBlur) != MaxBlurLevel || MaxWidthOf(ToolText) != MaxWidth {
		t.Error("наибольшая толщина: у размытия последний уровень, у остальных MaxWidth")
	}

	// Внутри режима сила растёт с уровнем: сигма и сторона квадрата строго возрастают, а мозаика
	// крупнее самого слабого размытия по размеру деталей (сторона больше сигмы).
	for l := float32(2); l <= softLevels; l++ {
		if gaussSigma(l) <= gaussSigma(l-1) {
			t.Errorf("сигма не растёт на уровне %v", l)
		}
	}
	for l := float32(softLevels + 2); l <= MaxBlurLevel; l++ {
		if mosaicBlock(l) <= mosaicBlock(l-1) {
			t.Errorf("сторона мозаики не растёт на уровне %v", l)
		}
	}
	if mosaicBlock(softLevels+1) < 8 {
		t.Error("клетки мозаики мельче 8 px не прячут обычный текст")
	}
	for _, m := range []BlurMode{BlurSoft, BlurMosaic} {
		if m.Label() == "" {
			t.Errorf("у режима %d нет названия", m)
		}
	}
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func TestGaussBlur(t *testing.T) {
	// Однотонный кадр остаётся прежним, в том числе у краёв: края продолжаются, а не темнеют.
	flat := solid(60, 40, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	out := gaussBlur(flat, 6)
	for _, p := range []image.Point{{0, 0}, {59, 39}, {30, 20}, {0, 39}} {
		if c := out.RGBAAt(p.X, p.Y); absDiff(c.R, 200) > 1 || absDiff(c.G, 100) > 1 || absDiff(c.B, 50) > 1 || c.A != 255 {
			t.Errorf("однотонный кадр изменился в %v: %v", p, c)
		}
	}

	// Мелкие полосы сглаживаются в ровный серый, средняя яркость сохраняется.
	stripes := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			v := uint8(0)
			if (x/2)%2 == 0 {
				v = 255
			}
			stripes.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	out = gaussBlur(stripes, 5)
	lo, hi := 255, 0
	for x := 20; x < 60; x++ {
		v := int(out.RGBAAt(x, 30).R)
		lo, hi = min(lo, v), max(hi, v)
	}
	if hi-lo > 12 || lo < 110 || hi > 145 {
		t.Errorf("полосы не сглажены до серого: от %d до %d", lo, hi)
	}

	// Средняя яркость не меняется: на шуме внутри кадра (вдали от краёв) среднее до и после совпадает.
	// Если остаток от деления отбрасывать, а не округлять, за шесть проходов кадр темнеет на два-три уровня.
	noise := image.NewRGBA(image.Rect(0, 0, 200, 200))
	seed := uint32(12345)
	for i := 0; i < len(noise.Pix); i += 4 {
		seed = seed*1664525 + 1013904223
		v := uint8(seed >> 24)
		noise.Pix[i], noise.Pix[i+1], noise.Pix[i+2], noise.Pix[i+3] = v, v, v, 255
	}
	nout := gaussBlur(noise, 6)
	var before, after int
	for y := 50; y < 150; y++ {
		for x := 50; x < 150; x++ {
			before += int(noise.RGBAAt(x, y).R)
			after += int(nout.RGBAAt(x, y).R)
		}
	}
	if d := float64(after-before) / 10000; d < -0.4 || d > 0.4 {
		t.Errorf("средняя яркость сдвинулась на %.2f уровня", d)
	}

	// Светлый квадрат 21×21 на чёрном превращается в симметричное пятно с пиком в центре.
	blk := solid(101, 101, color.RGBA{A: 255})
	for y := 40; y < 61; y++ {
		for x := 40; x < 61; x++ {
			blk.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	out = gaussBlur(blk, 4)
	peak := out.RGBAAt(50, 50).R
	if peak < 195 {
		t.Errorf("в центре квадрата яркость должна остаться почти прежней: %d", peak)
	}
	for _, d := range []int{12, 14, 16} {
		l, r, u, dn := out.RGBAAt(50-d, 50).R, out.RGBAAt(50+d, 50).R, out.RGBAAt(50, 50-d).R, out.RGBAAt(50, 50+d).R
		if absDiff(l, r) > 1 || absDiff(u, dn) > 1 || absDiff(l, u) > 2 {
			t.Errorf("пятно несимметрично на расстоянии %d: %d %d %d %d", d, l, r, u, dn)
		}
	}
	if out.RGBAAt(30, 50).R >= out.RGBAAt(36, 50).R || out.RGBAAt(36, 50).R >= out.RGBAAt(40, 50).R {
		t.Error("яркость должна монотонно убывать от края квадрата наружу")
	}

	// Сильнее сигма, шире пятно: за краем квадрата яркость растёт.
	wide := gaussBlur(blk, 9)
	if wide.RGBAAt(32, 50).R <= out.RGBAAt(32, 50).R {
		t.Errorf("с ростом сигмы пятно должно расплываться шире: %d и %d", wide.RGBAAt(32, 50).R, out.RGBAAt(32, 50).R)
	}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func TestBoxRadiiApproximateGauss(t *testing.T) {
	// Три коробки с этими радиусами дают дисперсию сигма²: сумма (w²-1)/12 по трём окнам.
	for _, sigma := range []float64{2.2, 5, 11.2, 20, 27.5} {
		var variance float64
		for _, r := range boxRadii(sigma, 3) {
			w := float64(2*r + 1)
			variance += (w*w - 1) / 12
		}
		if got := math.Sqrt(variance); math.Abs(got-sigma) > sigma*0.12+0.3 {
			t.Errorf("сигма %v: коробки дают %.2f", sigma, got)
		}
	}
}

func TestMosaicCells(t *testing.T) {
	// Кадр 50×30: градиент по X. Область начинается не с нуля: клетки считаются от её угла.
	src := image.NewRGBA(image.Rect(0, 0, 50, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 50; x++ {
			src.SetRGBA(x, y, color.RGBA{R: uint8(x * 5), G: uint8(y * 8), B: 7, A: 255})
		}
	}
	rect := image.Rect(7, 3, 40, 22) // 33×19
	got := downsample(src, rect, 10)
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 2 {
		t.Fatalf("клеток %v, ожидали 4×2", got.Bounds().Size())
	}
	// Первая клетка: пиксели (7..16, 3..12), среднее R = 5*(7+16)/2 = 57.5, G = 8*(3+12)/2 = 60.
	c := got.RGBAAt(0, 0)
	if absDiff(c.R, 57) > 2 || absDiff(c.G, 60) > 1 || c.B != 7 || c.A != 255 {
		t.Errorf("первая клетка %v", c)
	}
	// Последняя клетка короткая (3 px по X: 37..39, 9 по Y: 13..21): усредняет только то, что есть.
	c = got.RGBAAt(3, 1)
	if absDiff(c.R, 190) > 2 || absDiff(c.G, 136) > 1 { // R = 5*38 = 190, G = 8*(13+21)/2 = 136
		t.Errorf("последняя клетка %v", c)
	}
	// Область за кадром подрезается, пустая даёт заглушку, а не панику.
	if g := downsample(src, image.Rect(45, 25, 100, 100), 10); g.Bounds().Dx() != 1 || g.Bounds().Dy() != 1 {
		t.Errorf("область за краем: %v", g.Bounds())
	}
	if g := downsample(src, image.Rect(60, 60, 70, 70), 10); g.Bounds().Empty() {
		t.Error("область вне кадра должна давать непустую заглушку")
	}
}

// Размытие целого кадра FullHD укладывается в разумное время: оно пересчитывается при смене силы.
func TestGaussBlurIsFast(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	for i := range src.Pix {
		src.Pix[i] = uint8(i * 7)
	}
	start := time.Now()
	gaussBlur(src, 11)
	took := time.Since(start)
	t.Logf("гаусс сигма 11 на 1920×1080: %s", took.Round(time.Millisecond))
	if took > 2*time.Second {
		t.Errorf("размытие кадра заняло %s", took)
	}
}

func TestBlurrerCacheIsBounded(t *testing.T) {
	b := NewBlurrer(solid(100, 100, color.RGBA{R: 10, A: 255}))
	for i := 0; i < maxCached*3; i++ {
		b.mosaic(image.Rect(0, 0, 20+i, 20), 6)
	}
	if len(b.cache) > maxCached || len(b.order) > maxCached {
		t.Errorf("в кэше %d копий, предел %d", len(b.cache), maxCached)
	}
	// Повторный запрос берёт готовое, а не добавляет второе.
	n := len(b.cache)
	b.mosaic(image.Rect(0, 0, 20+maxCached*3-1, 20), 6)
	if len(b.cache) != n {
		t.Error("повторный запрос той же мозаики добавил запись")
	}
}
