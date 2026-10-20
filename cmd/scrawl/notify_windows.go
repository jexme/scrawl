//go:build windows

package main

import "golang.org/x/sys/windows"

// notify показывает сообщение окном: при запуске по горячей клавише или из автозапуска
// терминала нет, и без окна сбой остаётся незаметным.
func notify(title, msg string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(msg)
	windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONWARNING|windows.MB_TOPMOST|windows.MB_SETFOREGROUND)
}
