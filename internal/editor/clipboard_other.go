//go:build !linux

package editor

import "errors"

func copyPNG([]byte) error {
	return errors.New("копирование в буфер обмена пока не реализовано на этой платформе")
}

func copyFiles([]string) error {
	return errors.New("копирование файлов в буфер обмена пока не реализовано на этой платформе")
}
