//go:build windows

package editor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procOpenClipboard            = user32.NewProc("OpenClipboard")
	procCloseClipboard           = user32.NewProc("CloseClipboard")
	procEmptyClipboard           = user32.NewProc("EmptyClipboard")
	procSetClipboardData         = user32.NewProc("SetClipboardData")
	procGlobalAlloc              = kernel32.NewProc("GlobalAlloc")
	procGlobalLock               = kernel32.NewProc("GlobalLock")
	procGlobalUnlock             = kernel32.NewProc("GlobalUnlock")
	procGlobalFree               = kernel32.NewProc("GlobalFree")
	procRegisterClipboardFormatW = user32.NewProc("RegisterClipboardFormatW")
)

const (
	cfDIB   = 8
	cfHDROP = 15

	gmemMoveable = 0x0002
	gmemZeroInit = 0x0040
)

// openClipboard opens the clipboard, retrying: another program (a clipboard manager,
// remote desktop) may hold it for a moment.
func openClipboard() error {
	var err error
	for i := 0; i < 20; i++ {
		var r uintptr
		if r, _, err = procOpenClipboard.Call(0); r != 0 {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("OpenClipboard: %w", err)
}

// setData puts a copy of data into the clipboard in the given format. On success the
// memory belongs to the system; on error we free it ourselves.
func setData(format uintptr, data []byte) error {
	h, _, err := procGlobalAlloc.Call(gmemMoveable|gmemZeroInit, uintptr(len(data)))
	if h == 0 {
		return fmt.Errorf("GlobalAlloc: %w", err)
	}
	addr, _, err := procGlobalLock.Call(h)
	if addr == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("GlobalLock: %w", err)
	}
	// The address was given by the system; converting through a pointer to the variable does
	// not involve the garbage collector.
	p := *(*unsafe.Pointer)(unsafe.Pointer(&addr))
	copy(unsafe.Slice((*byte)(p), len(data)), data)
	procGlobalUnlock.Call(h)
	if r, _, err := procSetClipboardData.Call(format, h); r == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("SetClipboardData(%d): %w", format, err)
	}
	return nil
}

// withClipboard empties the clipboard and calls fill while it is open.
func withClipboard(fill func() error) error {
	if err := openClipboard(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()
	if r, _, err := procEmptyClipboard.Call(); r == 0 {
		return fmt.Errorf("EmptyClipboard: %w", err)
	}
	return fill()
}

// copyPNG puts the image in two forms: "PNG" (lossless, used by Telegram, Discord and
// browsers) and CF_DIB (Paint, Word, nearly everything else).
func copyPNG(data []byte) error {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	name, _ := syscall.UTF16PtrFromString("PNG")
	pngFormat, _, err := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name)))
	if pngFormat == 0 {
		return fmt.Errorf("RegisterClipboardFormat: %w", err)
	}
	return withClipboard(func() error {
		if err := setData(cfDIB, dib(img)); err != nil {
			return err
		}
		return setData(pngFormat, data)
	})
}

// dib builds CF_DIB: a BITMAPINFOHEADER header and 32-bit BGRA pixels, bottom-up.
func dib(img image.Image) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), img, b.Min, draw.Src)

	out := make([]byte, 40+w*h*4)
	binary.LittleEndian.PutUint32(out[0:], 40)
	binary.LittleEndian.PutUint32(out[4:], uint32(w))
	binary.LittleEndian.PutUint32(out[8:], uint32(h)) // positive height: rows go bottom-up
	binary.LittleEndian.PutUint16(out[12:], 1)
	binary.LittleEndian.PutUint16(out[14:], 32)
	binary.LittleEndian.PutUint32(out[20:], uint32(w*h*4))
	px := out[40:]
	for y := 0; y < h; y++ {
		row := src.Pix[y*src.Stride : y*src.Stride+w*4]
		dst := px[(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			// The shot is opaque; if it is not, we blend it with white, as programs that do not
			// understand the alpha channel in CF_DIB do.
			r, g, bl, a := uint32(row[x*4]), uint32(row[x*4+1]), uint32(row[x*4+2]), uint32(row[x*4+3])
			if a != 255 {
				r, g, bl = (r*a+255*(255-a))/255, (g*a+255*(255-a))/255, (bl*a+255*(255-a))/255
			}
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = byte(bl), byte(g), byte(r), 0
		}
	}
	return out
}

// copyFiles puts a list of files (CF_HDROP): several images are pasted into a chat or an
// email as separate attachments.
func copyFiles(paths []string) error {
	if len(paths) == 0 {
		return errors.New("no files")
	}
	// DROPFILES: list offset (20), point (8 bytes), fNC, fWide; then the UTF-16 paths
	// separated by a null character and one more null at the end.
	buf := make([]byte, 20)
	binary.LittleEndian.PutUint32(buf[0:], 20)
	binary.LittleEndian.PutUint32(buf[16:], 1) // fWide
	for _, p := range paths {
		for _, u := range utf16.Encode([]rune(p)) {
			buf = binary.LittleEndian.AppendUint16(buf, u)
		}
		buf = binary.LittleEndian.AppendUint16(buf, 0)
	}
	buf = binary.LittleEndian.AppendUint16(buf, 0)
	return withClipboard(func() error { return setData(cfHDROP, buf) })
}
