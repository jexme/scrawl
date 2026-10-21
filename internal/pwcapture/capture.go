//go:build linux

// Package pwcapture reads one frame from a PipeWire stream through libpipewire.
// It replaces the external gst-launch: the user does not need GStreamer.
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

// ErrTimeout means the frame did not arrive in the allotted time. On a static
// monitor this happens all the time: Mutter delivers frames only on a redraw.
var ErrTimeout = errors.New("the PipeWire frame did not arrive in time")

var initOnce sync.Once

// Capture connects to the given PipeWire node through the fd from xdg-desktop-portal and returns
// the first frame. The fd passes into the library's ownership and is closed inside,
// so it must not be used after the call.
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
