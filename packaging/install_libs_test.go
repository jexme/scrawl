package packaging

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerWarnsWhenBindirIsNotInPath(t *testing.T) {
	env := []string{"WAYLAND_DISPLAY=wayland-0", "LANG=en_US.UTF-8"}
	out, prefix := runInstaller(t, env, "wl-copy", "notify-send")
	if !strings.Contains(out, filepath.Join(prefix, "bin")+" is not in your PATH") {
		t.Errorf("BINDIR not in PATH, no warning:\n%s", out)
	}
	out, _ = runInstaller(t, append(env, "ON_PATH=1"), "wl-copy", "notify-send")
	if strings.Contains(out, "PATH") {
		t.Errorf("BINDIR is in PATH, but there is a warning:\n%s", out)
	}
	out, _ = runInstaller(t, []string{"WAYLAND_DISPLAY=wayland-0", "LANG=ru_RU.UTF-8"}, "wl-copy", "notify-send")
	if !strings.Contains(out, "нет в PATH") {
		t.Errorf("по-русски, вывод:\n%s", out)
	}
}

func TestInstallerReportsMissingLibraries(t *testing.T) {
	env := []string{"WAYLAND_DISPLAY=wayland-0", "LANG=en_US.UTF-8", "ON_PATH=1"}
	ldd := "ldd=\tlinux-vdso.so.1 (0x00007ffd)\n" +
		"\tlibX11-xcb.so.1 => not found\n" +
		"\tlibwayland-client.so.0 => not found\n" +
		"\tlibwayland-egl.so.1 => not found\n" +
		"\tlibEGL.so.1 => /usr/lib/libEGL.so.1 (0x1)\n" +
		"\tlibmystery.so.7 => not found"
	out, _ := runInstaller(t, env, "wl-copy", "notify-send", ldd)
	for _, want := range []string{
		"Shared libraries are missing: libX11-xcb.so.1 libwayland-client.so.0 libwayland-egl.so.1 libmystery.so.7",
		"sudo apt install libx11-xcb1 libwayland-client0 libwayland-egl1",
		"sudo dnf install libX11-xcb libwayland-client libwayland-egl",
		"sudo pacman -S libx11 wayland",
		"provides: libmystery.so.7",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in the output:\n%s", want, out)
		}
	}
	// Nothing is missing: ldd lists only found libraries.
	out, _ = runInstaller(t, env, "wl-copy", "notify-send", "ldd=\tlibc.so.6 => /lib/libc.so.6 (0x1)")
	if strings.Contains(out, "librar") || strings.Contains(out, "apt install") {
		t.Errorf("all libraries are present, but there is a warning:\n%s", out)
	}
	// Russian.
	out, _ = runInstaller(t, []string{"WAYLAND_DISPLAY=wayland-0", "LANG=ru_RU.UTF-8", "ON_PATH=1"}, "wl-copy", "notify-send", "ldd=\tlibXfixes.so.3 => not found")
	if !strings.Contains(out, "Не хватает библиотек: libXfixes.so.3") || !strings.Contains(out, "sudo apt install libxfixes3") {
		t.Errorf("по-русски, вывод:\n%s", out)
	}
}
