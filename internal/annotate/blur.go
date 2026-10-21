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

// Env is everything the shapes need for drawing besides their own coordinates.
type Env struct {
	Shaper *text.Shaper
	Blur   *Blurrer
}

// BlurMode is the blur method. It depends on the strength level: weak levels blur with a
// Gaussian and leave the image recognizable, strong ones turn the area into a mosaic from
// which text cannot be recovered.
type BlurMode int

const (
	BlurSoft   BlurMode = iota // levels 1-3: Gaussian blur
	BlurMosaic                 // levels 4-6: mosaic of squares
)

// Strength levels: for blur, the "width" is a level from 1 to MaxBlurLevel, not pixels.
const (
	DefaultBlurLevel = 2 // soft blur after which ordinary text cannot be read
	MaxBlurLevel     = 6
	softLevels       = 3 // lower levels blur with a Gaussian, the rest are a mosaic
)

// Gaussian blur sigma for levels 1-3 and mosaic square size for levels 4-6, in pixels.
var (
	softSigma   = [softLevels]float64{3, 6, 10}
	mosaicSizes = [MaxBlurLevel - softLevels]int{8, 14, 24}
)

// ModeOf returns the blur mode for strength level width.
func ModeOf(width float32) BlurMode {
	if blurLevel(width) <= softLevels {
		return BlurSoft
	}
	return BlurMosaic
}

// Label returns the mode name for the badge showing the strength value.
func (m BlurMode) Label() string {
	if m == BlurMosaic {
		return i18n.T("Mosaic")
	}
	return i18n.T("Blur")
}

// blurLevel rounds the strength to a level 1..MaxBlurLevel.
func blurLevel(width float32) int {
	return min(max(int(math.Round(float64(width))), 1), MaxBlurLevel)
}

// gaussSigma returns the Gaussian blur sigma for level width.
func gaussSigma(width float32) float64 { return softSigma[min(blurLevel(width), softLevels)-1] }

// mosaicBlock returns the mosaic square side in pixels for level width.
func mosaicBlock(width float32) int {
	return mosaicSizes[max(blurLevel(width), softLevels+1)-softLevels-1]
}

// Blurrer prepares processed copies of the frame: blurred and mosaic. A copy is computed once
// per strength and kept until newer ones evict it. Not safe for concurrent use.
type Blurrer struct {
	src   *image.RGBA
	cache map[blurKey]paint.ImageOp
	order []blurKey // order of appearance in the cache: the oldest are evicted
}

type blurKey struct {
	mode  BlurMode
	param int             // sigma in tenths of a pixel or the square side
	rect  image.Rectangle // mosaic area: cells are counted from its corner; empty for blur
}

// maxCached is how many processed copies to keep: a blurred copy is a whole frame in
// memory and on the GPU (there are only three of them), a mosaic covers only its own area,
// but while the frame is dragged a new one appears every frame.
const maxCached = 24

// NewBlurrer creates a blurrer for frame src.
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

// gauss returns the whole frame blurred with a Gaussian of sigma sigma.
func (b *Blurrer) gauss(sigma float64) paint.ImageOp {
	return b.get(blurKey{mode: BlurSoft, param: int(math.Round(sigma * 10))}, func() paint.ImageOp {
		return paint.NewImageOp(gaussBlur(b.src, sigma))
	})
}

// mosaic returns the mosaic of area rect: block×block cells are counted from its top-left
// corner, so all of them are whole and content outside the area does not get into them. It
// is scaled back up without smoothing, which gives crisp squares.
func (b *Blurrer) mosaic(rect image.Rectangle, block int) paint.ImageOp {
	return b.get(blurKey{mode: BlurMosaic, param: block, rect: rect}, func() paint.ImageOp {
		o := paint.NewImageOp(downsample(b.src, rect, block))
		o.Filter = paint.FilterNearest
		return o
	})
}

// downsample averages block×block cells inside rect, starting from its top-left corner;
// the last cells at the right and bottom edges average only what is there.
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

// parallel splits n rows (or columns) among all cores.
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

// gaussBlur blurs the frame with a Gaussian. Three passes of a "box" blur along each axis give
// a Gaussian accurate to the eye, and cost the same for any sigma: fast even at 4K.
// The frame edges are extended with the edge pixels.
func gaussBlur(src *image.RGBA, sigma float64) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	// Working copies with an even row stride; the result is in the same form.
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

// boxRadii picks the radii of n box passes that give a Gaussian with sigma sigma.
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

// boxH averages each row in [lo, hi) with a window of width 2r+1, the edges are extended
// with the edge pixels.
func boxH(src, dst []byte, w, stride, r, lo, hi int) {
	div := 2*r + 1
	half := div / 2 // rounding rather than truncation: otherwise the frame darkens over six passes
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

// boxV does the same vertically for the columns in [lo, hi). It goes by rows, not by
// columns, so that memory is read sequentially.
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

// drawBlur paints the shape's rectangle with a processed copy of the frame: blurred, mosaic or
// frosted, depending on the strength.
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
		// The cells are scaled up by block times and placed at the top-left corner of the area.
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

// RectOf returns an integer rectangle from two opposite corners.
func RectOf(a, b f32.Point) image.Rectangle {
	return image.Rect(
		int(min(a.X, b.X)), int(min(a.Y, b.Y)),
		int(max(a.X, b.X)+0.5), int(max(a.Y, b.Y)+0.5),
	)
}
