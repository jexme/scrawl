//go:build !linux && !windows

package editor

import "github.com/jexme/scrawl/internal/i18n"

import "errors"

func copyPNG([]byte) error {
	return errors.New(i18n.T("копирование в буфер обмена пока не реализовано на этой платформе"))
}

func copyFiles([]string) error {
	return errors.New(i18n.T("копирование файлов в буфер обмена пока не реализовано на этой платформе"))
}
