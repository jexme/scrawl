package tray

import (
	"image"
	"image/color"
)

// pixmap — картинка значка для StatusNotifierItem: ширина, высота и байты ARGB32
// в сетевом порядке (альфа первой).
type pixmap struct {
	W, H int32
	Data []byte
}

// drawIcon рисует значок: белые уголки видоискателя и точка-курсор в центре.
// Значок рисуется кодом, чтобы не зависеть от темы значков и файлов ресурсов.
func drawIcon(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	th := max(size/10, 2) // толщина линий
	arm := size / 4       // длина уголка: между уголками остаётся просвет
	m := size / 8         // отступ от края
	fill := func(x0, y0, x1, y1 int) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetNRGBA(x, y, white)
			}
		}
	}
	lo, hi := m, size-m
	for _, c := range [][2]int{{lo, lo}, {hi, lo}, {lo, hi}, {hi, hi}} {
		dx, dy := 1, 1
		if c[0] == hi {
			dx = -1
		}
		if c[1] == hi {
			dy = -1
		}
		// Горизонтальная и вертикальная стороны уголка.
		x0, x1 := c[0], c[0]+dx*arm
		if dx < 0 {
			x0, x1 = x1, c[0]
		}
		y := c[1]
		if dy < 0 {
			y -= th
		}
		fill(x0, y, x1, y+th)
		y0, y1 := c[1], c[1]+dy*arm
		if dy < 0 {
			y0, y1 = y1, c[1]
		}
		x := c[0]
		if dx < 0 {
			x -= th
		}
		fill(x, y0, x+th, y1)
	}
	r := max(size/10, 2)
	cx, cy := size/2, size/2
	fill(cx-r, cy-r, cx+r, cy+r)
	return img
}

func toPixmap(img image.Image) pixmap {
	b := img.Bounds()
	data := make([]byte, 0, b.Dx()*b.Dy()*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			data = append(data, c.A, c.R, c.G, c.B)
		}
	}
	return pixmap{W: int32(b.Dx()), H: int32(b.Dy()), Data: data}
}
