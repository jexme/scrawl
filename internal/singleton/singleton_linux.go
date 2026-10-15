//go:build linux

// Package singleton не даёт запустить второй экземпляр программы.
package singleton

import (
	"os"
	"path/filepath"
	"syscall"
)

// held хранит открытые файлы блокировок: если потерять ссылку, сборщик мусора
// закроет файл и блокировка снимется.
var held []*os.File

// Acquire берёт эксклюзивную блокировку файла в XDG_RUNTIME_DIR. Возвращает
// false, если другой экземпляр уже работает. Блокировка снимается сама, когда
// процесс завершается, даже аварийно, поэтому «застрявших» блокировок не бывает.
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
