//go:build unix

package fonts

import (
	"os"
	"syscall"
)

// readFile maps the file into memory instead of reading it whole: a CJK font file weighs
// about 20 MB, while the program needs only the tables for a few glyphs. The mapping
// lives until the process exits.
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
