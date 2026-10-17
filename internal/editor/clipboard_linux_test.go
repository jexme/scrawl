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
	isX11 = func() bool { return false }
	if name, args, err := clipboardTool("image/png"); err != nil || name != "wl-copy" || args[1] != "image/png" {
		t.Errorf("Wayland: %q %v %v", name, args, err)
	}

	// X11 без xclip: понятная ошибка, а не молчание.
	isX11 = func() bool { return true }
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
	isX11 = func() bool { return true }

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

// fakeBin кладёт в новую папку пустые исполняемые программы с указанными именами и делает её единственным PATH.
func fakeBin(t *testing.T, names ...string) {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(dir+"/"+n, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
}

// TestClipboardToolWaylandFallback: в Wayland без wl-copy копируем через xclip (XWayland сводит буфер
// X11 с Wayland), а без обеих программ и без X-сервера даём ошибку с тем, что поставить.
func TestClipboardToolWaylandFallback(t *testing.T) {
	old := isX11
	defer func() { isX11 = old }()
	isX11 = func() bool { return false }

	// Есть wl-copy: он главнее, даже если есть и xclip.
	fakeBin(t, "wl-copy", "xclip")
	t.Setenv("DISPLAY", ":0")
	if name, args, err := clipboardTool("image/png"); err != nil || name != "wl-copy" || args[0] != "--type" {
		t.Errorf("с wl-copy: %q %v %v", name, args, err)
	}

	// Нет wl-copy, есть xclip и X-сервер: xclip с нужным типом.
	fakeBin(t, "xclip")
	name, args, err := clipboardTool("text/uri-list")
	if err != nil || name != "xclip" || strings.Join(args, " ") != "-selection clipboard -t text/uri-list" {
		t.Errorf("без wl-copy: %q %v %v", name, args, err)
	}

	// Нет wl-copy, xclip есть, но X-сервера нет: xclip бесполезен, нужна подсказка про wl-clipboard.
	t.Setenv("DISPLAY", "")
	if _, _, err := clipboardTool("image/png"); err == nil || !strings.Contains(err.Error(), "wl-clipboard") {
		t.Errorf("без X-сервера нужна ошибка с wl-clipboard: %v", err)
	}

	// Нет ничего: понятная ошибка, а не «executable file not found».
	fakeBin(t)
	t.Setenv("DISPLAY", ":0")
	err = copyPNG([]byte("x"))
	if err == nil || !strings.Contains(err.Error(), "wl-clipboard") || strings.Contains(err.Error(), "executable file not found") {
		t.Errorf("без программ буфера: %v", err)
	}
}
