// Package autostart включает и выключает запуск значка в трее при входе в
// систему: файл .desktop в ~/.config/autostart/ (стандарт XDG Autostart).
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

// FileName — имя файла автозапуска; совпадает с AppID.
const FileName = "io.github.jexme.Scrawl.desktop"

// Path возвращает путь к файлу автозапуска.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "autostart", FileName)
}

// Enabled сообщает, есть ли файл автозапуска.
func Enabled() bool {
	if useRegistry {
		return regEnabled()
	}
	_, err := os.Stat(Path())
	return err == nil
}

// quoteExec записывает путь для поля Exec: пути с пробелами и служебными
// знаками берутся в двойные кавычки по правилам Desktop Entry Specification.
func quoteExec(path string) string {
	if !strings.ContainsAny(path, " \t\n\"'\\><~|&;$*?#()`") {
		return path
	}
	// Сначала кавычечное экранирование (\" \$ \` \\), затем обычное для строк
	// .desktop, где обратная косая удваивается: отсюда \\" и четыре косые для одной.
	r := strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", `$`, `\\$`)
	return `"` + r.Replace(path) + `"`
}

// Content возвращает содержимое файла автозапуска для бинарника bin.
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

// Enable создаёт файл автозапуска для бинарника bin.
func Enable(bin string) error {
	if !filepath.IsAbs(bin) {
		return fmt.Errorf(i18n.T("the path to the program must be absolute: %q"), bin)
	}
	if useRegistry {
		return regEnable(bin)
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(Content(bin)), 0o644)
}

// Disable удаляет файл автозапуска; если его нет, это не ошибка.
func Disable() error {
	if useRegistry {
		return regDisable()
	}
	if err := os.Remove(Path()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
