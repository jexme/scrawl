package annotate

import (
	"image"
	"math"
	"runtime"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"

	"github.com/jexme/scrawl/internal/i18n"
)

// Env — всё, что нужно фигурам для отрисовки помимо самих координат.
type Env struct {
	Shaper *text.Shaper
	Blur   *Blurrer
}

// BlurMode — способ размытия. Он зависит от уровня силы: слабые уровни размывают по Гауссу и
// оставляют картинку узнаваемой, сильные превращают область в мозаику, из которой текст не
// восстановить.
type BlurMode int

const (
	BlurSoft   BlurMode = iota // уровни 1-3: гауссово размытие
	BlurMosaic                 // уровни 4-6: мозаика из квадратов
)

// Уровни силы: у размытия «толщина» это уровень от 1 до MaxBlurLevel, а не пиксели.
const (
	DefaultBlurLevel = 2 // мягкое размытие, после которого обычный текст не прочитать
	MaxBlurLevel     = 6
	softLevels       = 3 // уровни до этого размывают по Гауссу, остальные это мозаика
)

// Сигма гауссова размытия по уровням 1-3 и сторона квадрата мозаики по уровням 4-6, в пикселях.
var (
	softSigma   = [softLevels]float64{3, 6, 10}
	mosaicSizes = [MaxBlurLevel - softLevels]int{8, 14, 24}
)

// ModeOf возвращает режим размытия для уровня силы width.
func ModeOf(width float32) BlurMode {
	if blurLevel(width) <= softLevels {
		return BlurSoft
	}
	return BlurMosaic
}

// Label возвращает название режима для плашки со значением силы.
func (m BlurMode) Label() string {
	if m == BlurMosaic {
		return i18n.T("Mosaic")
	}
	return i18n.T("Blur")
}

// blurLevel округляет силу до уровня 1..MaxBlurLevel.
func blurLevel(width float32) int {
	return min(max(int(math.Round(float64(width))), 1), MaxBlurLevel)
}

// gaussSigma возвращает сигму гауссова размытия для уровня width.
func gaussSigma(width float32) float64 { return softSigma[min(blurLevel(width), softLevels)-1] }

// mosaicBlock возвращает сторону квадрата мозаики в пикселях для уровня width.
func mosaicBlock(width float32) int {
	return mosaicSizes[max(blurLevel(width), softLevels+1)-softLevels-1]
}

// Blurrer готовит обработанные копии кадра: размытые и мозаичные. Копия считается один раз на
// каждую силу и хранится, пока не вытеснят новые. Не потокобезопасен.
type Blurrer struct {
	src   *image.RGBA
	cache map[blurKey]paint.ImageOp
	order []blurKey // порядок появления в кэше: самые старые вытесняются
}

type blurKey struct {
	mode  BlurMode
	param int             // сигма в десятых долях пикселя или сторона квадрата
	rect  image.Rectangle // область мозаики: клетки отсчитываются от её угла; у размытия пусто
}

// maxCached — сколько обработанных копий держать: размытая копия это кадр целиком в памяти и на GPU
// (их всего три), мозаика только своя область, но при перетаскивании рамки новая появляется каждый кадр.
const maxCached = 24

// NewBlurrer создаёт размыватель для кадра src.
func NewBlurrer(src *image.RGBA) *Blurrer {
	return &Blurrer{src: src, cache: make(map[blurKey]paint.ImageOp)}
}

func (b *Blurrer) get(key blurKey, build func() paint.ImageOp) paint.ImageOp {
	if o, ok := b.cache[key]; ok {
		return o
	}
	o := build()
	if len(b.order) >= maxCached {
		delete(b.cache, b.order[0])
		b.order = b.order[1:]
	}
	b.cache[key] = o
	b.order = append(b.order, key)
	return o
}

// gauss возвращает кадр целиком, размытый по Гауссу с сигмой sigma.
func (b *Blurrer) gauss(sigma float64) paint.ImageOp {
	return b.get(blurKey{mode: BlurSoft, param: int(math.Round(sigma * 10))}, func() paint.ImageOp {
		return paint.NewImageOp(gaussBlur(b.src, sigma))
	})
}

// mosaic возвращает мозаику области rect: клетки block×block считаются от её левого верхнего угла,
// поэтому все они целые, а содержимое снаружи области в них не попадает. Растягивается обратно
// без сглаживания, получаются чёткие квадраты.
func (b *Blurrer) mosaic(rect image.Rectangle, block int) paint.ImageOp {
	return b.get(blurKey{mode: BlurMosaic, param: block, rect: rect}, func() paint.ImageOp {
		o := paint.NewImageOp(downsample(b.src, rect, block))
		o.Filter = paint.FilterNearest
		return o
	})
}

// downsample усредняет клетки block×block внутри rect, начиная с его левого верхнего угла;
// последние клетки у правого и нижнего краёв усредняют только то, что есть.
func downsample(src *image.RGBA, rect image.Rectangle, block int) *image.RGBA {
	rect = rect.Intersect(src.Bounds())
	w, h := rect.Dx(), rect.Dy()
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	bw, bh := (w+block-1)/block, (h+block-1)/block
	dst := image.NewRGBA(image.Rect(0, 0, bw, bh))

	parallel(bh, func(lo, hi int) {
		for by := lo; by < hi; by++ {
			y0, y1 := by*block, min((by+1)*block, h)
			for bx := 0; bx < bw; bx++ {
				x0, x1 := bx*block, min((bx+1)*block, w)
				var r, g, bl, n uint32
				for y := y0; y < y1; y++ {
					row := src.Pix[src.PixOffset(rect.Min.X+x0, rect.Min.Y+y):]
					for x := 0; x < x1-x0; x++ {
						r += uint32(row[x*4])
						g += uint32(row[x*4+1])
						bl += uint32(row[x*4+2])
						n++
					}
				}
				o := dst.PixOffset(bx, by)
				dst.Pix[o], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] = byte(r/n), byte(g/n), byte(bl/n), 255
			}
		}
	})
	return dst
}

// parallel делит n строк (или столбцов) между всеми ядрами.
func parallel(n int, fn func(lo, hi int)) {
	workers := min(runtime.NumCPU(), n)
	if workers < 1 {
		return
	}
	var wg sync.WaitGroup
	step := (n + workers - 1) / workers
	for lo := 0; lo < n; lo += step {
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			fn(lo, hi)
		}(lo, min(lo+step, n))
	}
	wg.Wait()
}

// gaussBlur размывает кадр по Гауссу. Три прохода «коробочным» размытием по каждой оси дают
// гаусс с точностью до глаза, а стоят столько же при любой сигме: быстро даже на 4K.
// Края кадра продолжаются крайними пикселями.
func gaussBlur(src *image.RGBA, sigma float64) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	// Рабочие копии с ровным шагом строки; результат тоже в таком виде.
	cur := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(cur.Pix[y*cur.Stride:y*cur.Stride+w*4], src.Pix[(y+b.Min.Y-src.Rect.Min.Y)*src.Stride+(b.Min.X-src.Rect.Min.X)*4:])
	}
	if w == 0 || h == 0 || sigma <= 0 {
		return cur
	}
	tmp := image.NewRGBA(cur.Rect)
	for _, r := range boxRadii(sigma, 3) {
		if r < 1 {
			continue
		}
		parallel(h, func(lo, hi int) { boxH(cur.Pix, tmp.Pix, w, cur.Stride, r, lo, hi) })
		parallel(w, func(lo, hi int) { boxV(tmp.Pix, cur.Pix, w, h, cur.Stride, r, lo, hi) })
	}
	return cur
}

// boxRadii подбирает радиусы n коробочных проходов, дающих гаусс с сигмой sigma.
func boxRadii(sigma float64, n int) []int {
	wIdeal := math.Sqrt(12*sigma*sigma/float64(n) + 1)
	wl := int(math.Floor(wIdeal))
	if wl%2 == 0 {
		wl--
	}
	wu := wl + 2
	mIdeal := (12*sigma*sigma - float64(n*wl*wl) - 4*float64(n*wl) - 3*float64(n)) / (-4*float64(wl) - 4)
	m := int(math.Round(mIdeal))
	radii := make([]int, n)
	for i := range radii {
		size := wu
		if i < m {
			size = wl
		}
		radii[i] = (size - 1) / 2
	}
	return radii
}

// boxH усредняет каждую строку из [lo, hi) окном шириной 2r+1, края продолжаются крайними пикселями.
func boxH(src, dst []byte, w, stride, r, lo, hi int) {
	div := 2*r + 1
	half := div / 2 // округление, а не отбрасывание: иначе за шесть проходов кадр темнеет
	for y := lo; y < hi; y++ {
		row := src[y*stride : y*stride+w*4]
		out := dst[y*stride : y*stride+w*4]
		var s0, s1, s2 int
		for j := -r; j <= r; j++ {
			o := clampInt(j, 0, w-1) * 4
			s0 += int(row[o])
			s1 += int(row[o+1])
			s2 += int(row[o+2])
		}
		for x := 0; x < w; x++ {
			out[x*4], out[x*4+1], out[x*4+2], out[x*4+3] = byte((s0+half)/div), byte((s1+half)/div), byte((s2+half)/div), 255
			a := clampInt(x-r, 0, w-1) * 4
			b := clampInt(x+r+1, 0, w-1) * 4
			s0 += int(row[b]) - int(row[a])
			s1 += int(row[b+1]) - int(row[a+1])
			s2 += int(row[b+2]) - int(row[a+2])
		}
	}
}

// boxV то же по вертикали для столбцов из [lo, hi). Идёт по строкам, а не по столбцам: так
// память читается подряд.
func boxV(src, dst []byte, w, h, stride, r, lo, hi int) {
	div := 2*r + 1
	half := div / 2
	n := hi - lo
	sums := make([]int, n*3)
	add := func(y, sign int) {
		row := src[clampInt(y, 0, h-1)*stride:]
		for i := 0; i < n; i++ {
			o := (lo + i) * 4
			sums[i*3] += sign * int(row[o])
			sums[i*3+1] += sign * int(row[o+1])
			sums[i*3+2] += sign * int(row[o+2])
		}
	}
	for j := -r; j <= r; j++ {
		add(j, 1)
	}
	for y := 0; y < h; y++ {
		out := dst[y*stride:]
		for i := 0; i < n; i++ {
			o := (lo + i) * 4
			out[o], out[o+1], out[o+2], out[o+3] = byte((sums[i*3]+half)/div), byte((sums[i*3+1]+half)/div), byte((sums[i*3+2]+half)/div), 255
		}
		add(y+r+1, 1)
		add(y-r, -1)
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// drawBlur закрашивает прямоугольник фигуры обработанной копией кадра: размытой, мозаичной или
// матовой, по силе.
func (s Shape) drawBlur(ops *op.Ops, env Env) {
	if len(s.Pts) < 2 || env.Blur == nil {
		return
	}
	r := RectOf(s.Pts[0], s.Pts[1])
	if r.Empty() {
		return
	}
	switch ModeOf(s.Width) {
	case BlurMosaic:
		block := mosaicBlock(s.Width)
		img := env.Blur.mosaic(r, block)
		defer clip.Rect(r).Push(ops).Pop()
		// Клетки растягиваются в block раз и ставятся в левый верхний угол области.
		defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(float32(block), float32(block))).Offset(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))).Push(ops).Pop()
		img.Add(ops)
		paint.PaintOp{}.Add(ops)
	default:
		img := env.Blur.gauss(gaussSigma(s.Width))
		defer clip.Rect(r).Push(ops).Pop()
		img.Add(ops)
		paint.PaintOp{}.Add(ops)
	}
}

// RectOf возвращает целочисленный прямоугольник по двум противоположным углам.
func RectOf(a, b f32.Point) image.Rectangle {
	return image.Rect(
		int(min(a.X, b.X)), int(min(a.Y, b.Y)),
		int(max(a.X, b.X)+0.5), int(max(a.Y, b.Y)+0.5),
	)
}
