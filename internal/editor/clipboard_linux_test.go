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
	sandbox(t) // installs a fake wl-copy

	// Wayland: wl-copy.
	isX11 = func() bool { return false }
	if name, args, err := clipboardTool("image/png"); err != nil || name != "wl-copy" || args[1] != "image/png" {
		t.Errorf("Wayland: %q %v %v", name, args, err)
	}

	// X11 without xclip: a clear error, not silence.
	isX11 = func() bool { return true }
	t.Setenv("PATH", t.TempDir())
	if _, _, err := clipboardTool("image/png"); err == nil || !strings.Contains(err.Error(), "xclip") {
		t.Errorf("без xclip нужна ошибка с подсказкой: %v", err)
	}
}

// startXvfb starts a virtual X server and returns its DISPLAY; without Xvfb the test is
// skipped.
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

// TestClipboardOnX11 puts an image and a list of files into the clipboard of a real X
// server and reads them back.
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

// fakeBin puts empty executables with the given names into a new folder and makes it the
// only PATH.
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

// TestClipboardToolWaylandFallback: on Wayland without wl-copy we copy through xclip
// (XWayland syncs the X11 clipboard with Wayland), and without both programs and without
// an X server we give an error saying what to install.
func TestClipboardToolWaylandFallback(t *testing.T) {
	old := isX11
	defer func() { isX11 = old }()
	isX11 = func() bool { return false }

	// wl-copy is present: it takes precedence, even if xclip is present too.
	fakeBin(t, "wl-copy", "xclip")
	t.Setenv("DISPLAY", ":0")
	if name, args, err := clipboardTool("image/png"); err != nil || name != "wl-copy" || args[0] != "--type" {
		t.Errorf("с wl-copy: %q %v %v", name, args, err)
	}

	// No wl-copy, but there is xclip and an X server: xclip with the needed type.
	fakeBin(t, "xclip")
	name, args, err := clipboardTool("text/uri-list")
	if err != nil || name != "xclip" || strings.Join(args, " ") != "-selection clipboard -t text/uri-list" {
		t.Errorf("без wl-copy: %q %v %v", name, args, err)
	}

	// No wl-copy, xclip is present, but there is no X server: xclip is useless, a hint about
	// wl-clipboard is needed.
	t.Setenv("DISPLAY", "")
	if _, _, err := clipboardTool("image/png"); err == nil || !strings.Contains(err.Error(), "wl-clipboard") {
		t.Errorf("без X-сервера нужна ошибка с wl-clipboard: %v", err)
	}

	// Nothing at all: a clear error, not "executable file not found".
	fakeBin(t)
	t.Setenv("DISPLAY", ":0")
	err = copyPNG([]byte("x"))
	if err == nil || !strings.Contains(err.Error(), "wl-clipboard") || strings.Contains(err.Error(), "executable file not found") {
		t.Errorf("без программ буфера: %v", err)
	}
}
