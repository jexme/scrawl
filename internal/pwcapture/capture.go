//go:build linux

// Package pwcapture читает один кадр из потока PipeWire через libpipewire.
// Заменяет внешний gst-launch: пользователю не нужен GStreamer.
package pwcapture

/*
#cgo pkg-config: libpipewire-0.3
#include <stdlib.h>
#include "capture.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"sync"
	"time"
	"unsafe"
)

// ErrTimeout означает, что кадр не пришёл за отведённое время. На статичном
// мониторе так бывает постоянно: Mutter отдаёт кадры только при перерисовке.
var ErrTimeout = errors.New("кадр PipeWire не пришёл вовремя")

var initOnce sync.Once

// Capture подключается к узлу node через fd от xdg-desktop-portal и возвращает
// первый кадр. fd переходит во владение библиотеки и закрывается внутри,
// после вызова его использовать нельзя.
func Capture(fd int, node uint32, timeout time.Duration) (*image.RGBA, error) {
	initOnce.Do(func() { C.sec_init() })

	var (
		pixels        *C.uint8_t
		width, height C.int
		errbuf        [256]C.char
	)
	rc := C.sec_capture(C.int(fd), C.uint32_t(node), C.int(timeout.Milliseconds()),
		&pixels, &width, &height, &errbuf[0], C.int(len(errbuf)))
	switch rc {
	case C.SEC_OK:
		defer C.free(unsafe.Pointer(pixels))
		w, h := int(width), int(height)
		return &image.RGBA{
			Pix:    C.GoBytes(unsafe.Pointer(pixels), C.int(w*h*4)),
			Stride: w * 4,
			Rect:   image.Rect(0, 0, w, h),
		}, nil
	case C.SEC_TIMEOUT:
		return nil, ErrTimeout
	default:
		return nil, fmt.Errorf("PipeWire: %s", C.GoString(&errbuf[0]))
	}
}
