//go:build windows

package main

import (
	"fmt"
	"github.com/jexme/scrawl/internal/i18n"
	"time"

	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/wincapture"
)

// grabWindows снимает все мониторы через GDI.
func grabWindows(verbose bool) ([]screencast.Frame, time.Time) {
	t0 := time.Now() // момент «нажатия хоткея»
	frames, err := wincapture.GrabAll()
	if err != nil {
		fatal(i18n.T("кадр: %v"), err)
	}
	if verbose {
		fmt.Printf("кадры (%d) получены за %s: GDI\n", len(frames), time.Since(t0).Round(time.Millisecond))
	}
	return frames, t0
}
