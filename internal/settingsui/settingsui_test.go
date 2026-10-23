package settingsui

import (
	"errors"
	"github.com/jexme/scrawl/internal/testenv"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/jexme/scrawl/internal/hotkey"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/settings"
	"github.com/jexme/scrawl/internal/update"
)

func testState(t *testing.T) *state {
	t.Helper()
	testenv.Config(t, t.TempDir())
	testenv.Cache(t, t.TempDir()) // the remembered tab must not be taken from the user's cache
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
	if s.choice.Value != string(settings.DoubleClickCopy) {
		t.Fatalf("по умолчанию двойной клик копирует, а в окне %q", s.choice.Value)
	}
	s.choice.Value = string(settings.DoubleClickCopySave)
	s.layout(frameContext(700, 650)) // apply notices a changed choice within a frame
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
	testenv.Config(t, t.TempDir())
	if err := settings.Save(settings.Settings{DoubleClick: settings.DoubleClickCopy, SaveDir: tmpX}); err != nil {
		t.Fatal(err)
	}
	s := newState(settings.NewStore())
	if s.choice.Value != string(settings.DoubleClickCopy) || s.dir.Text() != tmpX {
		t.Errorf("окно должно показывать сохранённое: выбор=%q, папка=%q", s.choice.Value, s.dir.Text())
	}
}

// TestRenderWindow draws the window for visual inspection (SCRAWL_TEST_OUT).
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
	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		f, _ := os.Create(path + ".window.png")
		defer f.Close()
		png.Encode(f, out)
	}
}

// TestContentScrollsInSmallWindow: in a short window the content is not clipped but
// scrolls; in a tall one it fits entirely.
func TestContentScrollsInSmallWindow(t *testing.T) {
	small := testState(t)
	small.layout(frameContext(820, 700))
	if !small.list.Position.BeforeEnd {
		t.Error("в низком окне часть настроек должна быть ниже видимой области")
	}
	small.list.ScrollTo(len(small.rows()) - 1)
	small.layout(frameContext(820, 700))
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
	s.layout(frameContext(700, 650)) // a text change in the field is noticed within a frame
	if got := settings.Load().FilePrefix; got != "typed" {
		t.Errorf("текст из поля не дошёл до файла: %q", got)
	}
}

func TestOpenOutputDirCreatesAndOpensFolder(t *testing.T) {
	s := testState(t)
	home := t.TempDir()
	testenv.Home(t, home)
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

	// The directory from the settings; a launch error is shown in the window.
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
	s.dim.f.Value = 0.5 // as if the user had dragged the slider
	s.layout(frameContext(700, 650))
	if got := settings.Load().DimPercent; got != 50 {
		t.Errorf("в файле %d, ожидали 50", got)
	}
}

// Merely opening the window writes nothing to the file: slider values after rounding
// match the saved ones (for example, 4.6 does not become 4.6000004).
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
	if got := s.handle.format(0); got != "none" {
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
	blur.sl.Set(5)
	s.layout(frameContext(700, 650))

	got := settings.Load()
	if got.DefaultTool != "marker" {
		t.Errorf("инструмент по умолчанию: %q", got.DefaultTool)
	}
	if got.Tools.Arrow != (settings.ToolStyle{Color: "#1E88E5", Width: 8}) || got.Tools.Blur.Width != 5 {
		t.Errorf("стили инструментов не дошли до файла: %+v", got.Tools)
	}

	// An empty colour clears the setting; an invalid one is rejected and keeps the previous value.
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

	// Invalid colour: an error, the file does not change.
	s.palEd[1].SetText("oops")
	s.layout(frameContext(700, 650))
	if s.palErr == "" || settings.Load().Palette[1] != settings.DefaultPalette[1] || settings.Load().Palette[0] != "#FF0000" {
		t.Errorf("неверный цвет должен отклоняться целиком: ошибка=%q, палитра %v", s.palErr, settings.Load().Palette)
	}

	// The "Default" button.
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

func TestKeepOpenCheckboxIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.keepOpen.Value {
		t.Fatal("по умолчанию оверлей закрывается")
	}
	s.keepOpen.Value = true
	s.layout(frameContext(700, 650))
	if !settings.Load().KeepOpen {
		t.Error("включённая настройка не дошла до файла")
	}
}

func TestCursorCheckboxIsSavedToDisk(t *testing.T) {
	s := testState(t)
	if s.cursor.Value {
		t.Fatal("по умолчанию курсор выключен")
	}
	s.cursor.Value = true
	s.layout(frameContext(700, 650))
	if !settings.Load().IncludeCursor {
		t.Error("включённая настройка не дошла до файла")
	}
}

// waitDone waits until the background action finishes.
func waitDone(t *testing.T, tk *task) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, _, running := tk.status(); !running {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("действие не закончилось за 3 с")
}

func TestGrantButton(t *testing.T) {
	s := testState(t)
	old := grantPermission
	defer func() { grantPermission = old }()

	calls := 0
	grantPermission = func() error { calls++; return nil }
	s.grant.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.grant)
	if msg, failed, _ := s.grant.status(); calls != 1 || failed || !strings.Contains(msg, "io.github.jexme.Scrawl") {
		t.Errorf("вызовов %d, сообщение %q, ошибка %v", calls, msg, failed)
	}

	grantPermission = func() error { return errors.New("portal молчит") }
	s.grant.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.grant)
	if msg, failed, _ := s.grant.status(); !failed || !strings.Contains(msg, "portal молчит") {
		t.Errorf("ошибка должна показываться: %q, %v", msg, failed)
	}
}

func TestMonitorsButtonReportsProgress(t *testing.T) {
	s := testState(t)
	old := setupMonitors
	defer func() { setupMonitors = old }()

	release := make(chan struct{})
	setupMonitors = func(progress func(string)) (string, error) {
		progress("Диалог 1 из 2: выберите монитор")
		<-release
		return "Мониторов настроено: 2", nil
	}
	s.monitors.btn.Click()
	s.layout(frameContext(700, 650))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if msg, _, _ := s.monitors.status(); strings.HasPrefix(msg, "Диалог 1") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if msg, _, running := s.monitors.status(); !running || !strings.HasPrefix(msg, "Диалог 1") {
		t.Fatalf("пока диалоги идут, видна подсказка: %q, идёт=%v", msg, running)
	}
	// Pressing again while it is running starts nothing.
	s.monitors.btn.Click()
	s.layout(frameContext(700, 650))

	close(release)
	waitDone(t, &s.monitors)
	if msg, failed, _ := s.monitors.status(); failed || msg != "Мониторов настроено: 2" {
		t.Errorf("итог: %q, ошибка %v", msg, failed)
	}
}

func TestHotkeyButton(t *testing.T) {
	s := testState(t)
	if got := s.hotkeyEd.Text(); got != "Print" {
		t.Fatalf("по умолчанию Print, в поле %q", got)
	}
	old := assignHotkey
	defer func() { assignHotkey = old }()

	var gotAccel, gotBin string
	assignHotkey = func(accel, bin string) (hotkey.Result, error) {
		gotAccel, gotBin = accel, bin
		return hotkey.Result{Warning: "matches a built-in one"}, nil
	}
	s.hotkeyEd.SetText("  <Super>s ")
	s.hotkey.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.hotkey)
	msg, failed, _ := s.hotkey.status()
	if gotAccel != "<Super>s" || gotBin != "/test/bin/scrawl" || failed || !strings.Contains(msg, "Warning: matches") {
		t.Errorf("комбинация %q, бинарник %q, сообщение %q (ошибка %v)", gotAccel, gotBin, msg, failed)
	}

	// The plain "Ctrl+Alt+P" form is accepted and written the GNOME way.
	gotAccel = ""
	s.hotkeyEd.SetText("Ctrl+Alt+P")
	s.hotkey.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.hotkey)
	if gotAccel != "<Control><Alt>p" || s.hotkeyEd.Text() != "<Control><Alt>p" {
		t.Errorf("введено Ctrl+Alt+P: вызов %q, поле %q", gotAccel, s.hotkeyEd.Text())
	}

	// An invalid entry is rejected before gsettings is called.
	gotAccel = ""
	s.hotkeyEd.SetText("Hyper+S")
	s.hotkey.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.hotkey)
	if msg, failed, _ := s.hotkey.status(); !failed || gotAccel != "" {
		t.Errorf("неверная комбинация: %q (ошибка %v, вызов %q)", msg, failed, gotAccel)
	}

	// A conflict with someone else's shortcut is shown in the window.
	assignHotkey = func(string, string) (hotkey.Result, error) {
		return hotkey.Result{}, &hotkey.Conflict{Name: "Handy", Accel: "Print"}
	}
	s.hotkeyEd.SetText("Print")
	s.hotkey.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.hotkey)
	if msg, failed, _ := s.hotkey.status(); !failed || !strings.Contains(msg, "Handy") {
		t.Errorf("конфликт: %q (ошибка %v)", msg, failed)
	}
}

func keyRowFor(t *testing.T, s *state, action string) *keyRow {
	t.Helper()
	for _, r := range s.keyRows {
		if r.action == action {
			return r
		}
	}
	t.Fatalf("нет строки действия %s", action)
	return nil
}

func TestKeyAssignment(t *testing.T) {
	s := testState(t)
	if len(s.keyRows) != len(settings.KeyActions) {
		t.Fatalf("строк клавиш %d, действий %d", len(s.keyRows), len(settings.KeyActions))
	}
	restore := keyRowFor(t, s, settings.ActionRestore)

	s.assignKey(restore, "f5")
	if got := settings.Load().Keys.Of(settings.ActionRestore); got != "F5" || restore.err != "" {
		t.Errorf("F5 должна сохраниться: %q, ошибка %q", got, restore.err)
	}

	// A key taken by another action (A is "Add another") is rejected.
	s.assignKey(restore, "A")
	if restore.err == "" || !strings.Contains(restore.err, "Add to the collection") {
		t.Errorf("ожидали сообщение о занятой клавише: %q", restore.err)
	}
	if got := settings.Load().Keys.Of(settings.ActionRestore); got != "F5" {
		t.Errorf("при конфликте прежняя клавиша остаётся: %q", got)
	}

	s.assignKey(restore, "Super+Q")
	if restore.err == "" {
		t.Error("неизвестный модификатор должен давать ошибку")
	}

	// A key that was freed (R) is available to another action.
	add := keyRowFor(t, s, settings.ActionAdd)
	s.assignKey(add, "R")
	if got := settings.Load().Keys; got.Of(settings.ActionAdd) != "R" || got.Of(settings.ActionRestore) != "F5" {
		t.Errorf("клавиши: %+v", got)
	}

	// Reset restores the default.
	s.assignKey(restore, "") // the default R is taken by the "Add another" action: the reset fails
	if restore.err == "" || settings.Load().Keys.Of(settings.ActionRestore) != "F5" {
		t.Errorf("сброс в занятую клавишу должен отклоняться: %q", restore.err)
	}
	s.assignKey(add, "")
	s.assignKey(restore, "")
	if got := settings.Load().Keys; got != (settings.Keys{}) {
		t.Errorf("после сброса ожидали умолчания: %+v", got)
	}

	// Pressing the reset button in the window.
	s.assignKey(restore, "F6")
	restore.reset.Click()
	s.layout(frameContext(700, 650))
	if got := settings.Load().Keys.Of(settings.ActionRestore); got != "R" {
		t.Errorf("кнопка «Сброс»: %q", got)
	}
}

func TestLanguageSwitchesImmediately(t *testing.T) {
	defer i18n.SetLanguage(i18n.EN)
	s := testState(t)
	if s.lang.Value != i18n.Auto {
		t.Fatalf("по умолчанию язык как в системе: %q", s.lang.Value)
	}
	s.lang.Value = i18n.EN
	s.layout(frameContext(700, 650))
	if got := settings.Load().Language; got != "en" {
		t.Errorf("в файле %q", got)
	}
	if i18n.Language() != i18n.EN {
		t.Fatalf("язык интерфейса не переключился: %q", i18n.Language())
	}
	// The window is rebuilt on every frame, so the texts are already English.
	if got := i18n.T("Close"); got != "Close" {
		t.Errorf("перевод: %q", got)
	}
	s.lang.Value = i18n.RU
	s.layout(frameContext(700, 650))
	if i18n.Language() != i18n.RU || i18n.T("Close") != "Закрыть" {
		t.Errorf("возврат на русский: %q", i18n.Language())
	}
}

// No screen of the settings window may fail in the original English or in Russian, and
// the key texts are translated.
func TestRenderEnglishWindow(t *testing.T) {
	defer i18n.SetLanguage(i18n.EN)
	i18n.SetLanguage(i18n.EN)
	s := testState(t)
	s.layout(frameContext(700, 3300))
	i18n.SetLanguage(i18n.RU)
	s.layout(frameContext(700, 3300))
	for _, key := range []string{"General", "Interface language", "Screen capture"} {
		if i18n.T(key) == key {
			t.Errorf("нет русского перевода: %q", key)
		}
	}
}

// TestRenderEnglishPicture draws the window in English for visual inspection (SCRAWL_TEST_OUT).
func TestRenderEnglishPicture(t *testing.T) {
	path := os.Getenv("SCRAWL_TEST_OUT")
	if path == "" {
		t.Skip("нужна переменная SCRAWL_TEST_OUT")
	}
	defer i18n.SetLanguage(i18n.EN)
	i18n.SetLanguage(i18n.EN)
	const W, H = 700, 1100
	s := testState(t)
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
	f, _ := os.Create(path + ".window-en.png")
	defer f.Close()
	png.Encode(f, out)
}

func TestAutostartCheckbox(t *testing.T) {
	s := testState(t)
	oldOn, oldEnable, oldOff := autostartOn, autostartEnable, autostartOff
	defer func() { autostartOn, autostartEnable, autostartOff = oldOn, oldEnable, oldOff }()

	enabled := false
	var gotBin string
	autostartOn = func() bool { return enabled }
	autostartEnable = func(bin string) error { gotBin, enabled = bin, true; return nil }
	autostartOff = func() error { enabled = false; return nil }

	s.autostart.Value = true
	s.layout(frameContext(700, 650))
	if !enabled || gotBin != "/test/bin/scrawl" || s.autoErr != "" {
		t.Errorf("включение: файл %v, бинарник %q, ошибка %q", enabled, gotBin, s.autoErr)
	}
	s.autostart.Value = false
	s.layout(frameContext(700, 650))
	if enabled || s.autoErr != "" {
		t.Errorf("выключение: файл %v, ошибка %q", enabled, s.autoErr)
	}

	// Write error: the checkbox returns to the file's state, the reason is visible in the window.
	autostartEnable = func(string) error { return errors.New("диск только для чтения") }
	s.autostart.Value = true
	s.layout(frameContext(700, 650))
	if s.autostart.Value || !strings.Contains(s.autoErr, "диск только для чтения") {
		t.Errorf("при ошибке галочка должна сняться: %v, %q", s.autostart.Value, s.autoErr)
	}
}

func TestUpdateButtonAndSwitch(t *testing.T) {
	s := testState(t)
	old := checkUpdate
	defer func() { checkUpdate = old }()

	if s.checkUpd.Value {
		t.Fatal("проверка обновлений по умолчанию выключена")
	}
	calls := 0
	checkUpdate = func() (update.Result, error) {
		calls++
		return update.Result{Latest: "v9.9.9", URL: "https://example/r", Newer: true, Comparable: true}, nil
	}
	// The button works while the check is turned off.
	s.update.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.update)
	if msg, failed, _ := s.update.status(); calls != 1 || failed || !strings.Contains(msg, "v9.9.9") {
		t.Errorf("вызовов %d, сообщение %q, ошибка %v", calls, msg, failed)
	}

	checkUpdate = func() (update.Result, error) { return update.Result{}, errors.New("нет связи") }
	s.update.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.update)
	if msg, failed, _ := s.update.status(); !failed || !strings.Contains(msg, "нет связи") {
		t.Errorf("ошибка сети должна показываться: %q %v", msg, failed)
	}

	s.checkUpd.Value = true
	s.layout(frameContext(700, 650))
	if !settings.Load().CheckUpdates {
		t.Error("включённая проверка не дошла до файла")
	}
}

// TestDocSettingsImages draws the settings window in two languages for the README
// (the directory is in the SCRAWL_DOC_IMAGES variable).
func TestDocSettingsImages(t *testing.T) {
	dir := os.Getenv("SCRAWL_DOC_IMAGES")
	if dir == "" {
		t.Skip("нужна переменная SCRAWL_DOC_IMAGES")
	}
	defer i18n.SetLanguage(i18n.EN)
	const W, H = 820, 640
	for lang, name := range map[string]string{i18n.RU: "settings-ru.png", i18n.EN: "settings-en.png"} {
		i18n.SetLanguage(lang)
		testenv.Config(t, "/home/you/.config") // a pretty path in the label; the file is not created
		testenv.Cache(t, t.TempDir())
		s := newState(settings.NewStore())
		s.selectTab(1) // the "Tools" tab: the most is visible on it
		win, err := headless.NewWindow(W, H)
		if err != nil {
			t.Skipf("headless рендер недоступен: %v", err)
		}
		gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(W, H)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		paint.Fill(gtx.Ops, bgColor)
		s.layout(gtx)
		if err := win.Frame(gtx.Ops); err != nil {
			t.Fatal(err)
		}
		out := image.NewRGBA(image.Rect(0, 0, W, H))
		if err := win.Screenshot(out); err != nil {
			t.Fatal(err)
		}
		win.Release()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, out)
		f.Close()
	}
}

func TestCaptureRowsDependOnSession(t *testing.T) {
	old, oldWin := x11Session, onWindows
	defer func() { x11Session, onWindows = old, oldWin }()
	s := testState(t)
	onWindows = false
	x11Session = func() bool { return false }
	wayland := len(s.captureRows())
	x11Session = func() bool { return true }
	x11 := len(s.captureRows())
	if wayland != 5 || x11 != 3 {
		t.Errorf("строк захвата: Wayland %d (ожидали 5), X11 %d (ожидали 3: без кнопок портала)", wayland, x11)
	}
	// The first-launch welcome adds one card with the steps, in every kind of session.
	s.welcome = true
	if n := len(s.captureRows()); n != x11+1 {
		t.Errorf("с приветствием строк %d, ожидали %d", n, x11+1)
	}
	s.layout(frameContext(700, 3300))
	s.welcome = false
	s.layout(frameContext(700, 3300)) // the window is drawn in an X11 session without errors

	// Windows: no portal and no GNOME, only a single hotkey line remains.
	onWindows = true
	if n := len(s.captureRows()); n != 1 {
		t.Errorf("строк захвата в Windows %d, ожидали 1", n)
	}
	s.layout(frameContext(700, 3300))
}

func TestTabs(t *testing.T) {
	s := testState(t)
	tabs := s.tabs()
	if len(tabs) != 8 {
		t.Fatalf("вкладок %d, ожидали 8", len(tabs))
	}
	seen := map[string]bool{}
	for _, tb := range tabs {
		if tb.label == "" || len(tb.rows) == 0 || seen[tb.label] {
			t.Errorf("вкладка %q: пустая или повторяется (строк %d)", tb.label, len(tb.rows))
		}
		seen[tb.label] = true
	}

	// Every tab is drawn without errors, at any window width and in both languages.
	defer i18n.SetLanguage(i18n.EN)
	for _, l := range i18n.Languages {
		lang := l.Code
		i18n.SetLanguage(lang)
		for i := range tabs {
			s.selectTab(i)
			for _, w := range []int{620, 820, 1400} {
				s.layout(frameContext(w, 700))
			}
		}
	}

	// Switching tabs scrolls back to the top.
	s.selectTab(0)
	s.list.Position.First = 4
	s.selectTab(1)
	if s.list.Position.First != 0 || s.tab != 1 {
		t.Errorf("после переключения: вкладка %d, прокрутка с %d", s.tab, s.list.Position.First)
	}
	s.selectTab(99) // outside the list: the tab is clipped when drawn
	s.layout(frameContext(820, 700))
	if s.tab != len(tabs)-1 {
		t.Errorf("несуществующая вкладка должна сжаться до последней: %d", s.tab)
	}

	// Clicking a tab button selects it.
	s.tabBtns[2].Click()
	s.layout(frameContext(820, 700))
	if s.tab != 2 {
		t.Errorf("клик по вкладке: выбрана %d", s.tab)
	}
	// The content changes in the same frame in which the click arrived: there may be no next
	// frame until a new event, and the click would seem late.
	s.tabBtns[5].Click()
	s.layout(frameContext(820, 700))
	if s.shownTab != 5 {
		t.Errorf("после клика в этом же кадре нарисована вкладка %d, ожидали 5", s.shownTab)
	}
}

// The row of tabs fits in a window of minimum width in both languages.
func TestTabBarFitsMinimumWidth(t *testing.T) {
	defer i18n.SetLanguage(i18n.EN)
	const minWindowDp, padding = 700, 44 // the window's MinSize and 22 px padding on each side
	for _, l := range i18n.Languages {
		lang := l.Code
		i18n.SetLanguage(lang)
		s := testState(t)
		gtx := layout.Context{
			Ops:         new(op.Ops),
			Constraints: layout.Constraints{Max: image.Pt(4000, 400)},
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		}
		dims := s.tabRow(gtx, s.tabs())
		if dims.Size.X > minWindowDp-padding {
			t.Errorf("%s: ряд вкладок %d dp шире доступных %d dp", lang, dims.Size.X, minWindowDp-padding)
		}
	}
}

func TestEveryLanguageCanBeChosen(t *testing.T) {
	defer i18n.SetLanguage(i18n.EN)
	s := testState(t)
	for _, l := range i18n.Languages {
		s.lang.Value = l.Code
		s.layout(frameContext(820, 700))
		if got := settings.Load().Language; got != l.Code {
			t.Errorf("язык %s: в файле %q", l.Code, got)
		}
		if i18n.Language() != l.Code {
			t.Errorf("язык %s: действует %q", l.Code, i18n.Language())
		}
	}
	// "Same as the system" restores autodetection.
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	s.lang.Value = i18n.Auto
	s.layout(frameContext(820, 700))
	if i18n.Language() != i18n.DE || settings.Load().Language != i18n.Auto {
		t.Errorf("auto при немецкой локали: действует %q, в файле %q", i18n.Language(), settings.Load().Language)
	}
}

func TestLastTabIsRemembered(t *testing.T) {
	s := testState(t)
	if s.tab != 0 {
		t.Fatalf("в чистом кэше окно открывается на первой вкладке: %d", s.tab)
	}
	s.selectTab(6) // "System"
	again := newState(settings.NewStore())
	if again.tab != 6 || again.tabs()[again.tab].id != "system" {
		t.Errorf("после перезапуска ожидали вкладку «system», открыта %d", again.tab)
	}
	// An unknown name in the file does not break the window.
	saveTab("такой вкладки нет")
	if got := newState(settings.NewStore()).tab; got != 0 {
		t.Errorf("неизвестная вкладка: %d", got)
	}
}

func TestFontHintShownOnlyWhenFontIsMissing(t *testing.T) {
	defer i18n.SetLanguage(i18n.EN)
	old := fontAvailable
	defer func() { fontAvailable = old }()
	s := testState(t)
	i18n.SetLanguage(i18n.ZH)

	measure := func() int {
		gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(800, 2000)}, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		return s.languageRadios()(gtx).Size.Y
	}
	fontAvailable = func(string) bool { return true }
	without := measure()
	fontAvailable = func(string) bool { return false }
	with := measure()
	if with <= without {
		t.Errorf("без шрифта под списком должна быть подсказка: высота %d и %d", with, without)
	}
}

// tmpX: an absolute path on any platform (Windows needs a drive letter).
var tmpX = func() string {
	if runtime.GOOS == "windows" {
		return "C:/tmp/x"
	}
	return "/tmp/x"
}()

func TestFreeKeyButton(t *testing.T) {
	old := freeBuiltin
	defer func() { freeBuiltin = old }()
	var gotAccel string
	freeBuiltin = func(accel string) ([]string, error) {
		gotAccel = accel
		return []string{"show-screenshot-ui"}, nil
	}
	s := testState(t)
	s.hotkeyEd.SetText("  Print ")
	s.freeKey.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.freeKey)
	msg, failed, _ := s.freeKey.status()
	if gotAccel != "Print" || failed || !strings.Contains(msg, "show-screenshot-ui") {
		t.Errorf("combination %q, message %q (failed %v)", gotAccel, msg, failed)
	}

	freeBuiltin = func(string) ([]string, error) { return nil, nil }
	s.freeKey.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.freeKey)
	if msg, _, _ := s.freeKey.status(); !strings.Contains(msg, "nothing to change") {
		t.Errorf("nothing to free: %q", msg)
	}
}

func TestSetupCaptureButton(t *testing.T) {
	oldMon, oldGrant, oldSaved := setupMonitors, grantPermission, savedMonitors
	defer func() { setupMonitors, grantPermission, savedMonitors = oldMon, oldGrant, oldSaved }()
	var order []string
	setupMonitors = func(func(string)) (string, error) {
		order = append(order, "monitors")
		return "Monitors set up: 2", nil
	}
	grantPermission = func() error { order = append(order, "grant"); return nil }
	savedMonitors = func() int { return 2 }
	s := testState(t)
	s.setup.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.setup)
	msg, failed, _ := s.setup.status()
	if failed || strings.Join(order, ",") != "monitors,grant" || !strings.Contains(msg, "Permission granted") {
		t.Errorf("one button does both, in this order: %v, message %q (failed %v)", order, msg, failed)
	}

	// A failure of the first step stops the second.
	order = nil
	setupMonitors = func(func(string)) (string, error) { return "", errors.New("no portal") }
	s.setup.btn.Click()
	s.layout(frameContext(700, 650))
	waitDone(t, &s.setup)
	if msg, failed, _ := s.setup.status(); !failed || !strings.Contains(msg, "no portal") || len(order) != 0 {
		t.Errorf("a failed first step must stop the second: %q failed=%v order=%v", msg, failed, order)
	}
}

// Opening the settings window must not change the settings: every slider starts at the saved
// value. The tool widths used to be erased by the first frame.
func TestOpeningWindowKeepsSavedSettings(t *testing.T) {
	testenv.Config(t, t.TempDir())
	testenv.Cache(t, t.TempDir())
	want := settings.Default()
	want.Tools.Arrow.Color, want.Tools.Arrow.Width = "#00FF00", 15
	want.Tools.Pen.Width = 9
	want.Tools.Marker.Width = 7
	want.DimPercent = 55
	want.Glue.Pad = 40
	if err := settings.Save(want); err != nil {
		t.Fatal(err)
	}
	s := newState(settings.NewStore())
	s.layout(frameContext(700, 3300))
	s.layout(frameContext(700, 3300))
	if got := settings.Load(); got != want {
		t.Errorf("opening the window changed the settings:\n got %+v\nwant %+v", got.Tools, want.Tools)
	}
}
