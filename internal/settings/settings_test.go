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
	if got := d.FileName("20260101-120000"); got != "scrawl-20260101-120000.png" {
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

func TestToolDefaults(t *testing.T) {
	sandbox(t)
	got := Load()
	if got.DefaultTool != "arrow" || got.Tools != (ToolStyles{}) {
		t.Errorf("по умолчанию после выбора области включается стрелка, стили пустые: %+v", got)
	}

	// Неверное значение одного поля не портит соседние, в том числе вложенные.
	writeConfig(t, `{"default_tool":"pen","tools":{
		"pen":{"color":"e53935","width":"толстая"},
		"arrow":{"color":"red","width":99},
		"text":{"width":6},
		"marker":5}}`)
	got = Load()
	if got.DefaultTool != "pen" {
		t.Errorf("default_tool: %q", got.DefaultTool)
	}
	if got.Tools.Pen != (ToolStyle{Color: "#E53935"}) {
		t.Errorf("перо: цвет должен остаться, неверная толщина пропасть: %+v", got.Tools.Pen)
	}
	if got.Tools.Arrow != (ToolStyle{Width: MaxToolWidth}) {
		t.Errorf("стрелка: неверный цвет сбрасывается, толщина подрезается: %+v", got.Tools.Arrow)
	}
	if got.Tools.Text.Width != 6 || got.Tools.Marker != (ToolStyle{}) {
		t.Errorf("текст и маркер: %+v / %+v", got.Tools.Text, got.Tools.Marker)
	}

	writeConfig(t, `{"default_tool":"laser"}`)
	if got := Load().DefaultTool; got != "arrow" {
		t.Errorf("неизвестный инструмент должен стать стрелкой (значением по умолчанию): %q", got)
	}
}

func TestToolStylesSaveOnlyWhatIsSet(t *testing.T) {
	sandbox(t)
	st := NewStore()
	if err := st.Update(func(s *Settings) { s.Tools.Rect = ToolStyle{Color: "#43A047", Width: 3} }); err != nil {
		t.Fatal(err)
	}
	var tools map[string]ToolStyle
	if err := json.Unmarshal(readKeys(t)["tools"], &tools); err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools["rect"] != (ToolStyle{Color: "#43A047", Width: 3}) {
		t.Errorf("в файл должен попасть только заданный инструмент: %v", tools)
	}
	if got := NewStore().Get().Tools.Rect; got.Color != "#43A047" || got.Width != 3 {
		t.Errorf("после перезагрузки: %+v", got)
	}
}

func TestPalette(t *testing.T) {
	sandbox(t)
	if got := Load().Palette; got != DefaultPalette {
		t.Errorf("палитра по умолчанию: %v", got)
	}
	writeConfig(t, `{"palette":["ff0000","","nope","#00ff00"]}`)
	got := Load().Palette
	if got[0] != "#FF0000" || got[1] != DefaultPalette[1] || got[2] != DefaultPalette[2] || got[3] != "#00FF00" || got[7] != DefaultPalette[7] {
		t.Errorf("пустые и неверные цвета заменяются, остальные нормализуются: %v", got)
	}
	// Палитра не того типа: целиком по умолчанию, соседние ключи целы.
	writeConfig(t, `{"palette":"red","dim_percent":30}`)
	if got := Load(); got.Palette != DefaultPalette || got.DimPercent != 30 {
		t.Errorf("неверный тип палитры: %+v", got)
	}
	if c := Load().PaletteColors()[4]; c.R != 30 || c.G != 136 || c.B != 229 {
		t.Errorf("пятый цвет по умолчанию: %+v", c)
	}
}

func TestMarkerOpacity(t *testing.T) {
	sandbox(t)
	if got := Load(); got.MarkerOpacity != 43 || got.MarkerAlpha() != 110 {
		t.Errorf("по умолчанию 43%% (альфа 110): %d%%, альфа %d", got.MarkerOpacity, got.MarkerAlpha())
	}
	for body, want := range map[string]int{
		`{"marker_opacity":100}`: 100,
		`{"marker_opacity":0}`:   MinMarkerOpacity,
		`{"marker_opacity":500}`: 100,
		`{"marker_opacity":"x"}`: 43,
	} {
		writeConfig(t, body)
		if got := Load().MarkerOpacity; got != want {
			t.Errorf("%s: получили %d, ожидали %d", body, got, want)
		}
	}
	if a := (Settings{MarkerOpacity: 100}).MarkerAlpha(); a != 255 {
		t.Errorf("100%% это альфа 255, а не %d", a)
	}
}

func TestGlueLabels(t *testing.T) {
	sandbox(t)
	g := Load().Glue
	if got := g.Labels(2); got[0] != "Было" || got[1] != "Стало" {
		t.Errorf("для двух по умолчанию: %v", got)
	}
	if got := g.Labels(3); got[0] != "Шаг 1" || got[2] != "Шаг 3" {
		t.Errorf("для трёх по умолчанию: %v", got)
	}

	writeConfig(t, `{"glue":{"before":"До","after":"После","step":"Этап"}}`)
	g = Load().Glue
	if got := g.Labels(2); got[0] != "До" || got[1] != "После" {
		t.Errorf("свои слова для двух: %v", got)
	}
	if got := g.Labels(4); got[0] != "Этап 1" || got[3] != "Этап 4" {
		t.Errorf("своё слово для четырёх: %v", got)
	}

	// Неверное значение одной подписи сбрасывает только её.
	writeConfig(t, `{"glue":{"before":"a\nb","after":"После","step":5}}`)
	g = Load().Glue
	if got := g.Labels(2); got[0] != "Было" || got[1] != "После" {
		t.Errorf("неверная подпись сбрасывается, соседняя остаётся: %v", got)
	}
	if got := g.Labels(3); got[0] != "Шаг 1" {
		t.Errorf("шаг неверного типа: %v", got)
	}
	if ValidateGlueLabel(strings.Repeat("я", 31)) == nil {
		t.Error("31 знак — слишком длинная подпись")
	}
}

func TestGlueLayout(t *testing.T) {
	sandbox(t)
	if got := Load().Glue.Layout; got != GlueSide {
		t.Errorf("по умолчанию картинки рядом: %q", got)
	}
	writeConfig(t, `{"glue":{"layout":"stack"}}`)
	if got := Load().Glue.Layout; got != GlueStack {
		t.Errorf("stack должен читаться: %q", got)
	}
	writeConfig(t, `{"glue":{"layout":"diagonal","before":"До"}}`)
	if got := Load().Glue; got.Layout != GlueSide || got.Before != "До" {
		t.Errorf("неверное расположение сбрасывается, подписи остаются: %+v", got)
	}
}

func TestGlueLook(t *testing.T) {
	sandbox(t)
	g := Load().Glue
	if g.Background != GlueLight || g.Pad != 28 || g.LabelSize != 28 {
		t.Errorf("вид склейки по умолчанию: %+v", g)
	}
	writeConfig(t, `{"glue":{"background":"dark","pad":500,"label_size":1}}`)
	g = Load().Glue
	if g.Background != GlueDark || g.Pad != MaxGluePad || g.LabelSize != MinGlueLabelSize {
		t.Errorf("тёмный фон, числа подрезаются: %+v", g)
	}
	writeConfig(t, `{"glue":{"background":"red","pad":0}}`)
	if g := Load().Glue; g.Background != GlueLight || g.Pad != 0 {
		t.Errorf("неверный фон сбрасывается, отступ 0 допустим: %+v", g)
	}
}

func TestFileFormat(t *testing.T) {
	sandbox(t)
	got := Load()
	if got.Format != FormatPNG || got.JPEGQuality != 90 || got.Ext() != ".png" {
		t.Errorf("по умолчанию PNG, качество 90: %+v", got)
	}
	writeConfig(t, `{"format":"jpeg","jpeg_quality":250}`)
	got = Load()
	if got.Format != FormatJPEG || got.JPEGQuality != 100 || got.Ext() != ".jpg" {
		t.Errorf("JPEG, качество подрезается: %+v", got)
	}
	if name := got.FileName("20260101-120000"); name != "scrawl-20260101-120000.jpg" {
		t.Errorf("имя JPEG-файла: %q", name)
	}
	if name := got.NumberedFileName("20260101-120000", 2); name != "scrawl-20260101-120000-2.jpg" {
		t.Errorf("имя JPEG-файла набора: %q", name)
	}
	writeConfig(t, `{"format":"gif","jpeg_quality":0}`)
	if got := Load(); got.Format != FormatPNG || got.JPEGQuality != 1 {
		t.Errorf("неизвестный формат сбрасывается, качество не ниже 1: %+v", got)
	}
}

func TestKeepOpenDefaultOff(t *testing.T) {
	sandbox(t)
	if Load().KeepOpen {
		t.Error("по умолчанию оверлей закрывается после копирования")
	}
	writeConfig(t, `{"keep_open":true}`)
	if !Load().KeepOpen {
		t.Error("keep_open=true должен читаться")
	}
}

func TestIncludeCursorDefaultOff(t *testing.T) {
	sandbox(t)
	if Load().IncludeCursor {
		t.Error("по умолчанию курсор в снимок не попадает")
	}
	writeConfig(t, `{"include_cursor":true}`)
	if !Load().IncludeCursor {
		t.Error("include_cursor=true должен читаться")
	}
}

func writeLegacy(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(LegacyPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LegacyPath(), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyConfigIsReadAndKept(t *testing.T) {
	sandbox(t)
	if filepath.Base(filepath.Dir(Path())) != "scrawl" || filepath.Base(filepath.Dir(LegacyPath())) != "screen-editor" {
		t.Fatalf("папки настроек: %s и %s", Path(), LegacyPath())
	}
	const old = `{"double_click":"copy","dim_percent":40,"future":"x"}`
	writeLegacy(t, old)

	// Нового файла нет: настройки берутся из старого, но ничего не пишется.
	if got := Load(); got.DoubleClick != DoubleClickCopy || got.DimPercent != 40 {
		t.Errorf("настройки прежней программы должны подхватиться: %+v", got)
	}
	if _, err := os.Stat(Path()); !os.IsNotExist(err) {
		t.Error("чтение не должно создавать новый файл")
	}

	// Первое изменение создаёт новый файл с прежними значениями и чужими ключами.
	st := NewStore()
	if err := st.Update(func(s *Settings) { s.DimPercent = 55 }); err != nil {
		t.Fatal(err)
	}
	keys := readKeys(t)
	if string(keys["dim_percent"]) != "55" || string(keys["double_click"]) != `"copy"` || string(keys["future"]) != `"x"` {
		t.Errorf("новый файл: %v", keys)
	}
	if data, _ := os.ReadFile(LegacyPath()); string(data) != old {
		t.Errorf("старый файл не должен меняться: %q", data)
	}
	if _, err := os.Stat(Path() + ".bak"); !os.IsNotExist(err) {
		t.Error("старый файл не копируется в .bak")
	}

	// Дальше действует только новый файл.
	writeLegacy(t, `{"dim_percent":10}`)
	if got := Load().DimPercent; got != 55 {
		t.Errorf("новый файл главнее старого: %d", got)
	}

	// Повреждённый старый файл даёт значения по умолчанию.
	sandbox(t)
	writeLegacy(t, `{oops`)
	if got := Load(); got != Default() {
		t.Errorf("битый старый файл: %+v", got)
	}
}

func TestParseChord(t *testing.T) {
	good := map[string]string{
		"r": "R", "R": "R", "ctrl+z": "Ctrl+Z", "Control+Shift+s": "Ctrl+Shift+S", "alt+ctrl+X": "Ctrl+Alt+X",
		"enter": "Enter", "ESCAPE": "Escape", "f5": "F5", "F12": "F12", "pageup": "PageUp", "5": "5", " Ctrl + Z ": "Ctrl+Z",
	}
	for in, want := range good {
		if got, err := ParseChord(in); err != nil || got != want {
			t.Errorf("ParseChord(%q) = %q, %v; ожидали %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "Ctrl+", "Super+S", "F13", "F0", "foo", "-", "Ctrl+Shift", "Ctrl++"} {
		if got, err := ParseChord(bad); err == nil {
			t.Errorf("ParseChord(%q) = %q, ожидали ошибку", bad, got)
		}
	}
}

func TestKeys(t *testing.T) {
	sandbox(t)
	k := Load().Keys
	for _, a := range KeyActions {
		if k.Of(a) != DefaultKeys[a] || DefaultKeys[a] == "" {
			t.Errorf("действие %s: по умолчанию %q, получили %q", a, DefaultKeys[a], k.Of(a))
		}
	}
	if _, _, clash := k.Conflict(); clash {
		t.Error("клавиши по умолчанию не должны конфликтовать")
	}

	// Свои клавиши читаются в каноническом виде; значения по умолчанию хранятся пустыми.
	writeConfig(t, `{"keys":{"restore":"f5","add":"ctrl+shift+a","undo":"Ctrl+Z","glue":"nonsense","send":5}}`)
	k = Load().Keys
	if k.Of(ActionRestore) != "F5" || k.Of(ActionAdd) != "Ctrl+Shift+A" {
		t.Errorf("свои клавиши: %+v", k)
	}
	if k.Of(ActionGlue) != "G" || k.Of(ActionSend) != "D" || k.Undo != "" {
		t.Errorf("неверные и умолчательные клавиши: %+v", k)
	}

	// Две одинаковые клавиши: всё возвращается к умолчаниям.
	writeConfig(t, `{"keys":{"restore":"X","add":"x","glue":"F2"}}`)
	k = Load().Keys
	if k != (Keys{}) {
		t.Errorf("при конфликте ожидали умолчания: %+v", k)
	}
	// Клавиша действия совпала с умолчанием другого: тоже конфликт.
	writeConfig(t, `{"keys":{"restore":"A"}}`)
	if k := Load().Keys; k != (Keys{}) {
		t.Errorf("R -> A конфликтует с добавлением в подборку: %+v", k)
	}

	var set Keys
	set.Set(ActionCopy, "Ctrl+C")
	set.Set(ActionSave, "Ctrl+Shift+S")
	if set.Copy != "" || set.Save != "Ctrl+Shift+S" {
		t.Errorf("Set: %+v", set)
	}
	if a, b, clash := (Keys{Add: "R"}).Conflict(); !clash || a != ActionRestore || b != ActionAdd {
		t.Errorf("Conflict: %s %s %v", a, b, clash)
	}
}

// У каждого инструмента есть клавиша, и ключ в файле называется «tool_» + имя инструмента.
func TestToolKeys(t *testing.T) {
	sandbox(t)
	for _, name := range ToolNames {
		a := ToolAction(name)
		if DefaultKeys[a] == "" {
			t.Errorf("у инструмента %s нет клавиши по умолчанию", name)
		}
	}
	writeConfig(t, `{"keys":{"tool_pen":"q","tool_text":"ctrl+t"}}`)
	k := Load().Keys
	if k.Of(ActionToolPen) != "Q" || k.Of(ActionToolText) != "Ctrl+T" || k.Of(ActionToolLine) != "L" {
		t.Errorf("свои клавиши инструментов: %+v", k)
	}
	// Клавиша инструмента совпала с умолчанием действия: конфликт, всё по умолчанию.
	writeConfig(t, `{"keys":{"tool_pen":"A"}}`)
	if k := Load().Keys; k != (Keys{}) {
		t.Errorf("P -> A конфликтует с добавлением в подборку: %+v", k)
	}
}

func TestStampTool(t *testing.T) {
	sandbox(t)
	if DefaultKeys[ActionToolStamp] != "N" {
		t.Errorf("штамп: клавиша по умолчанию %q", DefaultKeys[ActionToolStamp])
	}
	tools := Load().Tools
	if ToolNames[len(ToolNames)-1] != "stamp" || tools.Of("stamp") == nil {
		t.Error("штамп должен быть последним инструментом и иметь вид по умолчанию")
	}
	writeConfig(t, `{"default_tool":"stamp","tools":{"stamp":{"color":"#1e88e5","width":8}}}`)
	got := Load()
	if got.DefaultTool != "stamp" || got.Tools.Stamp.Color != "#1E88E5" || got.Tools.Stamp.Width != 8 {
		t.Errorf("настройки штампа: %q %+v", got.DefaultTool, got.Tools.Stamp)
	}
}

func TestLanguageSetting(t *testing.T) {
	sandbox(t)
	if got := Load().Language; got != "auto" {
		t.Errorf("по умолчанию язык как в системе: %q", got)
	}
	for body, want := range map[string]string{
		`{"language":"en"}`: "en", `{"language":"ru"}`: "ru", `{"language":"de"}`: "de", `{"language":"zh-CN"}`: "zh",
		`{"language":"pt_BR"}`: "pt", `{"language":"KO"}`: "ko", `{"language":"sv"}`: "auto", `{"language":5}`: "auto",
	} {
		writeConfig(t, body)
		if got := Load().Language; got != want {
			t.Errorf("%s: %q, ожидали %q", body, got, want)
		}
	}
}

func TestDefaultToolSelectStaysPossible(t *testing.T) {
	sandbox(t)
	writeConfig(t, `{"default_tool":"select"}`)
	if got := Load().DefaultTool; got != "select" {
		t.Errorf("явный select (ничего не включать) должен сохраняться: %q", got)
	}
}
