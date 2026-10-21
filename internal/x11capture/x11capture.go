// Package x11capture снимает экран напрямую из X11 (сеанс GNOME на Xorg, KDE,
// XFCE и другие): без портала, диалогов и токенов. На Wayland не работает: там
// X-сервер показывает только окна XWayland.
package x11capture

import (
	"errors"
	"fmt"
	"image"
	"os"
	"sort"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"

	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/screencast"
)

// IsSession сообщает, что программа запущена в сеансе X11: есть DISPLAY и нет
// WAYLAND_DISPLAY. Под Wayland с XWayland DISPLAY тоже задан, поэтому смотрим
// на WAYLAND_DISPLAY.
func IsSession() bool {
	return os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("DISPLAY") != ""
}

// Monitor — прямоугольник монитора в координатах корневого окна.
type Monitor struct {
	Name    string
	Rect    image.Rectangle
	Primary bool
}

// Monitors возвращает мониторы по данным RandR (1.5), слева направо и сверху вниз.
// Если RandR недоступен, возвращает один «монитор» во весь экран.
func Monitors(c *xgb.Conn) ([]Monitor, error) {
	root := xproto.Setup(c).DefaultScreen(c).Root
	if err := randr.Init(c); err == nil {
		if reply, err := randr.GetMonitors(c, root, true).Reply(); err == nil && len(reply.Monitors) > 0 {
			var out []Monitor
			for _, m := range reply.Monitors {
				if m.Width == 0 || m.Height == 0 {
					continue
				}
				name := ""
				if a, err := xproto.GetAtomName(c, m.Name).Reply(); err == nil {
					name = a.Name
				}
				out = append(out, Monitor{
					Name: name, Primary: m.Primary,
					Rect: image.Rect(int(m.X), int(m.Y), int(m.X)+int(m.Width), int(m.Y)+int(m.Height)),
				})
			}
			out = dropOverlaps(out)
			if len(out) > 0 {
				sort.Slice(out, func(i, j int) bool {
					if out[i].Rect.Min.X != out[j].Rect.Min.X {
						return out[i].Rect.Min.X < out[j].Rect.Min.X
					}
					return out[i].Rect.Min.Y < out[j].Rect.Min.Y
				})
				return out, nil
			}
		}
	}
	s := xproto.Setup(c).DefaultScreen(c)
	return []Monitor{{Rect: image.Rect(0, 0, int(s.WidthInPixels), int(s.HeightInPixels)), Primary: true}}, nil
}

// pixelFormat проверяет, что корневое окно отдаёт 32 бита на пиксель с обычным
// порядком каналов (синий, зелёный, красный, пустой байт).
func pixelFormat(c *xgb.Conn) error {
	setup := xproto.Setup(c)
	screen := setup.DefaultScreen(c)
	bpp := 0
	for _, f := range setup.PixmapFormats {
		if f.Depth == screen.RootDepth {
			bpp = int(f.BitsPerPixel)
		}
	}
	if bpp != 32 || setup.ImageByteOrder != xproto.ImageOrderLSBFirst {
		return fmt.Errorf(i18n.T("unsupported X11 pixel format (depth %d, %d bits per pixel)"), screen.RootDepth, bpp)
	}
	for _, d := range screen.AllowedDepths {
		for _, v := range d.Visuals {
			if v.VisualId == screen.RootVisual && (v.RedMask != 0xff0000 || v.GreenMask != 0x00ff00 || v.BlueMask != 0x0000ff) {
				return errors.New(i18n.T("unusual X11 colour masks"))
			}
		}
	}
	return nil
}

// GrabAll снимает все мониторы и возвращает по кадру на монитор. Кадры идут в том
// же виде, что и от портала: Frame.X и Frame.Y это положение монитора.
func GrabAll() ([]screencast.Frame, error) {
	c, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf(i18n.T("could not connect to the X server: %w"), err)
	}
	defer c.Close()
	if err := pixelFormat(c); err != nil {
		return nil, err
	}
	mons, err := Monitors(c)
	if err != nil {
		return nil, err
	}
	root := xproto.Setup(c).DefaultScreen(c).Root

	// Запросы уходят все сразу, ответы читаются потом: мониторы снимаются параллельно на стороне сервера.
	cookies := make([]xproto.GetImageCookie, len(mons))
	for i, m := range mons {
		cookies[i] = xproto.GetImage(c, xproto.ImageFormatZPixmap, xproto.Drawable(root),
			int16(m.Rect.Min.X), int16(m.Rect.Min.Y), uint16(m.Rect.Dx()), uint16(m.Rect.Dy()), 0xffffffff)
	}
	frames := make([]screencast.Frame, len(mons))
	for i, m := range mons {
		reply, err := cookies[i].Reply()
		if err != nil {
			return nil, fmt.Errorf(i18n.T("capturing monitor %d: %w"), i+1, err)
		}
		frames[i] = screencast.Frame{Img: toRGBA(reply.Data, m.Rect.Dx(), m.Rect.Dy()), X: m.Rect.Min.X, Y: m.Rect.Min.Y, NodeID: uint32(i + 1)}
	}
	return frames, nil
}

// toRGBA переводит пиксели X11 (байты B, G, R, пустой) в RGBA с полной непрозрачностью.
func toRGBA(data []byte, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	n := min(len(data)/4, w*h)
	for i := 0; i < n; i++ {
		img.Pix[i*4+0] = data[i*4+2]
		img.Pix[i*4+1] = data[i*4+1]
		img.Pix[i*4+2] = data[i*4+0]
		img.Pix[i*4+3] = 0xff
	}
	return img
}

// dropOverlaps убирает мониторы, которые только дублируют другие: с одинаковым
// прямоугольником (клонированные экраны) и те, что целиком накрывают другой
// монитор (общий «экран» поверх монитора, добавленного вручную через RandR).
func dropOverlaps(mons []Monitor) []Monitor {
	var out []Monitor
	for i, m := range mons {
		drop := false
		for j, o := range mons {
			if i == j {
				continue
			}
			same := m.Rect == o.Rect
			covers := o.Rect != m.Rect && o.Rect.In(m.Rect)
			if (same && j < i) || covers {
				drop = true
			}
		}
		if !drop {
			out = append(out, m)
		}
	}
	return out
}
