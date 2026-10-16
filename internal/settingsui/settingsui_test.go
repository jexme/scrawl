package settingsui

import (
	"image"
	"image/png"
	"os"
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
	const W, H = 700, 650
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
	tall.layout(frameContext(600, 1200))
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
