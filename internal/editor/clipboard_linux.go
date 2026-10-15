//go:build linux

package editor

import (
	"bytes"
	"net/url"
	"os/exec"
)

// copyPNG кладёт PNG в буфер обмена Wayland. wl-copy сам уходит в фон и
// держит буфер, пока его не заменят.
func copyPNG(data []byte) error {
	cp := exec.Command("wl-copy", "--type", "image/png")
	cp.Stdin = bytes.NewReader(data)
	return cp.Run()
}

// copyFiles кладёт в буфер обмена список файлов (text/uri-list): так несколько
// картинок вставляются в чат или письмо как отдельные вложения.
func copyFiles(paths []string) error {
	var list bytes.Buffer
	for _, p := range paths {
		list.WriteString((&url.URL{Scheme: "file", Path: p}).String())
		list.WriteString("\r\n")
	}
	cp := exec.Command("wl-copy", "--type", "text/uri-list")
	cp.Stdin = &list
	return cp.Run()
}
