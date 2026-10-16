//go:build !unix

package screencast

import (
	"errors"
	"os"
)

// dupFD на этой системе недоступен: захват через PipeWire есть только в Linux.
func dupFD(f *os.File) (int, error) {
	f.Close()
	return -1, errors.New("PipeWire is only available on Linux")
}
