//go:build !unix

package screencast

import (
	"errors"
	"os"
)

// dupFD is unavailable on this system: PipeWire capture exists only on Linux.
func dupFD(f *os.File) (int, error) {
	f.Close()
	return -1, errors.New("PipeWire is only available on Linux")
}
