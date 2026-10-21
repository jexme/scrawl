//go:build !linux

// Package pwcapture reads a frame from a PipeWire stream (Linux only). Other
// systems get a stub so that the remaining packages still build.
package pwcapture

import (
	"errors"
	"image"
	"time"

	"github.com/jexme/scrawl/internal/i18n"
)

// ErrTimeout means the frame did not arrive within the allotted time.
var ErrTimeout = errors.New("the PipeWire frame did not arrive in time")

// Capture is unavailable on this system.
func Capture(fd int, node uint32, timeout time.Duration) (*image.RGBA, error) {
	return nil, errors.New(i18n.T("PipeWire is only available on Linux"))
}
