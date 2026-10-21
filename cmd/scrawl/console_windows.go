//go:build windows

package main

import (
	"os"
	"syscall"
)

// The program is built as a GUI (-H=windowsgui): when launched from the Start menu, autostart or
// by a hotkey, no black console window appears. If it is launched from a terminal,
// we attach to its console so that -version, -v and error messages are visible.
// Redirected output (> file, | program) is left as is.
func init() {
	attach := syscall.NewLazyDLL("kernel32.dll").NewProc("AttachConsole")
	if r, _, _ := attach.Call(^uintptr(0)); r == 0 { // ATTACH_PARENT_PROCESS; 0: the parent has no console
		return
	}
	for _, f := range []**os.File{&os.Stdout, &os.Stderr} {
		if _, err := (*f).Stat(); err == nil {
			continue // the handle is valid: output is already redirected
		}
		if h, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
			*f = h
		}
	}
}
