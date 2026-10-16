//go:build unix

package fonts

import (
	"os"
	"syscall"
)

// readFile отображает файл в память, а не читает целиком: файл CJK-шрифта весит
// около 20 МБ, а программе нужны лишь таблицы нескольких символов. Отображение
// живёт до конца процесса.
func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() == 0 {
		return nil, os.ErrInvalid
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(st.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return os.ReadFile(path)
	}
	return data, nil
}
