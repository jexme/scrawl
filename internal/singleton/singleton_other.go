//go:build !linux && !windows

package singleton

// Acquire does not block anything on this platform yet (Windows needs a named mutex).
func Acquire(string) (bool, error) { return true, nil }
