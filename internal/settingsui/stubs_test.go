package settingsui

import (
	"errors"

	"github.com/jexme/scrawl/internal/hotkey"
	"github.com/jexme/scrawl/internal/update"
)

// Тесты не должны трогать настройки GNOME и portal пользователя.
func init() {
	currentHotkey = func() string { return "" }
	hotkeyBinary = func() (string, error) { return "/test/bin/scrawl", nil }
	assignHotkey = func(accel, bin string) (hotkey.Result, error) { return hotkey.Result{}, nil }
	autostartOn = func() bool { return false }
	autostartEnable = func(string) error { return nil }
	autostartOff = func() error { return nil }
	checkUpdate = func() (update.Result, error) {
		return update.Result{}, errors.New("тест не ходит в сеть")
	}
	x11Session = func() bool { return false }
	grantPermission = func() error { return nil }
	setupMonitors = func(func(string)) (string, error) { return "", nil }
}
