package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func sandbox(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestDefaultsWithoutFile(t *testing.T) {
	sandbox(t)
	if got := Load(); got.DoubleClick != DoubleClickOff || got.SaveDir != "" {
		t.Errorf("без файла ожидали настройки по умолчанию, получили %+v", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	sandbox(t)
	want := Settings{DoubleClick: DoubleClickCopySave, SaveDir: "/tmp/shots"}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	if got := Load(); got != want {
		t.Errorf("после записи получили %+v, ожидали %+v", got, want)
	}
	// Временные файлы не остаются.
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(Path()), "*.tmp"))
	if len(left) != 0 {
		t.Errorf("остались временные файлы: %v", left)
	}
}

func TestInvalidValuesFallBack(t *testing.T) {
	sandbox(t)
	os.MkdirAll(filepath.Dir(Path()), 0o755)

	// Неверное значение заменяется, остальное сохраняется.
	os.WriteFile(Path(), []byte(`{"double_click":"explode","save_dir":"/x"}`), 0o644)
	if got := Load(); got.DoubleClick != DoubleClickOff || got.SaveDir != "/x" {
		t.Errorf("неверный double_click должен стать off, остальное остаться: %+v", got)
	}
	// Битый JSON: всё по умолчанию.
	os.WriteFile(Path(), []byte(`{oops`), 0o644)
	if got := Load(); got != Default() {
		t.Errorf("битый файл должен дать настройки по умолчанию: %+v", got)
	}
}

func TestStoreUpdatePersists(t *testing.T) {
	sandbox(t)
	st := NewStore()
	if err := st.Update(func(s *Settings) { s.DoubleClick = DoubleClickCopy }); err != nil {
		t.Fatal(err)
	}
	if st.Get().DoubleClick != DoubleClickCopy {
		t.Error("изменение не применилось в памяти")
	}
	if got := NewStore().Get().DoubleClick; got != DoubleClickCopy {
		t.Errorf("изменение не дошло до диска: %q", got)
	}
}

func TestValidateAndExpandDir(t *testing.T) {
	for _, ok := range []string{"", "/abs/path", "~", "~/Pictures/Shots"} {
		if err := ValidateSaveDir(ok); err != nil {
			t.Errorf("%q должен быть допустим: %v", ok, err)
		}
	}
	for _, bad := range []string{"relative/path", "Pictures", "./x", "~user/x"} {
		if err := ValidateSaveDir(bad); err == nil {
			t.Errorf("%q должен быть отклонён", bad)
		}
	}

	t.Setenv("HOME", "/home/tester")
	cases := map[string]string{
		"":                 "",
		"/abs/path":        "/abs/path",
		"~":                "/home/tester",
		"~/Pictures/Shots": "/home/tester/Pictures/Shots",
	}
	for in, want := range cases {
		got, err := ExpandDir(in)
		if err != nil || got != want {
			t.Errorf("ExpandDir(%q) = %q, %v; ожидали %q", in, got, err, want)
		}
	}
}
