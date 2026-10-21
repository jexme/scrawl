package tray

import (
	"image"
	"image/color"
	"image/draw"
)

// pixmap is an icon image for StatusNotifierItem: width, height and ARGB32 bytes
// in network byte order (alpha first).
type pixmap struct {
	W, H int32
	Data []byte
}

// drawIcon draws the icon: white viewfinder corners and a cursor dot in the center.
// The icon is drawn in code so as not to depend on the icon theme or resource files.
func drawIcon(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	th := max(size/10, 2) // line thickness
	arm := size / 4       // corner arm length: a gap remains between the corners
	m := size / 8         // margin from the edge
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
		// The horizontal and vertical sides of a corner.
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

// Framed draws the icon for Windows: a white viewfinder on a blue rounded square. The
// background is needed so the icon is readable on both a dark and a light taskbar. It is
// also used for the icon of the file scrawl.exe (scripts/genicon).
func Framed(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	bg := color.NRGBA{R: 0x25, G: 0x63, B: 0xEB, A: 0xFF}
	rad := size / 5
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if insideRounded(x, y, size, rad) {
				img.SetNRGBA(x, y, bg)
			}
		}
	}
	draw.Draw(img, img.Bounds(), drawIcon(size), image.Point{}, draw.Over)
	return img
}

// insideRounded reports whether a pixel lies inside an n×n square with rounded corners of
// radius rad.
func insideRounded(x, y, n, rad int) bool {
	cx, cy := x, y
	switch {
	case x < rad:
		cx = rad
	case x >= n-rad:
		cx = n - rad - 1
	}
	switch {
	case y < rad:
		cy = rad
	case y >= n-rad:
		cy = n - rad - 1
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= rad*rad
}
