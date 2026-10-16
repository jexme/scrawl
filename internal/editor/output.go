package editor

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"

	"github.com/jexme/screen-editor/internal/annotate"
	"github.com/jexme/screen-editor/internal/settings"
)

// render рисует выделенную область кадра вместе с аннотациями тем же кодом
// Gio, что и экран, поэтому результат совпадает с тем, что видел пользователь.
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

// saveFile пишет PNG в каталог скриншотов и возвращает путь.
func saveFile(cfg settings.Settings, data []byte) (string, error) {
	dir, err := cfg.OutputDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, cfg.FileName(time.Now().Format(settings.StampLayout)))
	return path, os.WriteFile(path, data, 0o644)
}

// saveNumbered пишет PNG в каталог скриншотов под именем с номером в наборе.
func saveNumbered(cfg settings.Settings, data []byte, stamp string, n int) (string, error) {
	dir, err := cfg.OutputDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, cfg.NumberedFileName(stamp, n))
	return path, os.WriteFile(path, data, 0o644)
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// inform показывает уведомление об успешном действии, если оно не отключено в
// настройках. Ошибки и подсказки идут через notify и показываются всегда.
func (m *monitor) inform(title, body string) {
	if m.store.Get().Notifications {
		notify(title, body)
	}
}

// notify показывает уведомление, если есть notify-send. Ошибки не важны.
func notify(title, body string) {
	exec.Command("notify-send", "-a", "Screen Editor", title, body).Run()
}
