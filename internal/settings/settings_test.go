package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	want := Default()
	want.DoubleClick, want.Enter, want.SaveDir = DoubleClickCopySave, EnterSave, "/tmp/shots"
	want.Notifications, want.DimPercent, want.BorderColor = false, 40, "#FF8800"
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
	if err := Save(func() Settings { d := Default(); d.Notifications = false; return d }()); err != nil {
		t.Fatal(err)
	}
	if Load().Notifications {
		t.Error("выключенные уведомления должны сохраняться как false")
	}
}

func TestFilePrefix(t *testing.T) {
	sandbox(t)
	d := Default()
	if got := d.FileName("20260101-120000"); got != "screen-editor-20260101-120000.png" {
		t.Errorf("имя по умолчанию: %q", got)
	}
	d.FilePrefix = "Скрин"
	if got := d.NumberedFileName("20260101-120000", 2); got != "Скрин-20260101-120000-2.png" {
		t.Errorf("имя с префиксом: %q", got)
	}
	for _, ok := range []string{"", "shot", "Мой скрин 1", "a.b-c_d"} {
		if err := ValidateFilePrefix(ok); err != nil {
			t.Errorf("%q должен быть допустим: %v", ok, err)
		}
	}
	for _, bad := range []string{"a/b", `a\b`, "a\nb", "../x", strings.Repeat("я", 61)} {
		if ValidateFilePrefix(bad) == nil {
			t.Errorf("%q должен быть отклонён", bad)
		}
	}
	// Неверный префикс в файле сбрасывается, остальное читается.
	writeConfig(t, `{"file_prefix":"a/b","save_dir":"/x"}`)
	if got := Load(); got.FilePrefix != "" || got.SaveDir != "/x" {
		t.Errorf("неверный префикс должен сброситься: %+v", got)
	}
}

func TestDimPercentClamped(t *testing.T) {
	sandbox(t)
	if got := Load().DimPercent; got != 86 {
		t.Errorf("затемнение по умолчанию 86%%, а не %d", got)
	}
	for body, want := range map[string]int{
		`{"dim_percent":40}`:   40,
		`{"dim_percent":0}`:    0,
		`{"dim_percent":150}`:  100,
		`{"dim_percent":-5}`:   0,
		`{"dim_percent":"x"}`:  86,
		`{"dim_percent":12.5}`: 86, // не целое: значение по умолчанию
	} {
		writeConfig(t, body)
		if got := Load().DimPercent; got != want {
			t.Errorf("%s: получили %d, ожидали %d", body, got, want)
		}
	}
}

func TestBorderSettings(t *testing.T) {
	sandbox(t)
	d := Load()
	if d.BorderWidth != 1 || d.BorderDash != 4.6 || d.BorderGap != 2 || d.BorderColor != "#FFFFFF" {
		t.Errorf("рамка по умолчанию: %+v", d)
	}
	writeConfig(t, `{"border_width":99,"border_dash":0,"border_gap":-3,"border_color":"ff8800"}`)
	got := Load()
	if got.BorderWidth != MaxBorderWidth || got.BorderDash != MinBorderDash || got.BorderGap != 0 || got.BorderColor != "#FF8800" {
		t.Errorf("числа подрезаются, цвет приводится к #RRGGBB: %+v", got)
	}
	if c := got.BorderNRGBA(); c.R != 255 || c.G != 0x88 || c.B != 0 || c.A != 255 {
		t.Errorf("цвет рамки: %+v", c)
	}
	for _, bad := range []string{"", "#12345", "#GG0000", "red", "#1234567"} {
		writeConfig(t, `{"border_color":"`+bad+`"}`)
		if got := Load().BorderColor; got != "#FFFFFF" {
			t.Errorf("цвет %q должен стать белым по умолчанию, а стал %q", bad, got)
		}
	}
	if _, err := ParseColor("#abc"); err == nil {
		t.Error("короткая запись #RGB не поддерживается")
	}
}

func TestStoreUpdateWithoutChangeDoesNotWrite(t *testing.T) {
	sandbox(t)
	st := NewStore()
	if err := st.Update(func(s *Settings) { s.DimPercent = Default().DimPercent }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Path()); !os.IsNotExist(err) {
		t.Error("Update без изменений не должен создавать файл")
	}
	if err := st.Update(func(s *Settings) { s.DimPercent = 10 }); err != nil {
		t.Fatal(err)
	}
	if Load().DimPercent != 10 {
		t.Error("настоящее изменение должно записаться")
	}
}

func TestHandleRadius(t *testing.T) {
	sandbox(t)
	if got := Load().HandleRadius; got != 3 {
		t.Errorf("радиус ручек по умолчанию 3, а не %v", got)
	}
	for body, want := range map[string]float64{
		`{"handle_radius":0}`:   0, // ручки спрятаны
		`{"handle_radius":6.5}`: 6.5,
		`{"handle_radius":100}`: MaxHandleRadius,
		`{"handle_radius":-1}`:  0,
		`{"handle_radius":"x"}`: 3,
	} {
		writeConfig(t, body)
		if got := Load().HandleRadius; got != want {
			t.Errorf("%s: получили %v, ожидали %v", body, got, want)
		}
	}
}

func TestShowSizeDefaultOn(t *testing.T) {
	sandbox(t)
	if !Load().ShowSize {
		t.Error("подпись размера по умолчанию показывается")
	}
	writeConfig(t, `{"show_size":false}`)
	if Load().ShowSize {
		t.Error("show_size=false должен выключать подпись")
	}
}
