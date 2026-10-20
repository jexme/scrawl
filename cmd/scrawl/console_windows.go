//go:build windows

package main

import (
	"os"
	"syscall"
)

// Программа собирается как GUI (-H=windowsgui): при запуске из меню Пуск, автозапуска или
// по горячей клавише чёрное окно консоли не появляется. Если запустили из терминала,
// подключаемся к его консоли, чтобы -version, -v и сообщения об ошибках были видны.
// Перенаправленный вывод (> файл, | программа) остаётся как есть.
func init() {
	attach := syscall.NewLazyDLL("kernel32.dll").NewProc("AttachConsole")
	if r, _, _ := attach.Call(^uintptr(0)); r == 0 { // ATTACH_PARENT_PROCESS; 0: консоли у родителя нет
		return
	}
	for _, f := range []**os.File{&os.Stdout, &os.Stderr} {
		if _, err := (*f).Stat(); err == nil {
			continue // дескриптор рабочий: вывод уже перенаправлен
		}
		if h, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
			*f = h
		}
	}
}
