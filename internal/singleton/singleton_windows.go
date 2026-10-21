//go:build windows

package singleton

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows"
)

// held keeps the mutex handles until the process ends: while a handle is open, a second
// instance sees the name as taken.
var held []windows.Handle

// Acquire takes the session's named mutex. false: such an instance is already running.
func Acquire(name string) (bool, error) {
	n, err := windows.UTF16PtrFromString(`Local\scrawl-` + strings.ReplaceAll(name, `\`, "_"))
	if err != nil {
		return false, err
	}
	h, err := windows.CreateMutex(nil, false, n)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		windows.CloseHandle(h)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	held = append(held, h)
	return true, nil
}
