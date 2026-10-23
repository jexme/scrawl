//go:build windows

package windowerr

import "golang.org/x/sys/windows"

// show opens a message box. It blocks until it is closed, which is fine: the program is
// about to exit anyway.
func show(title, msg string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(msg)
	windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONERROR|windows.MB_TOPMOST|windows.MB_SETFOREGROUND)
}
