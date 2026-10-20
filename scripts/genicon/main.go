// genicon записывает packaging/windows/scrawl.ico: значок scrawl.exe из того же рисунка, что и значок в
// трее (tray.Framed). Запуск: go run ./scripts/genicon. Файл лежит в репозитории, перегенерировать
// его нужно только при смене рисунка.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"os"

	"github.com/jexme/scrawl/internal/tray"
)

func main() {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	var images [][]byte
	for _, s := range sizes {
		var b bytes.Buffer
		if err := png.Encode(&b, tray.Framed(s)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		images = append(images, b.Bytes())
	}

	// ICO: заголовок, таблица записей, затем PNG-данные (формат понимает Windows Vista и новее).
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		dim := byte(s)
		if s >= 256 {
			dim = 0 // 0 означает 256
		}
		out.Write([]byte{dim, dim, 0, 0})
		binary.Write(&out, binary.LittleEndian, [2]uint16{1, 32}) // плоскости, бит на пиксель
		binary.Write(&out, binary.LittleEndian, [2]uint32{uint32(len(images[i])), uint32(offset)})
		offset += len(images[i])
	}
	for _, img := range images {
		out.Write(img)
	}
	path := "packaging/windows/scrawl.ico"
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wrote", path, out.Len(), "bytes")
}
