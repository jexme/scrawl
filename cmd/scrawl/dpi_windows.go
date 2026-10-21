//go:build windows

package main

import "github.com/jexme/scrawl/internal/wincapture"

// The process must become DPI-aware before the first window, otherwise Windows will
// stretch the settings window and the overlay like a bitmap, and the coordinates of scaled
// monitors will diverge from the pixels of the capture.
func init() { wincapture.EnsureDPIAware() }
