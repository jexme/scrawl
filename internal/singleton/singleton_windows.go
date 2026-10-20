//go:build windows

package singleton

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows"
)

// held держит дескрипторы мьютексов до конца процесса: пока дескриптор открыт, второй
// экземпляр видит имя занятым.
var held []windows.Handle

// Acquire занимает именованный мьютекс сеанса. false: такой экземпляр уже запущен.
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
