package screencast

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/url"
	"os"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/jexme/scrawl/internal/i18n"
)

const screenshotMethod = "org.freedesktop.portal.Screenshot.Screenshot"

// screenshot снимает весь рабочий стол через xdg-desktop-portal Screenshot.
// Медленнее PipeWire (около полусекунды), зато не зависит от перерисовки экрана.
func (p *Portal) screenshot() (*image.RGBA, error) {
	res, err := p.request(screenshotMethod, 30*time.Second, []any{""}, map[string]dbus.Variant{
		"interactive": dbus.MakeVariant(false),
	})
	if err != nil {
		return nil, err
	}
	uri, _ := res["uri"].Value().(string)
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return nil, fmt.Errorf(i18n.T("unexpected screenshot uri: %q"), uri)
	}
	// Файл создан порталом ради этого запроса, после чтения он не нужен.
	defer os.Remove(u.Path)

	f, err := os.Open(u.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	rgba := image.NewRGBA(src.Bounds())
	draw.Draw(rgba, rgba.Bounds(), src, src.Bounds().Min, draw.Src)
	return rgba, nil
}

// grabViaScreenshot режет снимок рабочего стола по мониторам сессий.
// Позиции мониторов считаются совпадающими с пикселями снимка (масштаб 1).
func (p *Portal) grabViaScreenshot(sessions []*Session) ([]Frame, error) {
	shot, err := p.screenshot()
	if err != nil {
		return nil, err
	}
	var frames []Frame
	for _, s := range sessions {
		for _, st := range s.streams {
			x, y, _ := pair(st.Props, "position")
			w, h, ok := pair(st.Props, "size")
			if !ok {
				return nil, fmt.Errorf(i18n.T("the portal did not report the size of stream %d"), st.ID)
			}
			r := image.Rect(x, y, x+w, y+h)
			if !r.In(shot.Bounds()) {
				return nil, fmt.Errorf(i18n.T("monitor %v is outside the screenshot %v (screen scale not 1?)"), r, shot.Bounds())
			}
			img := image.NewRGBA(image.Rect(0, 0, w, h))
			draw.Draw(img, img.Bounds(), shot, r.Min, draw.Src)
			frames = append(frames, Frame{NodeID: st.ID, X: x, Y: y, Img: img})
		}
	}
	return frames, nil
}
