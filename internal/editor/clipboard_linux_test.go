//go:build linux

package editor

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestClipboardToolBySession(t *testing.T) {
	sandbox(t) // подставляет поддельный wl-copy

	// Wayland: wl-copy.
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", ":0")
	if name, args, err := clipboardTool("image/png"); err != nil || name != "wl-copy" || args[1] != "image/png" {
		t.Errorf("Wayland: %q %v %v", name, args, err)
	}

	// X11 без xclip: понятная ошибка, а не молчание.
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("PATH", t.TempDir())
	if _, _, err := clipboardTool("image/png"); err == nil || !strings.Contains(err.Error(), "xclip") {
		t.Errorf("без xclip нужна ошибка с подсказкой: %v", err)
	}
}

// startXvfb запускает виртуальный X-сервер и возвращает его DISPLAY; без Xvfb тест пропускается.
func startXvfb(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("Xvfb"); err != nil {
		t.Skip("нет Xvfb")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("Xvfb", "-displayfd", "3", "-screen", "0", "800x600x24", "-nolisten", "tcp", "-noreset")
	cmd.ExtraFiles = []*os.File{w}
	if err := cmd.Start(); err != nil {
		t.Skipf("Xvfb не запустился: %v", err)
	}
	w.Close()
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait(); r.Close() })
	buf := make([]byte, 32)
	n, _ := r.Read(buf)
	display := ":" + strings.TrimSpace(string(buf[:n]))
	for i := 0; i < 50; i++ {
		if exec.Command("xdpyinfo", "-display", display).Run() == nil {
			return display
		}
		time.Sleep(50 * time.Millisecond)
	}
	return display
}

// TestClipboardOnX11 кладёт картинку и список файлов в буфер обмена настоящего X-сервера и читает обратно.
func TestClipboardOnX11(t *testing.T) {
	if _, err := exec.LookPath("xclip"); err != nil {
		t.Skip("нет xclip")
	}
	display := startXvfb(t)
	t.Setenv("DISPLAY", display)
	t.Setenv("WAYLAND_DISPLAY", "")

	read := func(mime string) string {
		out, err := exec.Command("xclip", "-display", display, "-selection", "clipboard", "-o", "-t", mime).Output()
		if err != nil {
			t.Fatalf("чтение буфера (%s): %v", mime, err)
		}
		return string(out)
	}
	if err := copyPNG([]byte("PNG-BYTES")); err != nil {
		t.Fatal(err)
	}
	if got := read("image/png"); got != "PNG-BYTES" {
		t.Errorf("картинка в буфере: %q", got)
	}
	if err := copyFiles([]string{"/tmp/a b.png", "/tmp/c.png"}); err != nil {
		t.Fatal(err)
	}
	if got := read("text/uri-list"); got != "file:///tmp/a%20b.png\r\nfile:///tmp/c.png\r\n" {
		t.Errorf("список файлов в буфере: %q", got)
	}
}
