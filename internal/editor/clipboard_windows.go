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

// openClipboard открывает буфер обмена, повторяя попытку: его может на мгновение занимать
// другая программа (менеджер буфера, удалённый рабочий стол).
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

// setData кладёт копию data в буфер обмена в формате format. При успехе память принадлежит
// системе; при ошибке её освобождаем сами.
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
	// Адрес выдала система; приведение через указатель на переменную не трогает сборщик мусора.
	p := *(*unsafe.Pointer)(unsafe.Pointer(&addr))
	copy(unsafe.Slice((*byte)(p), len(data)), data)
	procGlobalUnlock.Call(h)
	if r, _, err := procSetClipboardData.Call(format, h); r == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("SetClipboardData(%d): %w", format, err)
	}
	return nil
}

// withClipboard очищает буфер и вызывает fill, пока он открыт.
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

// copyPNG кладёт картинку в двух видах: «PNG» (без потерь, его берут Telegram, Discord,
// браузеры) и CF_DIB (Paint, Word, почти всё остальное).
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

// dib собирает CF_DIB: заголовок BITMAPINFOHEADER и 32-битные пиксели BGRA снизу вверх.
func dib(img image.Image) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), img, b.Min, draw.Src)

	out := make([]byte, 40+w*h*4)
	binary.LittleEndian.PutUint32(out[0:], 40)
	binary.LittleEndian.PutUint32(out[4:], uint32(w))
	binary.LittleEndian.PutUint32(out[8:], uint32(h)) // положительная высота: строки снизу вверх
	binary.LittleEndian.PutUint16(out[12:], 1)
	binary.LittleEndian.PutUint16(out[14:], 32)
	binary.LittleEndian.PutUint32(out[20:], uint32(w*h*4))
	px := out[40:]
	for y := 0; y < h; y++ {
		row := src.Pix[y*src.Stride : y*src.Stride+w*4]
		dst := px[(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			// Снимок непрозрачный; если нет, смешиваем с белым, как это делают программы,
			// не понимающие альфа-канал в CF_DIB.
			r, g, bl, a := uint32(row[x*4]), uint32(row[x*4+1]), uint32(row[x*4+2]), uint32(row[x*4+3])
			if a != 255 {
				r, g, bl = (r*a+255*(255-a))/255, (g*a+255*(255-a))/255, (bl*a+255*(255-a))/255
			}
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = byte(bl), byte(g), byte(r), 0
		}
	}
	return out
}

// copyFiles кладёт список файлов (CF_HDROP): несколько картинок вставляются в чат или
// письмо как отдельные вложения.
func copyFiles(paths []string) error {
	if len(paths) == 0 {
		return errors.New("no files")
	}
	// DROPFILES: смещение списка (20), точка (8 байт), fNC, fWide; дальше UTF-16 пути
	// через нулевой символ и ещё один нуль в конце.
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
