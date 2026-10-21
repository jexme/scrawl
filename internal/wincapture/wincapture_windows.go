// Package wincapture captures the screen on Windows via GDI: each monitor as a separate frame,
// in physical pixels. The process must be DPI-aware (EnsureDPIAware), otherwise on
// scaled monitors the sizes and coordinates are wrong.
package wincapture

import (
	"errors"
	"fmt"
	"image"
	"sort"
	"syscall"
	"unsafe"

	"github.com/jexme/scrawl/internal/screencast"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procEnumDisplayMonitors           = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	procGetDC                         = user32.NewProc("GetDC")
	procReleaseDC                     = user32.NewProc("ReleaseDC")

	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
)

const (
	srccopy    = 0x00CC0020
	captureblt = 0x40000000
	dibRGB     = 0 // DIB_RGB_COLORS
	biRGB      = 0

	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 == (HANDLE)-4.
	perMonitorAwareV2 = ^uintptr(3)
)

type rect struct{ Left, Top, Right, Bottom int32 }

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

type bitmapInfoHeader struct {
	Size          uint32
	Width, Height int32
	Planes, Bits  uint16
	Compression   uint32
	SizeImage     uint32
	XPels, YPels  int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// EnsureDPIAware makes the process DPI-aware (Per-Monitor V2). Call it before creating
// windows and before any capture; a repeated call is harmless (the system simply rejects
// it).
func EnsureDPIAware() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		if r, _, _ := procSetProcessDpiAwarenessContext.Call(perMonitorAwareV2); r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call() // Windows 7/8 and old builds of 10
}

// Monitor is a monitor rectangle in virtual desktop coordinates (pixels).
type Monitor struct {
	Rect    image.Rectangle
	Primary bool
}

// Monitors returns the monitors left to right and top to bottom.
func Monitors() ([]Monitor, error) {
	var out []Monitor
	cb := syscall.NewCallback(func(hmon, _, _, _ uintptr) uintptr {
		mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		if r, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi))); r != 0 {
			out = append(out, Monitor{
				Rect:    image.Rect(int(mi.Monitor.Left), int(mi.Monitor.Top), int(mi.Monitor.Right), int(mi.Monitor.Bottom)),
				Primary: mi.Flags&1 != 0,
			})
		}
		return 1
	})
	if r, _, err := procEnumDisplayMonitors.Call(0, 0, cb, 0); r == 0 {
		return nil, fmt.Errorf("EnumDisplayMonitors: %w", err)
	}
	if len(out) == 0 {
		return nil, errors.New("no monitors found")
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rect.Min.X != out[j].Rect.Min.X {
			return out[i].Rect.Min.X < out[j].Rect.Min.X
		}
		return out[i].Rect.Min.Y < out[j].Rect.Min.Y
	})
	return out, nil
}

// GrabAll captures every monitor. Frame.X and Frame.Y are the monitor's top-left corner on the
// desktop; the editor uses them to open its window on the right monitor.
func GrabAll() ([]screencast.Frame, error) {
	mons, err := Monitors()
	if err != nil {
		return nil, err
	}
	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return nil, errors.New("GetDC(NULL) returned no screen context")
	}
	defer procReleaseDC.Call(0, screen)

	frames := make([]screencast.Frame, 0, len(mons))
	for _, m := range mons {
		img, err := grabRect(screen, m.Rect)
		if err != nil {
			return nil, fmt.Errorf("monitor %v: %w", m.Rect, err)
		}
		frames = append(frames, screencast.Frame{X: m.Rect.Min.X, Y: m.Rect.Min.Y, Img: img})
	}
	return frames, nil
}

// grabRect copies a screen rectangle into an RGBA image.
func grabRect(screen uintptr, r image.Rectangle) (*image.RGBA, error) {
	w, h := r.Dx(), r.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("empty monitor")
	}
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	if mem == 0 {
		return nil, errors.New("CreateCompatibleDC")
	}
	defer procDeleteDC.Call(mem)
	bmp, _, _ := procCreateCompatibleBitmap.Call(screen, uintptr(w), uintptr(h))
	if bmp == 0 {
		return nil, errors.New("CreateCompatibleBitmap")
	}
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(mem, bmp)
	defer procSelectObject.Call(mem, old)

	if ok, _, err := procBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), screen, uintptr(int32(r.Min.X)), uintptr(int32(r.Min.Y)), srccopy|captureblt); ok == 0 {
		return nil, fmt.Errorf("BitBlt: %w", err)
	}

	// Negative height: rows go from top to bottom.
	hdr := bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: int32(w), Height: -int32(h), Planes: 1, Bits: 32, Compression: biRGB}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if n, _, err := procGetDIBits.Call(mem, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&img.Pix[0])), uintptr(unsafe.Pointer(&hdr)), dibRGB); n == 0 {
		return nil, fmt.Errorf("GetDIBits: %w", err)
	}
	// GDI returns B, G, R, 0: swap the channels and make the frame opaque.
	pix := img.Pix
	for i := 0; i+3 < len(pix); i += 4 {
		pix[i], pix[i+2], pix[i+3] = pix[i+2], pix[i], 0xFF
	}
	return img, nil
}
