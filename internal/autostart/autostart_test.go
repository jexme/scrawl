package autostart

import (
	"github.com/jexme/scrawl/internal/testenv"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func sandbox(t *testing.T) {
	t.Helper()
	testenv.Config(t, t.TempDir())
	setRegistry(t, false) // file branch: the tests do not touch the real registry
}

// bin: an absolute path to the program on any platform (on Windows a drive letter is needed).
var bin = func() string {
	if runtime.GOOS == "windows" {
		return `C:/home/me/.local/bin/scrawl`
	}
	return "/home/me/.local/bin/scrawl"
}()

func TestEnableDisable(t *testing.T) {
	sandbox(t)
	if Enabled() {
		t.Fatal("в чистой папке автозапуска нет")
	}
	if err := Disable(); err != nil {
		t.Errorf("выключить выключенный автозапуск не ошибка: %v", err)
	}
	if err := Enable(bin); err != nil {
		t.Fatal(err)
	}
	if !Enabled() || filepath.Base(Path()) != "io.github.jexme.Scrawl.desktop" || filepath.Base(filepath.Dir(Path())) != "autostart" {
		t.Errorf("файл автозапуска: %s", Path())
	}
	data, _ := os.ReadFile(Path())
	for _, want := range []string{"[Desktop Entry]", "Type=Application", "Exec=" + bin + " -tray\n", "X-GNOME-Autostart-enabled=true"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("в файле нет %q:\n%s", want, data)
		}
	}
	if err := Enable("scrawl"); err == nil {
		t.Error("относительный путь должен отклоняться")
	}
	if err := Disable(); err != nil || Enabled() {
		t.Errorf("после Disable: %v, включён=%v", err, Enabled())
	}
}

func TestQuoteExec(t *testing.T) {
	cases := map[string]string{
		"/usr/bin/scrawl":         "/usr/bin/scrawl",
		"/home/me/my apps/scrawl": `"/home/me/my apps/scrawl"`,
		`/tmp/a"b`:                `"/tmp/a\\"b"`,
		"/tmp/$x":                 `"/tmp/\\$x"`,
		`/tmp/a\b`:                `"/tmp/a\\\\b"`,
	}
	for in, want := range cases {
		if got := quoteExec(in); got != want {
			t.Errorf("quoteExec(%q) = %q, ожидали %q", in, got, want)
		}
	}
	if !strings.Contains(Content("/a b/scrawl"), `Exec="/a b/scrawl" -tray`) {
		t.Error("путь с пробелом должен быть в кавычках")
	}
}
