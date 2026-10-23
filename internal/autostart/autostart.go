// Package autostart enables and disables launching the tray icon at login:
// a .desktop file in ~/.config/autostart/ (the XDG Autostart standard).
package autostart

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jexme/scrawl/internal/i18n"
)

// FileName is the name of the autostart file; it matches AppID.
const FileName = "io.github.jexme.Scrawl.desktop"

// Path returns the path to the autostart file.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "autostart", FileName)
}

// Enabled reports whether the autostart file exists.
func Enabled() bool {
	if useRegistry {
		return regEnabled()
	}
	_, err := os.Stat(Path())
	return err == nil
}

// quoteExec writes a path for the Exec field: paths with spaces and special
// characters are put in double quotes per the Desktop Entry Specification.
func quoteExec(path string) string {
	if !strings.ContainsAny(path, " \t\n\"'\\><~|&;$*?#()`") {
		return path
	}
	// First the quote escaping (\" \$ \` \\), then the usual one for .desktop strings,
	// where the backslash is doubled: hence \\" and four backslashes for one.
	r := strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", `$`, `\\$`)
	return `"` + r.Replace(path) + `"`
}

// Content returns the contents of the autostart file for the binary bin.
func Content(bin string) string {
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Scrawl\n" +
		"Comment=Screenshot tray icon\n" +
		"Exec=" + quoteExec(bin) + " -tray\n" +
		"Icon=applets-screenshooter\n" +
		"Terminal=false\n" +
		"NoDisplay=true\n" +
		"X-GNOME-Autostart-enabled=true\n"
}

// Enable creates the autostart file for the binary bin.
func Enable(bin string) error {
	if !filepath.IsAbs(bin) {
		return fmt.Errorf(i18n.T("the path to the program must be absolute: %q"), bin)
	}
	if useRegistry {
		// A program in %TEMP% (for example run straight from an opened zip) would leave a
		// dead entry behind as soon as the temporary files are cleaned.
		if inTempDir(bin, os.TempDir()) {
			return fmt.Errorf(i18n.T("autostart was not set: the program is in a temporary folder (%s); move it to a permanent folder first"), filepath.Dir(bin))
		}
		return regEnable(bin)
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(Content(bin)), 0o644)
}

// Heal repairs an enabled autostart after the program was moved or updated in place: if the
// entry exists but starts another path than bin, it is rewritten to bin. A disabled autostart
// is never switched on, and a bin inside a temporary folder is never written. Only the
// Windows registry entry is healed (it reports whether it changed anything); the XDG file
// of Linux is managed by the packages.
func Heal(bin string) (bool, error) {
	if !useRegistry || !filepath.IsAbs(bin) {
		return false, nil
	}
	return regHeal(bin)
}

// Disable removes the autostart file; if it does not exist, that is not an error.
func Disable() error {
	if useRegistry {
		return regDisable()
	}
	if err := os.Remove(Path()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
