//go:build windows

package tray

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/jexme/scrawl/internal/hotkey"
	"github.com/jexme/scrawl/internal/i18n"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW       = user32.NewProc("RegisterClassExW")
	procCreateWindowExW        = user32.NewProc("CreateWindowExW")
	procDefWindowProcW         = user32.NewProc("DefWindowProcW")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procGetMessageW            = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessageW       = user32.NewProc("DispatchMessageW")
	procPostMessageW           = user32.NewProc("PostMessageW")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procRegisterHotKey         = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey       = user32.NewProc("UnregisterHotKey")
	procSetTimer               = user32.NewProc("SetTimer")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
	procGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	procCreateIcon             = user32.NewProc("CreateIcon")
	procDestroyIcon            = user32.NewProc("DestroyIcon")
	procShellNotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW       = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmDestroy     = 0x0002
	wmClose       = 0x0010
	wmTimer       = 0x0113
	wmHotkey      = 0x0312
	wmNull        = 0x0000
	wmLButtonUp   = 0x0202
	wmRButtonUp   = 0x0205
	wmContextMenu = 0x007B
	wmTrayIcon    = 0x8000 + 1 // WM_APP+1: сообщения от значка

	nimAdd, nimModify, nimDelete = 0, 1, 2
	nifMessage, nifIcon, nifTip  = 0x1, 0x2, 0x4
	nifInfo                      = 0x10
	niifWarning                  = 0x2

	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020
	tpmReturnCmd   = 0x0100
	mfString       = 0x0000
	mfSeparator    = 0x0800

	smCxSmIcon = 49

	modNoRepeat = 0x4000
	hotkeyID    = 1
	watchTimer  = 1
)

type point struct{ X, Y int32 }

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	_       uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type notifyIconData struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GuidItem        [16]byte
	BalloonIcon     uintptr
}

// state — единственный значок процесса; оконная процедура Windows получает только числа,
// поэтому состояние лежит в переменной пакета.
type state struct {
	h           Handlers
	hwnd        uintptr
	icon        uintptr
	taskbarMsg  uint32
	accel       string // комбинация, которую пытались зарегистрировать
	hotkeyReady bool
	warned      bool // подсказку про занятую комбинацию уже показали
}

var cur *state

func utf16Copy(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) {
		u = append(u[:len(dst)-1], 0)
	}
	copy(dst, u)
}

// Run показывает значок в области уведомлений и слушает глобальную комбинацию клавиш, пока
// не вызван h.Quit или не закрыт канал stop (nil: без него). Левый клик по значку и
// комбинация делают снимок, правый клик открывает меню.
func Run(h Handlers, stop <-chan struct{}) error {
	runtime.LockOSThread() // окно и цикл сообщений принадлежат одному потоку
	defer runtime.UnlockOSThread()

	s := &state{h: h}
	cur = s
	userQuit := h.Quit
	s.h.Quit = func() {
		if userQuit != nil {
			userQuit()
		}
	}

	inst, _, _ := procGetModuleHandleW.Call(0)
	class, _ := syscall.UTF16PtrFromString("ScrawlTrayWindow")
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: syscall.NewCallback(wndProc), Instance: inst, ClassName: class}
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("RegisterClassEx: %w", err)
	}
	title, _ := syscall.UTF16PtrFromString("scrawl")
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, 0, 0, inst, 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx: %w", err)
	}
	s.hwnd = hwnd
	name, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(name)))
	s.taskbarMsg = uint32(r)

	s.icon = makeIcon()
	if err := s.notify(nimAdd, nifMessage|nifIcon|nifTip, "", ""); err != nil {
		procDestroyWindow.Call(hwnd)
		return err
	}
	s.syncHotkey()
	// Комбинацию меняют в окне настроек (другой процесс): раз в пару секунд сверяемся с файлом.
	procSetTimer.Call(hwnd, watchTimer, 2000, 0)

	if stop != nil {
		go func() {
			<-stop
			procPostMessageW.Call(hwnd, wmClose, 0, 0)
		}()
	}

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 { // WM_QUIT или ошибка
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	s := cur
	if s != nil && hwnd == s.hwnd {
		switch uint32(message) {
		case wmTrayIcon:
			switch uint32(lParam) & 0xFFFF {
			case wmLButtonUp:
				s.shot()
			case wmRButtonUp, wmContextMenu:
				s.showMenu()
			}
			return 0
		case wmHotkey:
			s.shot()
			return 0
		case wmTimer:
			s.syncHotkey()
			return 0
		case wmClose:
			procDestroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			procUnregisterHotKey.Call(hwnd, hotkeyID)
			s.notify(nimDelete, 0, "", "")
			if s.icon != 0 {
				procDestroyIcon.Call(s.icon)
			}
			procPostQuitMessage.Call(0)
			return 0
		}
		if s.taskbarMsg != 0 && uint32(message) == s.taskbarMsg {
			// Проводник перезапустился: значок пропал вместе с панелью, добавляем заново.
			s.notify(nimAdd, nifMessage|nifIcon|nifTip, "", "")
			return 0
		}
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func (s *state) shot() {
	if s.h.Shot != nil {
		go s.h.Shot()
	}
}

// notify добавляет, меняет или убирает значок; balloon (если не пуст) показывает всплывающую подсказку.
func (s *state) notify(op uintptr, flags uint32, balloonTitle, balloon string) error {
	nid := notifyIconData{Size: uint32(unsafe.Sizeof(notifyIconData{})), Wnd: s.hwnd, ID: 1, Flags: flags, CallbackMessage: wmTrayIcon, Icon: s.icon}
	utf16Copy(nid.Tip[:], "Scrawl")
	if balloon != "" {
		nid.Flags |= nifInfo
		nid.InfoFlags = niifWarning
		utf16Copy(nid.InfoTitle[:], balloonTitle)
		utf16Copy(nid.Info[:], balloon)
	}
	if r, _, err := procShellNotifyIconW.Call(op, uintptr(unsafe.Pointer(&nid))); r == 0 && op != nimDelete {
		return fmt.Errorf("Shell_NotifyIcon: %w", errOr(err))
	}
	return nil
}

func errOr(err error) error {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return errors.New("system call failed")
	}
	return err
}

// syncHotkey приводит зарегистрированную комбинацию в соответствие с файлом настроек.
func (s *state) syncHotkey() {
	accel := hotkey.Current()
	if accel == "" {
		accel = hotkey.DefaultWindows
	}
	if accel == s.accel && s.hotkeyReady {
		return
	}
	// Комбинацию могла держать другая программа: пока не зарегистрирована, пробуем снова.
	if accel != s.accel {
		s.warned = false
	}
	if s.hotkeyReady {
		procUnregisterHotKey.Call(s.hwnd, hotkeyID)
		s.hotkeyReady = false
	}
	s.accel = accel
	mods, vk, err := hotkey.ParseWin(accel)
	if err == nil {
		if r, _, e := procRegisterHotKey.Call(s.hwnd, hotkeyID, uintptr(mods|modNoRepeat), uintptr(vk)); r != 0 {
			s.hotkeyReady = true
			return
		} else {
			err = errOr(e)
		}
	}
	if s.warned {
		return
	}
	s.warned = true
	s.notify(nimModify, nifMessage|nifIcon|nifTip, "Scrawl",
		fmt.Sprintf(i18n.T("Комбинация %s недоступна (%v). Выберите другую в настройках: scrawl -settings"), accel, err))
}

func (s *state) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)
	for _, it := range items() {
		if it.label == "" {
			procAppendMenuW.Call(menu, mfSeparator, 0, 0)
			continue
		}
		label, _ := syscall.UTF16PtrFromString(it.label)
		procAppendMenuW.Call(menu, mfString, uintptr(it.id), uintptr(unsafe.Pointer(label)))
	}
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	// Без выхода на передний план меню не закроется кликом мимо.
	procSetForegroundWindow.Call(s.hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton|tpmBottomAlign, uintptr(pt.X), uintptr(pt.Y), 0, s.hwnd, 0)
	procPostMessageW.Call(s.hwnd, wmNull, 0, 0)
	switch int32(cmd) {
	case idShot:
		s.shot()
	case idSettings:
		if s.h.Settings != nil {
			go s.h.Settings()
		}
	case idQuit:
		if s.h.Quit != nil {
			s.h.Quit()
		}
		procDestroyWindow.Call(s.hwnd)
	}
}

// makeIcon рисует значок: синий скруглённый квадрат с белым видоискателем. Фон нужен,
// чтобы значок читался и на тёмной, и на светлой панели задач.
func makeIcon() uintptr {
	size, _, _ := procGetSystemMetrics.Call(smCxSmIcon)
	n := int(size)
	if n < 16 {
		n = 16
	}
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	bg := color.NRGBA{R: 0x25, G: 0x63, B: 0xEB, A: 0xFF}
	rad := n / 5
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if insideRounded(x, y, n, rad) {
				img.SetNRGBA(x, y, bg)
			}
		}
	}
	draw.Draw(img, img.Bounds(), drawIcon(n), image.Point{}, draw.Over)

	xor := make([]byte, 0, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := img.NRGBAAt(x, y)
			xor = append(xor, c.B, c.G, c.R, c.A)
		}
	}
	and := make([]byte, ((n+15)/16*2)*n) // маска нулевая: прозрачность берётся из альфа-канала
	icon, _, _ := procCreateIcon.Call(0, uintptr(n), uintptr(n), 1, 32, uintptr(unsafe.Pointer(&and[0])), uintptr(unsafe.Pointer(&xor[0])))
	return icon
}

// insideRounded сообщает, лежит ли пиксель внутри квадрата n×n со скруглёнными углами радиуса rad.
func insideRounded(x, y, n, rad int) bool {
	cx, cy := x, y
	switch {
	case x < rad:
		cx = rad
	case x >= n-rad:
		cx = n - rad - 1
	}
	switch {
	case y < rad:
		cy = rad
	case y >= n-rad:
		cy = n - rad - 1
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= rad*rad
}
