//go:build linux

package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"
)

// wmLog records what the stub window manager saw from the program.
type wmLog struct {
	mu         sync.Mutex
	mapped     map[xproto.Window]bool
	moved      map[xproto.Window]image.Point // position from ConfigureRequest
	fullscreen map[xproto.Window]bool        // a _NET_WM_STATE_FULLSCREEN request arrived
}

// runFakeWM takes over the root window as a window manager (SubstructureRedirect) and
// behaves like an EWMH-compliant one: it maps windows, honours their position requests and,
// on a fullscreen request, stretches the window over the monitor it is on. This checks
// the client side of the protocol: what the program asks the manager for. The manager
// itself (mutter, kwin) is not tested this way.
func runFakeWM(t *testing.T, display string, monitors []image.Rectangle) *wmLog {
	t.Helper()
	c, err := xgb.NewConnDisplay(display)
	if err != nil {
		t.Fatal(err)
	}
	root := xproto.Setup(c).DefaultScreen(c).Root
	if err := xproto.ChangeWindowAttributesChecked(c, root, xproto.CwEventMask,
		[]uint32{xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify}).Check(); err != nil {
		t.Fatal(err)
	}
	atom := func(name string) xproto.Atom {
		a, err := xproto.InternAtom(c, false, uint16(len(name)), name).Reply()
		if err != nil {
			t.Fatal(err)
		}
		return a.Atom
	}
	wmState, fullscreen := atom("_NET_WM_STATE"), atom("_NET_WM_STATE_FULLSCREEN")
	log := &wmLog{mapped: map[xproto.Window]bool{}, moved: map[xproto.Window]image.Point{}, fullscreen: map[xproto.Window]bool{}}
	t.Cleanup(func() { c.Close() })

	go func() {
		for {
			ev, err := c.WaitForEvent()
			if ev == nil && err == nil {
				return
			}
			if err != nil {
				continue
			}
			log.mu.Lock()
			switch e := ev.(type) {
			case xproto.MapRequestEvent:
				log.mapped[e.Window] = true
				xproto.MapWindow(c, e.Window)
			case xproto.ConfigureRequestEvent:
				var vals []uint32
				if e.ValueMask&xproto.ConfigWindowX != 0 {
					vals = append(vals, uint32(int32(e.X)))
				}
				if e.ValueMask&xproto.ConfigWindowY != 0 {
					vals = append(vals, uint32(int32(e.Y)))
				}
				if e.ValueMask&xproto.ConfigWindowWidth != 0 {
					vals = append(vals, uint32(e.Width))
				}
				if e.ValueMask&xproto.ConfigWindowHeight != 0 {
					vals = append(vals, uint32(e.Height))
				}
				if e.ValueMask&xproto.ConfigWindowX != 0 || e.ValueMask&xproto.ConfigWindowY != 0 {
					log.moved[e.Window] = image.Pt(int(e.X), int(e.Y))
				}
				mask := e.ValueMask & (xproto.ConfigWindowX | xproto.ConfigWindowY | xproto.ConfigWindowWidth | xproto.ConfigWindowHeight)
				xproto.ConfigureWindow(c, e.Window, mask, vals)
			case xproto.ClientMessageEvent:
				data := e.Data.Data32
				if e.Type == wmState && data[0] == 1 && (xproto.Atom(data[1]) == fullscreen || xproto.Atom(data[2]) == fullscreen) {
					log.fullscreen[e.Window] = true
					// Stretch over the monitor where the window's top-left corner is.
					if g, err := xproto.GetGeometry(c, xproto.Drawable(e.Window)).Reply(); err == nil {
						for _, m := range monitors {
							if image.Pt(int(g.X), int(g.Y)).In(m) {
								xproto.ConfigureWindow(c, e.Window,
									xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
									[]uint32{uint32(m.Min.X), uint32(m.Min.Y), uint32(m.Dx()), uint32(m.Dy())})
							}
						}
					}
				}
			}
			log.mu.Unlock()
		}
	}()
	return log
}

// TestOverlayWindowsPerMonitorOnX11 runs the real program in an X11 session (Xvfb with two
// RandR monitors) and checks that a window opens on each monitor and asks for fullscreen
// exactly there. Under Xvfb the program may crash during EGL initialisation (Xvfb has no
// DRI), but by then the windows are already created and placed: that is what is checked.
func TestOverlayWindowsPerMonitorOnX11(t *testing.T) {
	if testing.Short() {
		t.Skip("долгий тест: собирает программу")
	}
	if _, err := exec.LookPath("Xvfb"); err != nil {
		t.Skip("нет Xvfb")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "scrawl")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("сборка: %v\n%s", err, out)
	}

	r, w, _ := os.Pipe()
	xvfb := exec.Command("Xvfb", "-displayfd", "3", "-screen", "0", "1600x600x24", "-nolisten", "tcp", "-noreset")
	xvfb.ExtraFiles = []*os.File{w}
	if err := xvfb.Start(); err != nil {
		t.Skipf("Xvfb не запустился: %v", err)
	}
	w.Close()
	t.Cleanup(func() { xvfb.Process.Kill(); xvfb.Wait(); r.Close() })
	buf := make([]byte, 32)
	n, _ := r.Read(buf)
	display := ":" + strings.TrimSpace(string(buf[:n]))
	for i := 0; i < 50; i++ {
		if c, err := xgb.NewConnDisplay(display); err == nil {
			c.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Two RandR monitors, like a person with two screens.
	left, right := image.Rect(0, 0, 800, 600), image.Rect(800, 0, 1600, 600)
	func() {
		c, err := xgb.NewConnDisplay(display)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err := randr.Init(c); err != nil {
			t.Skipf("нет RandR: %v", err)
		}
		root := xproto.Setup(c).DefaultScreen(c).Root
		for i, m := range []image.Rectangle{left, right} {
			name := fmt.Sprintf("mon%d", i)
			a, _ := xproto.InternAtom(c, false, uint16(len(name)), name).Reply()
			err := randr.SetMonitorChecked(c, root, randr.MonitorInfo{Name: a.Atom, Primary: i == 0,
				X: int16(m.Min.X), Y: int16(m.Min.Y), Width: uint16(m.Dx()), Height: uint16(m.Dy()),
				WidthInMillimeters: 200, HeightInMillimeters: 150}).Check()
			if err != nil {
				t.Skipf("Xvfb не принял монитор: %v", err)
			}
		}
	}()
	log := runFakeWM(t, display, []image.Rectangle{left, right})

	// Hide Wayland: libwayland without WAYLAND_DISPLAY still looks for wayland-0 in XDG_RUNTIME_DIR.
	env := []string{"HOME=" + dir, "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "XDG_RUNTIME_DIR=" + dir,
		"DISPLAY=" + display, "PATH=" + os.Getenv("PATH"), "LIBGL_ALWAYS_SOFTWARE=1"}
	app := exec.Command(bin, "-fresh", "-exit-after", "3s")
	app.Env = env
	app.Stdout, app.Stderr = nil, nil
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { app.Wait(); close(done) }()
	deadline := time.After(10 * time.Second)
	for {
		log.mu.Lock()
		ready := len(log.fullscreen) >= 2
		log.mu.Unlock()
		if ready {
			break
		}
		select {
		case <-done:
			// The program exited or crashed: look at what it managed to do.
			log.mu.Lock()
			ready = len(log.fullscreen) >= 2
			log.mu.Unlock()
			goto check
		case <-deadline:
			app.Process.Kill()
			goto check
		case <-time.After(50 * time.Millisecond):
		}
	}
check:
	app.Process.Kill()
	log.mu.Lock()
	defer log.mu.Unlock()
	if len(log.mapped) != 2 {
		t.Errorf("окон показано %d, ожидали по одному на монитор (2)", len(log.mapped))
	}
	if len(log.fullscreen) != 2 {
		t.Fatalf("полноэкранный режим запросили %d окон из 2", len(log.fullscreen))
	}
	got := map[image.Point]bool{}
	for win := range log.fullscreen {
		got[log.moved[win]] = true
	}
	if !got[left.Min] || !got[right.Min] {
		t.Errorf("окна должны просить положение (0,0) и (800,0), а просили %v", log.moved)
	}
}
