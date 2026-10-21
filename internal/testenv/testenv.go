// Package testenv replaces the user's folders in tests so that they do not
// touch the real profile, either on Linux (XDG_*) or on Windows (%AppData%).
package testenv

import "testing"

// Config points the config folder (os.UserConfigDir) to dir.
func Config(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
}

// Cache points the cache folder (os.UserCacheDir) to dir.
func Cache(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
}

// Home points the home folder (os.UserHomeDir) to dir.
func Home(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}
