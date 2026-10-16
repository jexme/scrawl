//go:build unix

package screencast

import (
	"os"
	"syscall"
)

// dupFD дублирует дескриптор файла и закрывает исходный файл: PipeWire получает
// собственную копию и сам её закроет.
func dupFD(f *os.File) (int, error) {
	fd, err := syscall.Dup(int(f.Fd()))
	f.Close()
	return fd, err
}
