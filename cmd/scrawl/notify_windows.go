//go:build windows

package main

import "golang.org/x/sys/windows"

// notify shows a message in a window: when launched by a hotkey or from autostart
// there is no terminal, and without a window a failure goes unnoticed.
func notify(title, msg string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(msg)
	windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONWARNING|windows.MB_TOPMOST|windows.MB_SETFOREGROUND)
}
