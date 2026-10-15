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
	annotate.DrawAll(&ops, shaper, shapes)
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

// saveFile пишет PNG в ~/Pictures/Screenshots и возвращает путь.
func saveFile(data []byte) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Pictures", "Screenshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "screen-editor-"+time.Now().Format("20060102-150405")+".png")
	return path, os.WriteFile(path, data, 0o644)
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// notify показывает уведомление, если есть notify-send. Ошибки не важны.
func notify(title, body string) {
	exec.Command("notify-send", "-a", "Screen Editor", title, body).Run()
}
