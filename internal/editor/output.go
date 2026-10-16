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

// fileBytes готовит PNG к записи в файл: в формате из настроек.
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

// writeUnique записывает файл, не затирая существующий: при совпадении имени
// (два снимка в одну секунду) к имени добавляется -2, -3 и так далее.
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

// saveFile пишет снимок (PNG на входе) в каталог скриншотов в выбранном формате
// и возвращает путь.
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

// saveNumbered пишет снимок (PNG на входе) в каталог скриншотов под именем с номером в наборе.
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
