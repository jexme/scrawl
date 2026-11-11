//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/jexme/scrawl/internal/settings"
)

// bareStartIsTray is false here: a start without arguments takes one screenshot (this is what
// the GNOME hotkey runs).
const bareStartIsTray = false

func firstRunHint() string { return "" }

func showAlreadyRunning(string) {}

// ensureTray starts the tray icon next to a screenshot unless it is hidden in the settings.
// The tray is its own detached process (a new session, so that it outlives this one); if one
// is already running, the new start exits silently, because of -autostart.
func ensureTray() {
	if !settings.Load().TrayIcon {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "-tray", "-autostart")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if cmd.Start() == nil {
		go cmd.Wait()
	}
}
