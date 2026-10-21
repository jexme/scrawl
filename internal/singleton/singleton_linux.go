//go:build linux

// Package singleton prevents a second instance of the program from starting.
package singleton

import (
	"os"
	"path/filepath"
	"syscall"
)

// held keeps the open lock files: if the reference is lost, the garbage collector
// will close the file and the lock will be released.
var held []*os.File

// Acquire takes an exclusive file lock in XDG_RUNTIME_DIR. It returns
// false if another instance is already running. The lock is released automatically when the
// process exits, even abnormally, so there are no "stuck" locks.
func Acquire(name string) (bool, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	f, err := os.OpenFile(filepath.Join(dir, name+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			return false, nil
		}
		return false, err
	}
	held = append(held, f)
	return true, nil
}
