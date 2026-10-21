package hotkey

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jexme/scrawl/internal/i18n"
)

// onWindows chooses the combination storage: gsettings or a file. Tests replace it.
var onWindows = runtime.GOOS == "windows"

// DefaultWindows is the default combination on Windows.
const DefaultWindows = "Print"

// RegisterHotKey modifiers.
const (
	ModAlt     = 0x1
	ModControl = 0x2
	ModShift   = 0x4
	ModWin     = 0x8
)

// WinFile returns the path to the file with the combination: Windows has no GNOME
// shortcuts, so the program stores the combination itself, and the tray icon registers it
// through RegisterHotKey.
func WinFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "scrawl", "hotkey")
}

func winCurrent() string {
	b, err := os.ReadFile(WinFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func winAssign(accel string) (Result, error) {
	var res Result
	if err := ValidateAccel(accel); err != nil {
		return res, err
	}
	if _, _, err := ParseWin(accel); err != nil {
		return res, err
	}
	res.Updated = winCurrent() != ""
	if err := os.MkdirAll(filepath.Dir(WinFile()), 0o755); err != nil {
		return res, err
	}
	return res, os.WriteFile(WinFile(), []byte(accel+"\n"), 0o644)
}

var winKeys = map[string]uint32{
	"print": 0x2C, "pause": 0x13, "scroll_lock": 0x91, "insert": 0x2D, "delete": 0x2E,
	"home": 0x24, "end": 0x23, "page_up": 0x21, "prior": 0x21, "page_down": 0x22, "next": 0x22,
	"space": 0x20, "return": 0x0D, "tab": 0x09, "escape": 0x1B, "backspace": 0x08,
	"left": 0x25, "up": 0x26, "right": 0x27, "down": 0x28,
}

// ParseWin parses a GTK combination (Print, <Control><Alt>a, <Super>F9) into modifiers and
// a virtual key code for RegisterHotKey.
func ParseWin(accel string) (mods, vk uint32, err error) {
	rest := strings.TrimSpace(accel)
	for strings.HasPrefix(rest, "<") {
		end := strings.Index(rest, ">")
		if end < 0 {
			return 0, 0, errors.New(i18n.T("use the GNOME format: Print, <Super>s, <Control><Alt>p"))
		}
		switch strings.ToLower(rest[1:end]) {
		case "control", "ctrl", "primary":
			mods |= ModControl
		case "alt", "mod1":
			mods |= ModAlt
		case "shift":
			mods |= ModShift
		case "super":
			mods |= ModWin
		default:
			return 0, 0, fmt.Errorf(i18n.T("unknown modifier %s"), rest[:end+1])
		}
		rest = rest[end+1:]
	}
	name := strings.ToLower(rest)
	switch {
	case len(name) == 1 && name[0] >= 'a' && name[0] <= 'z':
		vk = uint32(name[0] - 'a' + 'A')
	case len(name) == 1 && name[0] >= '0' && name[0] <= '9':
		vk = uint32(name[0])
	case len(name) >= 2 && name[0] == 'f':
		var n int
		if _, err := fmt.Sscanf(name, "f%d", &n); err == nil && n >= 1 && n <= 24 && fmt.Sprintf("f%d", n) == name {
			vk = uint32(0x70 + n - 1)
		}
	default:
		vk = winKeys[name]
	}
	if vk == 0 {
		return 0, 0, fmt.Errorf(i18n.T("key %q is not supported"), rest)
	}
	return mods, vk, nil
}
