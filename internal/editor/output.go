package editor

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/settings"
)

// render draws the selected region of the frame together with the annotations using the
// same Gio code as the screen, so the result matches what the user saw.
func render(src *image.RGBA, sel image.Rectangle, shapes []annotate.Shape, shaper *text.Shaper) (*image.RGBA, error) {
	w, h := sel.Dx(), sel.Dy()
	win, err := headless.NewWindow(w, h)
	if err != nil {
		return nil, fmt.Errorf("headless: %w", err)
	}
	defer win.Release()

	var ops op.Ops
	shift := op.Offset(image.Pt(-sel.Min.X, -sel.Min.Y)).Push(&ops)
	clipArea := clip.Rect(src.Bounds()).Push(&ops)
	paint.NewImageOp(src).Add(&ops)
	paint.PaintOp{}.Add(&ops)
	clipArea.Pop()
	annotate.DrawAll(&ops, annotate.Env{Shaper: shaper, Blur: annotate.NewBlurrer(src)}, shapes)
	shift.Pop()

	if err := win.Frame(&ops); err != nil {
		return nil, err
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	if err := win.Screenshot(out); err != nil {
		return nil, err
	}
	return out, nil
}

// fileBytes prepares the PNG for writing to a file: in the format from settings.
func fileBytes(cfg settings.Settings, pngData []byte) ([]byte, error) {
	if cfg.Format != settings.FormatJPEG {
		return pngData, nil
	}
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("JPEG: %w", err)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: cfg.JPEGQuality}); err != nil {
		return nil, fmt.Errorf("JPEG: %w", err)
	}
	return buf.Bytes(), nil
}

// writeUnique writes a file without overwriting an existing one: on a name clash (two
// shots in the same second) -2, -3 and so on is appended to the name.
func writeUnique(path string, data []byte) (string, error) {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for n := 1; ; n++ {
		p := path
		if n > 1 {
			p = fmt.Sprintf("%s-%d%s", base, n, ext)
		}
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = f.Write(data)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return p, err
	}
}

// saveFile writes the shot (PNG on input) to the screenshot directory in the chosen format
// and returns the path.
func saveFile(cfg settings.Settings, data []byte) (string, error) {
	dir, err := cfg.OutputDir()
	if err != nil {
		return "", err
	}
	data, err = fileBytes(cfg, data)
	if err != nil {
		return "", err
	}
	return writeUnique(filepath.Join(dir, cfg.FileName(time.Now().Format(settings.StampLayout))), data)
}

// saveNumbered writes the shot (PNG on input) to the screenshot directory under a name
// with its number in the set.
func saveNumbered(cfg settings.Settings, data []byte, stamp string, n int) (string, error) {
	dir, err := cfg.OutputDir()
	if err != nil {
		return "", err
	}
	data, err = fileBytes(cfg, data)
	if err != nil {
		return "", err
	}
	return writeUnique(filepath.Join(dir, cfg.NumberedFileName(stamp, n)), data)
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// inform shows a notification about a successful action unless it is disabled in settings.
// Errors and hints go through notify and are always shown.
func (m *monitor) inform(title, body string) {
	if m.store.Get().Notifications {
		notify(title, body)
	}
}

// notify shows a notification if notify-send is available. Errors do not matter.
func notify(title, body string) {
	exec.Command("notify-send", "-a", "Scrawl", title, body).Run()
}
