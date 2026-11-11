package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jexme/scrawl/internal/autostart"
)

// Small facts about the tray process that other starts of the program need. They live in
// the user cache directory next to the update marker, as plain one-line files.

func cacheFile(name string) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "scrawl", name)
}

// markFirstRun reports whether this is the first tray start on this account and records
// that it was seen, so the one-time hint is shown exactly once. Failing to write the marker
// only means the hint may be shown again, which is harmless.
func markFirstRun() bool {
	path := cacheFile("tray-welcome")
	if _, err := os.Stat(path); err == nil {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
		os.WriteFile(path, []byte("1\n"), 0o600)
	}
	return true
}

// recordTrayVersion stores the version of the running tray, so that a second start
// (for example the new version after a manual update) can tell the user what is running.
func recordTrayVersion(v string) {
	path := cacheFile("tray-version")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
		os.WriteFile(path, []byte(v+"\n"), 0o600)
	}
}

// runningTrayVersion returns the version recorded by the running tray, or "" if unknown.
// It is only called when the tray's single-instance lock is known to be held by another
// process, so a stale file from a crashed tray is never reported.
func runningTrayVersion() string {
	data, err := os.ReadFile(cacheFile("tray-version"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// enableAutostartOnce turns on the start of the tray at login the first time the tray runs on
// this account, so that the icon is there by default. It happens once: a user who switches
// autostart off in the settings is not overridden on the next start. A binary in a temporary
// folder (go run, an opened archive) is never written into autostart.
func enableAutostartOnce(exe string) {
	path := cacheFile("tray-autostart-default")
	if _, err := os.Stat(path); err == nil {
		return
	}
	if !strings.HasPrefix(exe, os.TempDir()+string(filepath.Separator)) && !autostart.Enabled() {
		if err := autostart.Enable(exe); err != nil {
			fmt.Fprintln(os.Stderr, "autostart:", err)
			return
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
		os.WriteFile(path, []byte("1\n"), 0o600)
	}
}
