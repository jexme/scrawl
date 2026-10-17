//go:build linux

package editor

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"os/exec"

	"github.com/jexme/scrawl/internal/i18n"
)

// clipboardTool выбирает программу буфера обмена по сеансу: в Wayland это
// wl-copy, в X11 xclip. Возвращает имя команды и аргументы для типа данных.
//
// Если в Wayland нет wl-copy, но есть xclip и X-сервер (XWayland есть почти везде), берём
// xclip: композитор сводит буфер X11 с буфером Wayland, и вставка работает в обоих мирах.
// Без обоих программ возвращается ошибка с подсказкой, что поставить.
func clipboardTool(mime string) (name string, args []string, err error) {
	xclip := []string{"-selection", "clipboard", "-t", mime}
	if !isX11() {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return "wl-copy", []string{"--type", mime}, nil
		}
		if _, err := exec.LookPath("xclip"); err == nil && os.Getenv("DISPLAY") != "" {
			return "xclip", xclip, nil
		}
		return "", nil, errors.New(i18n.T("для буфера обмена в Wayland нужен wl-copy: sudo apt install wl-clipboard"))
	}
	if _, err := exec.LookPath("xclip"); err != nil {
		return "", nil, errors.New(i18n.T("для буфера обмена в X11 нужен xclip: sudo apt install xclip"))
	}
	return "xclip", xclip, nil
}

// clipboardCopy кладёт данные в буфер обмена. И wl-copy, и xclip сами уходят в
// фон и держат буфер, пока его не заменят.
func clipboardCopy(mime string, data []byte) error {
	name, args, err := clipboardTool(mime)
	if err != nil {
		return err
	}
	cp := exec.Command(name, args...)
	cp.Stdin = bytes.NewReader(data)
	return cp.Run()
}

// copyPNG кладёт PNG в буфер обмена.
func copyPNG(data []byte) error { return clipboardCopy("image/png", data) }

// copyFiles кладёт в буфер обмена список файлов (text/uri-list): так несколько
// картинок вставляются в чат или письмо как отдельные вложения.
func copyFiles(paths []string) error {
	var list bytes.Buffer
	for _, p := range paths {
		list.WriteString((&url.URL{Scheme: "file", Path: p}).String())
		list.WriteString("\r\n")
	}
	return clipboardCopy("text/uri-list", list.Bytes())
}
