//go:build unix

package screencast

import (
	"os"
	"syscall"
)

// dupFD duplicates a file descriptor and closes the original file: PipeWire gets its own
// copy and closes it itself.
func dupFD(f *os.File) (int, error) {
	fd, err := syscall.Dup(int(f.Fd()))
	f.Close()
	return fd, err
}
