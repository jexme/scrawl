package x11capture

import (
	"image"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"
)

// startXvfb starts a virtual X server (the server itself picks the display number through
// -displayfd) and returns its name like ":97". Without Xvfb the test is skipped.
func startXvfb(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("Xvfb"); err != nil {
		t.Skip("нет Xvfb")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("Xvfb", "-displayfd", "3", "-screen", "0", "1600x600x24", "-nolisten", "tcp", "-noreset")
	cmd.ExtraFiles = []*os.File{w}
	if err := cmd.Start(); err != nil {
		t.Skipf("Xvfb не запустился: %v", err)
	}
	w.Close()
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait(); r.Close() })

	line := make(chan string, 1)
	go func() {
		buf := make([]byte, 32)
		n, _ := r.Read(buf)
		line <- strings.TrimSpace(string(buf[:n]))
	}()
	select {
	case num := <-line:
		if num == "" {
			t.Skip("Xvfb не назвал номер дисплея")
		}
		display := ":" + num
		// The server announces the number before it starts accepting clients: we wait for the
		// first connection.
		for i := 0; i < 50; i++ {
			if c, err := xgb.NewConnDisplay(display); err == nil {
				c.Close()
				return display
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Skip("Xvfb не принимает подключения")
	case <-time.After(5 * time.Second):
		t.Skip("Xvfb не ответил")
	}
	return ""
}

// fill paints a rectangle of the root window.
func fill(t *testing.T, display string, x, y, w, h int, c color.RGBA) {
	t.Helper()
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	root := xproto.Setup(conn).DefaultScreen(conn).Root
	gc, _ := xproto.NewGcontextId(conn)
	pixel := uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
	if err := xproto.CreateGCChecked(conn, gc, xproto.Drawable(root), xproto.GcForeground, []uint32{pixel}).Check(); err != nil {
		t.Fatal(err)
	}
	if err := xproto.PolyFillRectangleChecked(conn, xproto.Drawable(root), gc, []xproto.Rectangle{{X: int16(x), Y: int16(y), Width: uint16(w), Height: uint16(h)}}).Check(); err != nil {
		t.Fatal(err)
	}
}

func TestGrabAllOnXvfb(t *testing.T) {
	display := startXvfb(t)
	t.Setenv("DISPLAY", display)
	t.Setenv("WAYLAND_DISPLAY", "")
	if !IsSession() {
		t.Fatal("с DISPLAY и без WAYLAND_DISPLAY это сеанс X11")
	}

	// One "monitor" for the whole screen.
	red, blue := color.RGBA{R: 220, G: 30, B: 40, A: 255}, color.RGBA{R: 20, G: 60, B: 230, A: 255}
	fill(t, display, 0, 0, 800, 600, red)
	fill(t, display, 800, 0, 800, 600, blue)
	frames, err := GrabAll()
	if err != nil || len(frames) == 0 {
		t.Fatalf("GrabAll: %v, кадров %d", err, len(frames))
	}
	if len(frames) != 1 || frames[0].Img.Bounds().Dx() != 1600 || frames[0].Img.Bounds().Dy() != 600 {
		t.Fatalf("без RandR-мониторов ожидали один кадр 1600x600: %d кадров, %v", len(frames), frames[0].Img.Bounds())
	}
	if got := frames[0].Img.RGBAAt(100, 100); got != red {
		t.Errorf("слева красный: %v", got)
	}
	if got := frames[0].Img.RGBAAt(1000, 300); got != blue {
		t.Errorf("справа синий: %v", got)
	}

	// Two monitors: left and right, 800 pixels each.
	addMonitor(t, display, "left", 0, 0, 800, 600)
	addMonitor(t, display, "right", 800, 0, 800, 600)
	frames, err = GrabAll()
	if err != nil || len(frames) != 2 {
		t.Fatalf("два монитора: %v, кадров %d", err, len(frames))
	}
	if frames[0].X != 0 || frames[1].X != 800 || frames[1].Y != 0 {
		t.Errorf("положение мониторов: (%d,%d) и (%d,%d)", frames[0].X, frames[0].Y, frames[1].X, frames[1].Y)
	}
	if frames[0].Img.Bounds().Dx() != 800 || frames[1].Img.Bounds().Dx() != 800 {
		t.Errorf("размеры: %v и %v", frames[0].Img.Bounds(), frames[1].Img.Bounds())
	}
	if got := frames[0].Img.RGBAAt(10, 10); got != red {
		t.Errorf("левый монитор красный: %v", got)
	}
	if got := frames[1].Img.RGBAAt(10, 10); got != blue {
		t.Errorf("правый монитор синий: %v", got)
	}
}

func TestIsSession(t *testing.T) {
	for _, c := range []struct {
		wayland, display string
		want             bool
	}{{"", ":0", true}, {"wayland-0", ":0", false}, {"wayland-0", "", false}, {"", "", false}} {
		t.Setenv("WAYLAND_DISPLAY", c.wayland)
		t.Setenv("DISPLAY", c.display)
		if got := IsSession(); got != c.want {
			t.Errorf("WAYLAND_DISPLAY=%q DISPLAY=%q: %v, ожидали %v", c.wayland, c.display, got, c.want)
		}
	}
}

func TestToRGBA(t *testing.T) {
	// X11 bytes: B, G, R, empty.
	img := toRGBA([]byte{10, 20, 30, 0, 40, 50, 60, 99}, 2, 1)
	if got := img.RGBAAt(0, 0); got != (color.RGBA{R: 30, G: 20, B: 10, A: 255}) {
		t.Errorf("первый пиксель: %v", got)
	}
	if got := img.RGBAAt(1, 0); got != (color.RGBA{R: 60, G: 50, B: 40, A: 255}) {
		t.Errorf("второй пиксель (пустой байт игнорируется): %v", got)
	}
}

// addMonitor adds a RandR 1.5 monitor without outputs (like xrandr --setmonitor ... none).
func addMonitor(t *testing.T, display, name string, x, y, w, h int) {
	t.Helper()
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := randr.Init(conn); err != nil {
		t.Skipf("нет RandR: %v", err)
	}
	root := xproto.Setup(conn).DefaultScreen(conn).Root
	atom, err := xproto.InternAtom(conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		t.Fatal(err)
	}
	info := randr.MonitorInfo{Name: atom.Atom, Primary: x == 0, Automatic: false, NOutput: 0,
		X: int16(x), Y: int16(y), Width: uint16(w), Height: uint16(h), WidthInMillimeters: uint32(w / 4), HeightInMillimeters: uint32(h / 4)}
	if err := randr.SetMonitorChecked(conn, root, info).Check(); err != nil {
		t.Skipf("Xvfb не принял монитор %s: %v", name, err)
	}
}

func TestDropOverlaps(t *testing.T) {
	r := func(x0, y0, x1, y1 int) Monitor { return Monitor{Rect: image.Rect(x0, y0, x1, y1)} }
	// A common screen covers two monitors: the two remain.
	got := dropOverlaps([]Monitor{r(0, 0, 1600, 600), r(0, 0, 800, 600), r(800, 0, 1600, 600)})
	if len(got) != 2 || got[0].Rect.Max.X != 800 {
		t.Errorf("накрывающий монитор должен исчезнуть: %v", got)
	}
	// Clone: two identical rectangles give one.
	if got := dropOverlaps([]Monitor{r(0, 0, 100, 100), r(0, 0, 100, 100)}); len(got) != 1 {
		t.Errorf("клоны: %v", got)
	}
	// Adjacent monitors are left alone.
	if got := dropOverlaps([]Monitor{r(0, 0, 100, 100), r(100, 0, 200, 100)}); len(got) != 2 {
		t.Errorf("соседние: %v", got)
	}
}
