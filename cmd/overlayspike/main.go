// overlayspike показывает кадр из живой ScreenCast-сессии в fullscreen-окне Gio
// и даёт выделить область мышью. Печатает время до появления окна и картинки.
//
// Esc закрывает окно, отпускание кнопки мыши печатает выделение и выходит.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/jexme/screen-editor/internal/screencast"
)

func main() {
	reset := flag.Bool("reset", false, "игнорировать сохранённый restore_token (покажет диалог)")
	flag.Parse()

	p, err := screencast.New()
	if err != nil {
		fatal("portal: %v", err)
	}
	if !p.Registered {
		fmt.Fprintln(os.Stderr, "portal не принял app_id: restore_token работать не будет, диалог покажется при каждом запуске")
	}
	token := ""
	if !*reset {
		token = screencast.LoadToken()
	}
	if token == "" {
		fmt.Println(">>> restore_token нет: GNOME покажет диалог выбора экрана, выберите нужные мониторы и подтвердите")
	}

	// Старт сессии в замер не входит: в реальном приложении её держит демон.
	s, newToken, err := p.NewSession(token)
	if err != nil {
		fatal("сессия: %v", err)
	}
	if newToken != "" {
		if err := screencast.SaveToken(newToken); err != nil {
			fmt.Fprintf(os.Stderr, "restore_token не сохранён: %v\n", err)
		}
	}
	fmt.Printf("сессия готова, мониторов в ней: %d\n", s.Streams())

	t0 := time.Now() // момент «нажатия хоткея»
	go func() {
		run(s, t0)
		s.Close()
		os.Exit(0)
	}()
	app.Main()
}

func run(s *screencast.Session, t0 time.Time) {
	w := new(app.Window)
	w.Option(app.Title("screen-editor"), app.Fullscreen.Option())

	var frames atomic.Pointer[[]screencast.Frame]
	var tGrab atomic.Int64
	go func() {
		fr, err := s.Grab()
		if err != nil {
			fatal("кадр: %v", err)
		}
		tGrab.Store(int64(time.Since(t0)))
		frames.Store(&fr)
		w.Invalidate()
	}()

	var (
		ops        op.Ops
		tag        = new(int)
		focused    bool
		chosen     *screencast.Frame
		imageOp    paint.ImageOp
		selecting  bool
		start, cur f32.Point
		tWin       time.Duration
		reported   bool
	)

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if tWin == 0 {
				tWin = time.Since(t0)
			}

			// Выбираем кадр монитора, который по пропорциям совпадает с окном.
			if chosen == nil {
				if fr := frames.Load(); fr != nil {
					chosen = pick(*fr, e.Size)
					// Окно открылось на другом мониторе, чем снят кадр.
					if !sameAspect(chosen.Img.Bounds(), e.Size) {
						notify("Скриншот отменён", "Нажмите хоткей, когда курсор на главном мониторе")
						return
					}
					imageOp = paint.NewImageOp(chosen.Img)
				}
			}

			// Ввод.
			area := clip.Rect(image.Rectangle{Max: e.Size}).Push(gtx.Ops)
			event.Op(gtx.Ops, tag)
			pointer.CursorCrosshair.Add(gtx.Ops)
			area.Pop()
			if !focused {
				gtx.Execute(key.FocusCmd{Tag: tag})
				focused = true
			}
			done := false
			for {
				ev, ok := gtx.Event(
					pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Drag | pointer.Release},
					key.Filter{Focus: tag, Name: key.NameEscape},
				)
				if !ok {
					break
				}
				switch ev := ev.(type) {
				case pointer.Event:
					switch ev.Kind {
					case pointer.Press:
						selecting, start, cur = true, ev.Position, ev.Position
					case pointer.Drag:
						cur = ev.Position
					case pointer.Release:
						cur = ev.Position
						if selecting && chosen != nil {
							r := selection(chosen, e.Size, start, cur)
							// Простой клик без выделения не считаем скриншотом.
							if r.Dx() >= 2 && r.Dy() >= 2 {
								if err := save(chosen.Img, r); err != nil {
									fmt.Fprintf(os.Stderr, "сохранение: %v\n", err)
									notify("Скриншот не сохранён", err.Error())
								}
								done = true
							}
						}
						selecting = false
					}
				case key.Event:
					if ev.State == key.Press {
						done = true
					}
				}
			}
			if done {
				return
			}

			// Отрисовка.
			paint.Fill(gtx.Ops, color.NRGBA{A: 255})
			if chosen != nil {
				draw(gtx.Ops, imageOp, chosen.Img.Bounds(), e.Size)
				dim(gtx.Ops, e.Size, selecting, start, cur)
			}
			e.Frame(gtx.Ops)

			if chosen != nil && !reported {
				reported = true
				fmt.Printf("окно появилось: %s | кадр захвачен: %s | картинка на экране: %s (от «хоткея»)\n",
					ms(tWin), ms(time.Duration(tGrab.Load())), ms(time.Since(t0)))
				fmt.Printf("окно %dx%d, кадр %dx%d\n", e.Size.X, e.Size.Y, chosen.Img.Bounds().Dx(), chosen.Img.Bounds().Dy())
			}
		}
	}
}

// pick выбирает кадр, чьи пропорции ближе всего к размеру окна.
func pick(frames []screencast.Frame, win image.Point) *screencast.Frame {
	wr := float64(win.X) / float64(win.Y)
	best, bestDiff := 0, math.MaxFloat64
	for i, f := range frames {
		b := f.Img.Bounds()
		diff := math.Abs(float64(b.Dx())/float64(b.Dy()) - wr)
		// При равных пропорциях предпочитаем кадр, ближайший по размеру.
		diff += math.Abs(float64(b.Dx()-win.X)) * 1e-9
		if diff < bestDiff {
			best, bestDiff = i, diff
		}
	}
	return &frames[best]
}

func draw(ops *op.Ops, img paint.ImageOp, src image.Rectangle, win image.Point) {
	sx := float32(win.X) / float32(src.Dx())
	sy := float32(win.Y) / float32(src.Dy())
	defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(sx, sy))).Push(ops).Pop()
	defer clip.Rect(src).Push(ops).Pop()
	img.Add(ops)
	paint.PaintOp{}.Add(ops)
}

// dim затемняет всё, кроме выделенной области, и рисует её рамку.
func dim(ops *op.Ops, win image.Point, selecting bool, a, b f32.Point) {
	shade := color.NRGBA{A: 110}
	if !selecting {
		paint.FillShape(ops, shade, clip.Rect(image.Rectangle{Max: win}).Op())
		return
	}
	r := rectOf(a, b)
	for _, side := range []image.Rectangle{
		{Min: image.Pt(0, 0), Max: image.Pt(win.X, r.Min.Y)},
		{Min: image.Pt(0, r.Max.Y), Max: win},
		{Min: image.Pt(0, r.Min.Y), Max: image.Pt(r.Min.X, r.Max.Y)},
		{Min: image.Pt(r.Max.X, r.Min.Y), Max: image.Pt(win.X, r.Max.Y)},
	} {
		paint.FillShape(ops, shade, clip.Rect(side).Op())
	}
	paint.FillShape(ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		clip.Stroke{Path: clip.Rect(r).Path(), Width: 2}.Op())
}

func rectOf(a, b f32.Point) image.Rectangle {
	return image.Rect(
		int(math.Min(float64(a.X), float64(b.X))), int(math.Min(float64(a.Y), float64(b.Y))),
		int(math.Max(float64(a.X), float64(b.X))), int(math.Max(float64(a.Y), float64(b.Y))),
	)
}

func sameAspect(src image.Rectangle, win image.Point) bool {
	sa := float64(src.Dx()) / float64(src.Dy())
	wa := float64(win.X) / float64(win.Y)
	return math.Abs(sa-wa) < 0.01
}

// selection переводит выделение из координат окна в пиксели кадра.
func selection(f *screencast.Frame, win image.Point, a, b f32.Point) image.Rectangle {
	r := rectOf(a, b)
	src := f.Img.Bounds()
	sx := float64(src.Dx()) / float64(win.X)
	sy := float64(src.Dy()) / float64(win.Y)
	out := image.Rect(
		int(float64(r.Min.X)*sx), int(float64(r.Min.Y)*sy),
		int(float64(r.Max.X)*sx), int(float64(r.Max.Y)*sy),
	)
	fmt.Printf("выделение в кадре: x=%d y=%d %dx%d (монитор node %d)\n",
		out.Min.X, out.Min.Y, out.Dx(), out.Dy(), f.NodeID)
	return out.Intersect(src)
}

// save вырезает область, сохраняет PNG в ~/Pictures/Screenshots и копирует в буфер обмена.
func save(img *image.RGBA, r image.Rectangle) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img.SubImage(r)); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "Pictures", "Screenshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "screen-editor-"+time.Now().Format("20060102-150405")+".png")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return err
	}

	// wl-copy сам уходит в фон и держит буфер, пока его не заменят.
	cp := exec.Command("wl-copy", "--type", "image/png")
	cp.Stdin = bytes.NewReader(buf.Bytes())
	if err := cp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wl-copy: %v\n", err)
	}
	fmt.Println("сохранено:", path)
	notify("Скриншот сохранён", path)
	return nil
}

// notify показывает уведомление, если есть notify-send. Ошибки не важны.
func notify(title, body string) {
	exec.Command("notify-send", "-a", "Screen Editor", title, body).Run()
}

func ms(d time.Duration) string { return fmt.Sprintf("%.0fms", float64(d.Microseconds())/1000) }

func fatal(f string, a ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	os.Exit(1)
}
