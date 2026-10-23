//go:build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows"

	"github.com/jexme/scrawl/internal/i18n"
)

// bareStartIsTray: on Windows a double click on scrawl.exe passes no arguments, and a
// screenshot is not what the user expects from it; it starts the tray instead.
const bareStartIsTray = true

// firstRunHint is the one-time balloon shown by the tray on the very first start.
func firstRunHint() string {
	if !markFirstRun() {
		return ""
	}
	return i18n.T("Scrawl is running in the notification area. Press Print (or the combination from the settings) to take a screenshot.")
}

// showAlreadyRunning tells the user that the tray of another start owns the hotkey. After a
// manual update this is the old version, which keeps serving the hotkey until it is quit.
func showAlreadyRunning(running string) {
	var msg string
	if running != "" {
		msg = fmt.Sprintf(i18n.T("Scrawl is already running (version %s). Quit it from the tray icon menu, then start the new version."), running)
	} else {
		msg = i18n.T("Scrawl is already running. Quit it from the tray icon menu, then start the new version.")
	}
	t, _ := windows.UTF16PtrFromString("Scrawl")
	m, _ := windows.UTF16PtrFromString(msg)
	windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONINFORMATION|windows.MB_SETFOREGROUND)
}
