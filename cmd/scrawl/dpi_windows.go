//go:build windows

package main

import "github.com/jexme/scrawl/internal/wincapture"

// Процесс должен стать DPI-aware до первого окна, иначе Windows растянет окно настроек и
// оверлей как картинку, а координаты мониторов с масштабом разойдутся с пикселями снимка.
func init() { wincapture.EnsureDPIAware() }
