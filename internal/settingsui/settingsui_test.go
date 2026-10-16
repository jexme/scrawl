package settingsui

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/jexme/screen-editor/internal/settings"
)

func testState(t *testing.T) *state {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return newState(settings.NewStore())
}

func frameContext(w, h int) layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(w, h)),
		Metric:      unit.Metric{PxPerDp: 1.25, PxPerSp: 1.25},
	}
}

func TestChoiceIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.choice.Value != string(settings.DoubleClickOff) {
		t.Fatalf("по умолчанию двойной клик выключен, а в окне %q", s.choice.Value)
	}
	s.choice.Value = string(settings.DoubleClickCopySave)
	s.layout(frameContext(700, 650)) // apply на кадре замечает смену выбора
	if got := settings.Load().DoubleClick; got != settings.DoubleClickCopySave {
		t.Errorf("выбор не дошёл до файла: %q", got)
	}
}

func TestInvalidDirIsRejected(t *testing.T) {
	s := testState(t)
	s.applyDir("относительный/путь")
	if s.dirErr == "" {
		t.Error("относительный путь должен давать ошибку")
	}
	if got := settings.Load().SaveDir; got != "" {
		t.Errorf("неверный путь не должен сохраняться: %q", got)
	}
	s.applyDir("~/Shots")
	if s.dirErr != "" || settings.Load().SaveDir != "~/Shots" {
		t.Errorf("верный путь должен сохраниться без ошибки: ошибка=%q, файл=%q", s.dirErr, settings.Load().SaveDir)
	}
}

func TestInitialStateReflectsFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := settings.Save(settings.Settings{DoubleClick: settings.DoubleClickCopy, SaveDir: "/tmp/x"}); err != nil {
		t.Fatal(err)
	}
	s := newState(settings.NewStore())
	if s.choice.Value != string(settings.DoubleClickCopy) || s.dir.Text() != "/tmp/x" {
		t.Errorf("окно должно показывать сохранённое: выбор=%q, папка=%q", s.choice.Value, s.dir.Text())
	}
}

// TestRenderWindow рисует окно для просмотра глазами (SCREEN_EDITOR_TEST_OUT).
func TestRenderWindow(t *testing.T) {
	const W, H = 700, 3300
	s := testState(t)
	s.choice.Value = string(settings.DoubleClickCopy)

	win, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer win.Release()

	gtx := frameContext(W, H)
	paint.Fill(gtx.Ops, bgColor)
	s.layout(gtx)
	if err := win.Frame(gtx.Ops); err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := win.Screenshot(out); err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("SCREEN_EDITOR_TEST_OUT"); path != "" {
		f, _ := os.Create(path + ".window.png")
		defer f.Close()
		png.Encode(f, out)
	}
}

// TestContentScrollsInSmallWindow: в низком окне содержимое не обрезается, а
// прокручивается; в высоком влезает целиком.
func TestContentScrollsInSmallWindow(t *testing.T) {
	small := testState(t)
	small.layout(frameContext(600, 380))
	if !small.list.Position.BeforeEnd {
		t.Error("в низком окне часть настроек должна быть ниже видимой области")
	}
	small.list.ScrollTo(len(small.items()) - 1)
	small.layout(frameContext(600, 380))
	if small.list.Position.BeforeEnd {
		t.Errorf("после прокрутки к последнему разделу конец списка должен быть виден: %+v", small.list.Position)
	}

	tall := testState(t)
	tall.layout(frameContext(600, 12000))
	if tall.list.Position.BeforeEnd || tall.list.Position.First != 0 {
		t.Errorf("в высоком окне всё должно помещаться: %+v", tall.list.Position)
	}
}

func TestEnterChoiceIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.enter.Value != string(settings.EnterCopySave) {
		t.Fatalf("по умолчанию Enter копирует и сохраняет, а в окне %q", s.enter.Value)
	}
	s.enter.Value = string(settings.EnterCopy)
	s.layout(frameContext(700, 650))
	if got := settings.Load().Enter; got != settings.EnterCopy {
		t.Errorf("выбор Enter не дошёл до файла: %q", got)
	}
}

func TestNotificationsCheckboxIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if !s.notify.Value {
		t.Fatal("по умолчанию уведомления включены")
	}
	s.notify.Value = false
	s.layout(frameContext(700, 650))
	if settings.Load().Notifications {
		t.Error("выключенные уведомления не дошли до файла")
	}
}

func TestRememberRegionCheckboxIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if !s.remember.Value {
		t.Fatal("по умолчанию область запоминается")
	}
	s.remember.Value = false
	s.layout(frameContext(700, 650))
	if settings.Load().RememberRegion {
		t.Error("выключенная настройка не дошла до файла")
	}
}

func TestFilePrefixIsValidatedAndSaved(t *testing.T) {
	s := testState(t)
	s.applyPrefix("a/b")
	if s.prefixErr == "" || settings.Load().FilePrefix != "" {
		t.Errorf("префикс со слэшем должен отклоняться: ошибка=%q, файл=%q", s.prefixErr, settings.Load().FilePrefix)
	}
	s.applyPrefix("shot")
	if s.prefixErr != "" || settings.Load().FilePrefix != "shot" {
		t.Errorf("верный префикс должен сохраниться: ошибка=%q, файл=%q", s.prefixErr, settings.Load().FilePrefix)
	}
	s.prefix.SetText("typed")
	s.layout(frameContext(700, 650)) // изменение текста в поле замечается на кадре
	if got := settings.Load().FilePrefix; got != "typed" {
		t.Errorf("текст из поля не дошёл до файла: %q", got)
	}
}

func TestOpenOutputDirCreatesAndOpensFolder(t *testing.T) {
	s := testState(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	var opened string
	old := openFolder
	openFolder = func(dir string) error { opened = dir; return nil }
	defer func() { openFolder = old }()

	s.openOutputDir()
	want := filepath.Join(home, "Pictures", "Screenshots")
	if opened != want || s.dirErr != "" {
		t.Errorf("открыто %q (ошибка %q), ожидали %q", opened, s.dirErr, want)
	}
	if st, err := os.Stat(want); err != nil || !st.IsDir() {
		t.Errorf("папка должна быть создана: %v", err)
	}

	// Папка из настроек, ошибка запуска показывается в окне.
	custom := filepath.Join(home, "мои снимки")
	s.store.Update(func(c *settings.Settings) { c.SaveDir = custom })
	openFolder = func(dir string) error { opened = dir; return errors.New("нет xdg-open") }
	s.openOutputDir()
	if opened != custom || s.dirErr == "" {
		t.Errorf("открыто %q, ошибка %q: ожидали свою папку и сообщение об ошибке", opened, s.dirErr)
	}
}

func TestDimSliderSnapsAndSaves(t *testing.T) {
	s := testState(t)
	if got := s.dim.Value(); got != 86 {
		t.Fatalf("ползунок должен показывать 86%%, а не %v", got)
	}
	s.dim.Set(33.4)
	if got := s.dim.Value(); got != 33 {
		t.Errorf("значение должно прижиматься к шагу: %v", got)
	}
	s.dim.f.Value = 0.5 // как если бы пользователь перетащил ползунок
	s.layout(frameContext(700, 650))
	if got := settings.Load().DimPercent; got != 50 {
		t.Errorf("в файле %d, ожидали 50", got)
	}
}

// Простое открытие окна ничего не пишет в файл: значения ползунков после
// округления совпадают с сохранёнными (например, 4.6 не становится 4.6000004).
func TestOpeningWindowDoesNotRewriteConfig(t *testing.T) {
	s := testState(t)
	s.layout(frameContext(700, 650))
	s.layout(frameContext(700, 650))
	if _, err := os.Stat(settings.Path()); !os.IsNotExist(err) {
		data, _ := os.ReadFile(settings.Path())
		t.Errorf("окно без изменений не должно писать файл настроек, а там: %s", data)
	}
}

func TestBorderControlsAreSavedToDisk(t *testing.T) {
	s := testState(t)
	s.bWidth.Set(2.5)
	s.bDash.Set(6)
	s.bGap.Set(0)
	s.layout(frameContext(700, 650))
	got := settings.Load()
	if got.BorderWidth != 2.5 || got.BorderDash != 6 || got.BorderGap != 0 {
		t.Errorf("ползунки рамки не дошли до файла: %+v", got)
	}

	s.applyBorderColor("#12")
	if s.colorErr == "" || settings.Load().BorderColor != "#FFFFFF" {
		t.Errorf("неверный цвет должен отклоняться: ошибка=%q, файл=%q", s.colorErr, settings.Load().BorderColor)
	}
	s.bColor.SetText("00ff80")
	s.layout(frameContext(700, 650))
	if s.colorErr != "" || settings.Load().BorderColor != "#00FF80" {
		t.Errorf("верный цвет должен сохраниться как #RRGGBB: ошибка=%q, файл=%q", s.colorErr, settings.Load().BorderColor)
	}
}

func TestHandleSliderIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.handle.Value() != 3 {
		t.Fatalf("ручки по умолчанию 3 px, а в окне %v", s.handle.Value())
	}
	s.handle.Set(0)
	s.layout(frameContext(700, 650))
	if got := settings.Load().HandleRadius; got != 0 {
		t.Errorf("0 (ручки спрятаны) должен сохраниться как 0, а не вернуться к умолчанию: %v", got)
	}
	if got := s.handle.format(0); got != "нет" {
		t.Errorf("подпись нуля: %q", got)
	}
}

func TestShowSizeCheckboxIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if !s.showSize.Value {
		t.Fatal("по умолчанию подпись размера включена")
	}
	s.showSize.Value = false
	s.layout(frameContext(700, 650))
	if settings.Load().ShowSize {
		t.Error("выключенная подпись не дошла до файла")
	}
}

func TestToolDefaultsAreSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.toolChoice.Value != "select" || len(s.toolRows) != len(settings.ToolNames)-1 {
		t.Fatalf("выбор инструмента %q, строк инструментов %d", s.toolChoice.Value, len(s.toolRows))
	}
	s.toolChoice.Value = "marker"
	var arrow, blur *toolRow
	for _, r := range s.toolRows {
		switch r.name {
		case "arrow":
			arrow = r
		case "blur":
			blur = r
		}
	}
	arrow.ed.SetText("1e88e5")
	arrow.sl.Set(8)
	blur.sl.Set(12)
	s.layout(frameContext(700, 650))

	got := settings.Load()
	if got.DefaultTool != "marker" {
		t.Errorf("инструмент по умолчанию: %q", got.DefaultTool)
	}
	if got.Tools.Arrow != (settings.ToolStyle{Color: "#1E88E5", Width: 8}) || got.Tools.Blur.Width != 12 {
		t.Errorf("стили инструментов не дошли до файла: %+v", got.Tools)
	}

	// Пустой цвет очищает настройку, неверный отклоняется и оставляет прошлое значение.
	arrow.ed.SetText("zzz")
	s.layout(frameContext(700, 650))
	if arrow.err == "" || settings.Load().Tools.Arrow.Color != "#1E88E5" {
		t.Errorf("неверный цвет: ошибка=%q, в файле %q", arrow.err, settings.Load().Tools.Arrow.Color)
	}
	arrow.ed.SetText("")
	s.layout(frameContext(700, 650))
	if arrow.err != "" || settings.Load().Tools.Arrow.Color != "" {
		t.Errorf("пустой цвет должен очищать: ошибка=%q, в файле %q", arrow.err, settings.Load().Tools.Arrow.Color)
	}
}

func TestPaletteIsSavedToDisk(t *testing.T) {
	s := testState(t)
	s.palEd[0].SetText("ff0000")
	s.palEd[3].SetText("")
	s.layout(frameContext(700, 650))
	got := settings.Load().Palette
	if got[0] != "#FF0000" || got[3] != settings.DefaultPalette[3] || s.palErr != "" {
		t.Errorf("палитра: %v, ошибка %q", got, s.palErr)
	}

	// Неверный цвет: ошибка, файл не меняется.
	s.palEd[1].SetText("oops")
	s.layout(frameContext(700, 650))
	if s.palErr == "" || settings.Load().Palette[1] != settings.DefaultPalette[1] || settings.Load().Palette[0] != "#FF0000" {
		t.Errorf("неверный цвет должен отклоняться целиком: ошибка=%q, палитра %v", s.palErr, settings.Load().Palette)
	}

	// Кнопка «По умолчанию».
	s.palEd[1].SetText("")
	s.palEd[0].SetText("00ff00")
	s.layout(frameContext(700, 650))
	if settings.Load().Palette[0] != "#00FF00" {
		t.Fatalf("новый цвет не сохранился: %v", settings.Load().Palette)
	}
	s.palReset.Click()
	s.layout(frameContext(700, 650))
	if settings.Load().Palette != settings.DefaultPalette || s.palEd[0].Text() != settings.DefaultPalette[0] {
		t.Errorf("кнопка должна вернуть палитру по умолчанию: %v, поле %q", settings.Load().Palette, s.palEd[0].Text())
	}
}

func TestMarkerOpacitySliderIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.marker.Value() != 43 {
		t.Fatalf("по умолчанию 43%%, в окне %v", s.marker.Value())
	}
	s.marker.Set(70)
	s.layout(frameContext(700, 650))
	if got := settings.Load().MarkerOpacity; got != 70 {
		t.Errorf("в файле %d, ожидали 70", got)
	}
}

func TestGlueLabelsAreSavedToDisk(t *testing.T) {
	s := testState(t)
	s.glue[0].SetText("До")
	s.glue[2].SetText("Этап")
	s.layout(frameContext(700, 650))
	g := settings.Load().Glue
	if g.Before != "До" || g.After != "" || g.Step != "Этап" {
		t.Errorf("подписи не дошли до файла: %+v", g)
	}
	s.glue[1].SetText(strings.Repeat("я", 31))
	s.layout(frameContext(700, 650))
	if s.glueErr[1] == "" || settings.Load().Glue.After != "" {
		t.Errorf("слишком длинная подпись должна отклоняться: ошибка=%q", s.glueErr[1])
	}
}

func TestGlueLayoutIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.glueLayout.Value != string(settings.GlueSide) {
		t.Fatalf("по умолчанию рядом, в окне %q", s.glueLayout.Value)
	}
	s.glueLayout.Value = string(settings.GlueStack)
	s.layout(frameContext(700, 650))
	if got := settings.Load().Glue.Layout; got != settings.GlueStack {
		t.Errorf("в файле %q", got)
	}
}

func TestGlueLookIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.gluePad.Value() != 28 || s.glueFont.Value() != 28 || s.glueBg.Value != string(settings.GlueLight) {
		t.Fatalf("по умолчанию: отступ %v, подпись %v, фон %q", s.gluePad.Value(), s.glueFont.Value(), s.glueBg.Value)
	}
	s.glueBg.Value = string(settings.GlueDark)
	s.gluePad.Set(12)
	s.glueFont.Set(40)
	s.layout(frameContext(700, 650))
	g := settings.Load().Glue
	if g.Background != settings.GlueDark || g.Pad != 12 || g.LabelSize != 40 {
		t.Errorf("вид склейки не дошёл до файла: %+v", g)
	}
}

func TestFormatIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.format.Value != string(settings.FormatPNG) || s.quality.Value() != 90 {
		t.Fatalf("по умолчанию PNG и 90, в окне %q и %v", s.format.Value, s.quality.Value())
	}
	s.format.Value = string(settings.FormatJPEG)
	s.quality.Set(65)
	s.layout(frameContext(700, 650))
	got := settings.Load()
	if got.Format != settings.FormatJPEG || got.JPEGQuality != 65 {
		t.Errorf("формат не дошёл до файла: %+v", got)
	}
}
