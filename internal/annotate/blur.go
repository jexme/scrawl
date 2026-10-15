package annotate

import (
	"image"
	"runtime"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
)

// Env — всё, что нужно фигурам для отрисовки помимо самих координат.
type Env struct {
	Shaper *text.Shaper
	Blur   *Blurrer
}

// BlockSize переводит толщину инструмента в размер блока размытия в пикселях:
// чем толще, тем сильнее размазывание. На толщине по умолчанию (4) это 14 px,
// чего достаточно, чтобы текст обычного размера стал нечитаемым.
func BlockSize(width float32) int { return 6 + int(width)*2 }

// Blurrer готовит размытые копии кадра. Размытие делается уменьшением кадра в
// block раз (усреднение блоков) и растяжением обратно с линейным сглаживанием:
// исходные пиксели из результата восстановить нельзя, а стоит это один проход
// по кадру. Не потокобезопасен.
type Blurrer struct {
	src   *image.RGBA
	cache map[int]paint.ImageOp
}

// NewBlurrer создаёт размыватель для кадра src.
func NewBlurrer(src *image.RGBA) *Blurrer {
	return &Blurrer{src: src, cache: make(map[int]paint.ImageOp)}
}

func (b *Blurrer) op(block int) paint.ImageOp {
	if o, ok := b.cache[block]; ok {
		return o
	}
	o := paint.NewImageOp(downsample(b.src, block))
	b.cache[block] = o
	return o
}

// downsample усредняет блоки block×block; края учитываются по фактическому числу пикселей.
func downsample(src *image.RGBA, block int) *image.RGBA {
	sb := src.Bounds()
	w, h := sb.Dx(), sb.Dy()
	bw, bh := (w+block-1)/block, (h+block-1)/block
	dst := image.NewRGBA(image.Rect(0, 0, bw, bh))

	workers := runtime.NumCPU()
	if workers > bh {
		workers = bh
	}
	var wg sync.WaitGroup
	rows := make(chan int, bh)
	for y := 0; y < bh; y++ {
		rows <- y
	}
	close(rows)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for by := range rows {
				y0, y1 := by*block, min((by+1)*block, h)
				for bx := 0; bx < bw; bx++ {
					x0, x1 := bx*block, min((bx+1)*block, w)
					var r, g, bl, n uint32
					for y := y0; y < y1; y++ {
						row := src.Pix[(y+sb.Min.Y-src.Rect.Min.Y)*src.Stride+(x0+sb.Min.X-src.Rect.Min.X)*4:]
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
		}()
	}
	wg.Wait()
	return boxBlur3(dst)
}

// boxBlur3 сглаживает уменьшенное изображение окном 3×3. Изображение крошечное,
// поэтому проход дешёвый, а размазывание получается ровнее и убирает муар от
// несовпадения периода деталей с размером блока.
func boxBlur3(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(b)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, bl, n int
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					xx, yy := x+dx, y+dy
					if xx < 0 || yy < 0 || xx >= w || yy >= h {
						continue
					}
					o := src.PixOffset(xx, yy)
					r += int(src.Pix[o])
					g += int(src.Pix[o+1])
					bl += int(src.Pix[o+2])
					n++
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] = byte(r/n), byte(g/n), byte(bl/n), 255
		}
	}
	return dst
}

// drawBlur закрашивает прямоугольник фигуры размытой копией кадра.
func (s Shape) drawBlur(ops *op.Ops, env Env) {
	if len(s.Pts) < 2 || env.Blur == nil {
		return
	}
	r := RectOf(s.Pts[0], s.Pts[1])
	if r.Empty() {
		return
	}
	block := BlockSize(s.Width)
	img := env.Blur.op(block)

	defer clip.Rect(r).Push(ops).Pop()
	defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(float32(block), float32(block)))).Push(ops).Pop()
	img.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// RectOf возвращает целочисленный прямоугольник по двум противоположным углам.
func RectOf(a, b f32.Point) image.Rectangle {
	return image.Rect(
		int(min(a.X, b.X)), int(min(a.Y, b.Y)),
		int(max(a.X, b.X)+0.5), int(max(a.Y, b.Y)+0.5),
	)
}
