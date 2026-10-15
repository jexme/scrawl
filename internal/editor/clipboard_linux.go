//go:build linux

package editor

import (
	"bytes"
	"os/exec"
)

// copyPNG кладёт PNG в буфер обмена Wayland. wl-copy сам уходит в фон и
// держит буфер, пока его не заменят.
func copyPNG(data []byte) error {
	cp := exec.Command("wl-copy", "--type", "image/png")
	cp.Stdin = bytes.NewReader(data)
	return cp.Run()
}
