//go:build windows

package main

import (
	"fmt"
	"github.com/jexme/scrawl/internal/i18n"
	"time"

	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/wincapture"
)

// grabWindows captures all monitors through GDI.
func grabWindows(verbose bool) ([]screencast.Frame, time.Time) {
	t0 := time.Now() // the moment of the "hotkey press"
	frames, err := wincapture.GrabAll()
	if err != nil {
		fatal(i18n.T("frame: %v"), err)
	}
	if verbose {
		fmt.Printf("frames (%d) received in %s: GDI\n", len(frames), time.Since(t0).Round(time.Millisecond))
	}
	return frames, t0
}
