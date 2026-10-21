//go:build !linux

// Package pwcapture читает кадр из потока PipeWire (только Linux). На других
// системах есть заглушка, чтобы остальные пакеты собирались.
package pwcapture

import (
	"errors"
	"image"
	"time"

	"github.com/jexme/scrawl/internal/i18n"
)

// ErrTimeout означает, что кадр не пришёл за отведённое время.
var ErrTimeout = errors.New("the PipeWire frame did not arrive in time")

// Capture на этой системе недоступен.
func Capture(fd int, node uint32, timeout time.Duration) (*image.RGBA, error) {
	return nil, errors.New(i18n.T("PipeWire is only available on Linux"))
}
