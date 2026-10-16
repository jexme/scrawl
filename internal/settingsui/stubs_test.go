package settingsui

import "github.com/jexme/scrawl/internal/hotkey"

// Тесты не должны трогать настройки GNOME и portal пользователя.
func init() {
	currentHotkey = func() string { return "" }
	hotkeyBinary = func() (string, error) { return "/test/bin/scrawl", nil }
	assignHotkey = func(accel, bin string) (hotkey.Result, error) { return hotkey.Result{}, nil }
	grantPermission = func() error { return nil }
	setupMonitors = func(func(string)) (string, error) { return "", nil }
}
