package hotkey

import (
	"testing"

	"github.com/jexme/scrawl/internal/testenv"
)

func TestParseWin(t *testing.T) {
	for _, c := range []struct {
		accel     string
		mods, vk  uint32
		wantError bool
	}{
		{"Print", 0, 0x2C, false},
		{"<Control><Alt>a", ModControl | ModAlt, 'A', false},
		{"<Super>s", ModWin, 'S', false},
		{"<Primary><Shift>F9", ModControl | ModShift, 0x78, false},
		{"<Control>5", ModControl, '5', false},
		{"<Hyper>x", 0, 0, true},
		{"<Control>", 0, 0, true},
		{"F25", 0, 0, true},
		{"Unknown", 0, 0, true},
	} {
		mods, vk, err := ParseWin(c.accel)
		if (err != nil) != c.wantError || mods != c.mods || vk != c.vk {
			t.Errorf("ParseWin(%q) = %#x, %#x, %v", c.accel, mods, vk, err)
		}
	}
}

func TestWindowsAssignKeepsAccelInFile(t *testing.T) {
	testenv.Config(t, t.TempDir())
	old := onWindows
	onWindows = true
	t.Cleanup(func() { onWindows = old })

	if got := Current(); got != "" {
		t.Fatalf("без файла комбинации нет, а есть %q", got)
	}
	if res, err := Assign("<Control><Alt>a", ""); err != nil || res.Updated {
		t.Fatalf("первое назначение: %+v, %v", res, err)
	}
	if got := Current(); got != "<Control><Alt>a" {
		t.Errorf("после назначения %q", got)
	}
	if res, err := Assign("Print", ""); err != nil || !res.Updated {
		t.Errorf("повторное назначение должно обновлять: %+v, %v", res, err)
	}
	if _, err := Assign("<Hyper>x", ""); err == nil {
		t.Error("неизвестный модификатор должен отклоняться")
	}
	if got := Current(); got != "Print" {
		t.Errorf("отклонённая комбинация не должна менять файл: %q", got)
	}
}
