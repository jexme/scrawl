//go:build !windows

package main

import (
	"time"

	"github.com/jexme/scrawl/internal/screencast"
)

func grabWindows(bool) ([]screencast.Frame, time.Time) {
	panic("GDI capture is Windows-only")
}
