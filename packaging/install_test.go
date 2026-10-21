// Package packaging tests the installer from the release archive.
package packaging

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runInstaller runs the real install.sh from a temporary folder holding fake scrawl and
// .desktop files, with a PATH made only of the listed programs. It returns the output and
// the install folder.
func runInstaller(t *testing.T, env []string, tools ...string) (out string, prefix string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh — установщик для Linux: ему нужны симлинки и sh")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("нет sh")
	}
	src, err := os.ReadFile("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write := func(name, body string, mode os.FileMode) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("install.sh", string(src), 0o755)
	write("scrawl", "#!/bin/sh\n", 0o755)
	write("io.github.jexme.Scrawl.desktop", "[Desktop Entry]\nExec=scrawl\n", 0o644)

	// PATH: the utilities the script itself needs (install, sed, chmod, dirname), plus the
	// listed programs.
	bin := t.TempDir()
	for _, name := range []string{"install", "sed", "chmod", "dirname", "mkdir", "cat"} {
		p, err := exec.LookPath(name)
		if err != nil {
			t.Skipf("нет %s", name)
		}
		if err := os.Symlink(p, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range tools {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	prefix = t.TempDir()
	cmd := exec.Command(sh, filepath.Join(dir, "install.sh"))
	cmd.Env = append([]string{"PATH=" + bin, "PREFIX=" + prefix, "HOME=" + prefix}, env...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh: %v\n%s", err, b)
	}
	return string(b), prefix
}

func TestInstallerReportsMissingDependencies(t *testing.T) {
	wayland := []string{"WAYLAND_DISPLAY=wayland-0", "DISPLAY=:0", "LANG=en_US.UTF-8"}
	x11 := []string{"WAYLAND_DISPLAY=", "DISPLAY=:0", "LANG=en_US.UTF-8"}

	// The installation itself succeeds, the binary and .desktop are in place.
	out, prefix := runInstaller(t, wayland)
	if _, err := os.Stat(filepath.Join(prefix, "bin", "scrawl")); err != nil {
		t.Errorf("бинарник не установлен: %v", err)
	}
	desktop, err := os.ReadFile(filepath.Join(prefix, "share", "applications", "io.github.jexme.Scrawl.desktop"))
	if err != nil || !strings.Contains(string(desktop), "Exec="+filepath.Join(prefix, "bin", "scrawl")) {
		t.Errorf(".desktop: %q %v", desktop, err)
	}

	// Wayland without the programs: asks for wl-clipboard and libnotify-bin in a single command.
	if !strings.Contains(out, "sudo apt install wl-clipboard libnotify-bin") {
		t.Errorf("Wayland без программ, вывод:\n%s", out)
	}
	// Wayland, wl-copy is present, notify-send is not.
	out, _ = runInstaller(t, wayland, "wl-copy")
	if !strings.Contains(out, "sudo apt install libnotify-bin") || strings.Contains(out, "wl-clipboard") {
		t.Errorf("нет только notify-send, вывод:\n%s", out)
	}
	// Everything is present: no warning.
	out, _ = runInstaller(t, wayland, "wl-copy", "notify-send")
	if strings.Contains(out, "apt install") || strings.Contains(out, "missing") {
		t.Errorf("всё установлено, а предупреждение есть:\n%s", out)
	}
	// X11: xclip is needed instead of wl-clipboard, wl-copy is not required.
	out, _ = runInstaller(t, x11, "notify-send")
	if !strings.Contains(out, "sudo apt install xclip") || strings.Contains(out, "wl-clipboard") {
		t.Errorf("X11 без xclip, вывод:\n%s", out)
	}
	out, _ = runInstaller(t, x11, "xclip", "notify-send")
	if strings.Contains(out, "apt install") {
		t.Errorf("X11 со всеми программами, а предупреждение есть:\n%s", out)
	}
	// Russian system language.
	out, _ = runInstaller(t, []string{"WAYLAND_DISPLAY=wayland-0", "LANG=ru_RU.UTF-8"})
	if !strings.Contains(out, "Не хватает программ") || !strings.Contains(out, "sudo apt install wl-clipboard libnotify-bin") {
		t.Errorf("по-русски, вывод:\n%s", out)
	}
}
