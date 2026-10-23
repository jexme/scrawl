//go:build !windows

package main

// bareStartIsTray is false here: a start without arguments takes one screenshot (this is what
// the GNOME hotkey runs).
const bareStartIsTray = false

func firstRunHint() string { return "" }

func showAlreadyRunning(string) {}
