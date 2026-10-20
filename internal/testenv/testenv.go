// Package testenv подменяет папки пользователя в тестах так, чтобы они не
// трогали настоящий профиль ни на Linux (XDG_*), ни на Windows (%AppData%).
package testenv

import "testing"

// Config направляет папку настроек (os.UserConfigDir) в dir.
func Config(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
}

// Cache направляет папку кэша (os.UserCacheDir) в dir.
func Cache(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
}

// Home направляет домашнюю папку (os.UserHomeDir) в dir.
func Home(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}
