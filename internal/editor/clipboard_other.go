//go:build !linux && !windows

package editor

import "github.com/jexme/scrawl/internal/i18n"

import "errors"

func copyPNG([]byte) error {
	return errors.New(i18n.T("copying to the clipboard is not implemented on this platform yet"))
}

func copyFiles([]string) error {
	return errors.New(i18n.T("copying files to the clipboard is not implemented on this platform yet"))
}
