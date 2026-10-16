package settings

import (
	"encoding/json"
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
	want := Settings{DoubleClick: DoubleClickCopySave, Enter: EnterSave, SaveDir: "/tmp/shots"}
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

func writeConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEachKeyIsReadIndependently(t *testing.T) {
	sandbox(t)

	writeConfig(t, `{}`)
	if got := Load(); got != Default() {
		t.Errorf("пустой объект должен дать значения по умолчанию: %+v", got)
	}
	writeConfig(t, `{"save_dir":"/x"}`)
	if got := Load(); got.DoubleClick != DoubleClickOff || got.SaveDir != "/x" {
		t.Errorf("отсутствующий ключ должен получить значение по умолчанию: %+v", got)
	}
	// Неверный тип одного ключа не отбрасывает остальные.
	writeConfig(t, `{"double_click":5,"save_dir":"/x"}`)
	if got := Load(); got.DoubleClick != DoubleClickOff || got.SaveDir != "/x" {
		t.Errorf("число вместо строки: ожидали off и /x, получили %+v", got)
	}
	writeConfig(t, `{"double_click":"copy","save_dir":["a"]}`)
	if got := Load(); got.DoubleClick != DoubleClickCopy || got.SaveDir != "" {
		t.Errorf("массив вместо строки: ожидали copy и пустую папку, получили %+v", got)
	}
	// Относительный путь отклоняется так же, как в окне настроек.
	writeConfig(t, `{"double_click":"copy","save_dir":"relative/dir"}`)
	if got := Load(); got.DoubleClick != DoubleClickCopy || got.SaveDir != "" {
		t.Errorf("относительная папка должна сброситься: %+v", got)
	}
	// Не объект: всё по умолчанию.
	for _, body := range []string{`[]`, `null`, `"text"`, ``} {
		writeConfig(t, body)
		if got := Load(); got != Default() {
			t.Errorf("%q: ожидали значения по умолчанию, получили %+v", body, got)
		}
	}
}

func readKeys(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("файл настроек не JSON: %v\n%s", err, data)
	}
	return m
}

func TestUnknownKeysSurviveSave(t *testing.T) {
	sandbox(t)
	writeConfig(t, `{"double_click":"copy","future_option":{"a":[1,2]},"note":"hi"}`)

	st := NewStore()
	if err := st.Update(func(s *Settings) { s.DoubleClick = DoubleClickCopySave }); err != nil {
		t.Fatal(err)
	}
	m := readKeys(t)
	if string(m["double_click"]) != `"copy_save"` {
		t.Errorf("свой ключ не обновился: %s", m["double_click"])
	}
	if string(m["note"]) != `"hi"` {
		t.Errorf("чужой ключ потерян: %s", m["note"])
	}
	var future struct{ A []int }
	if err := json.Unmarshal(m["future_option"], &future); err != nil || len(future.A) != 2 {
		t.Errorf("чужой вложенный ключ испорчен: %s (%v)", m["future_option"], err)
	}
}

func TestEmptyOptionalKeyIsRemoved(t *testing.T) {
	sandbox(t)
	writeConfig(t, `{"double_click":"off","save_dir":"/old"}`)
	if err := Save(Settings{DoubleClick: DoubleClickOff}); err != nil {
		t.Fatal(err)
	}
	if _, has := readKeys(t)["save_dir"]; has {
		t.Error("пустая папка не должна оставлять в файле старое значение")
	}
	if got := Load().SaveDir; got != "" {
		t.Errorf("после сброса папка %q", got)
	}
}

func TestBrokenFileIsBackedUpBeforeSave(t *testing.T) {
	sandbox(t)
	writeConfig(t, `{oops, "my": "edit"`)
	if err := Save(Settings{DoubleClick: DoubleClickCopy}); err != nil {
		t.Fatal(err)
	}
	if got := Load().DoubleClick; got != DoubleClickCopy {
		t.Errorf("новый файл не записан: %q", got)
	}
	bak, err := os.ReadFile(Path() + ".bak")
	if err != nil || string(bak) != `{oops, "my": "edit"` {
		t.Errorf("копия битого файла не сохранена: %q, %v", bak, err)
	}
}

func TestEnterDefaultsAndTargets(t *testing.T) {
	sandbox(t)
	if got := Load().Enter; got != EnterCopySave {
		t.Errorf("по умолчанию Enter копирует и сохраняет, а не %q", got)
	}
	writeConfig(t, `{"enter":"explode"}`)
	if got := Load().Enter; got != EnterCopySave {
		t.Errorf("неверное значение должно стать значением по умолчанию: %q", got)
	}
	cases := []struct {
		e         Enter
		clip, fil bool
	}{{EnterCopySave, true, true}, {EnterCopy, true, false}, {EnterSave, false, true}}
	for _, c := range cases {
		if clip, file := c.e.Targets(); clip != c.clip || file != c.fil {
			t.Errorf("%s: буфер=%v файл=%v, ожидали %v и %v", c.e, clip, file, c.clip, c.fil)
		}
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

func TestNotificationsDefaultOnAndToggles(t *testing.T) {
	sandbox(t)
	if !Load().Notifications {
		t.Error("по умолчанию уведомления включены")
	}
	writeConfig(t, `{"notifications":"no"}`)
	if !Load().Notifications {
		t.Error("значение неверного типа должно дать значение по умолчанию (включено)")
	}
	if err := Save(Settings{DoubleClick: DoubleClickOff, Enter: EnterCopySave, Notifications: false}); err != nil {
		t.Fatal(err)
	}
	if Load().Notifications {
		t.Error("выключенные уведомления должны сохраняться как false")
	}
}
