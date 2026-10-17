package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
)

func TestHSVRoundTrip(t *testing.T) {
	for _, c := range []color.NRGBA{
		{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255},
		{R: 20, G: 90, B: 160, A: 255}, {R: 230, G: 230, B: 230, A: 255}, {A: 255}, {R: 255, G: 255, B: 255, A: 255},
		{R: 255, G: 128, B: 0, A: 255}, {R: 66, G: 133, B: 244, A: 255},
	} {
		if got := hsvOf(c).rgb(); got != c {
			t.Errorf("%v -> %+v -> %v", c, hsvOf(c), got)
		}
	}
	if got := (hsv{H: 120, S: 1, V: 1}).rgb(); got != (color.NRGBA{G: 255, A: 255}) {
		t.Errorf("чистый зелёный: %v", got)
	}
	if got := (hsv{H: 360, S: 1, V: 1}).rgb(); got != (color.NRGBA{R: 255, A: 255}) {
		t.Errorf("оттенок 360 это красный: %v", got)
	}
}

// newPickerMonitor делает монитор с открытым окошком цвета и областью под панелью.
func newPickerMonitor(t *testing.T, w, h int) *monitor {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(100, 20, 500, 120), true
	m.pick.open = true
	m.pick.hsv = hsv{H: 0, S: 1, V: 1}
	m.pickAnchor = 400
	return m
}

// TestPickerPointer проверяет, что нажатие в квадрате и на полосе меняет цвет в нужную сторону
// (позиция события считается от угла окошка), а отпускание запоминает цвет в настройках.
func TestPickerPointer(t *testing.T) {
	const W, H = 900, 500
	m := newPickerMonitor(t, W, H)
	bar := image.Rect(100, 130, 800, 180)
	win := image.Pt(W, H)

	var r input.Router
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(win), Source: r.Source()}
		m.layoutPicker(gtx, win, bar, m.pickAnchor)
		r.Frame(&ops)
	}
	frame()

	gtx := layout.Context{Metric: identity}
	_, _, pad, svH, hueH, _ := pickerSize(gtx)
	origin := pickerOrigin(gtx, win, bar, m.pickAnchor)
	w, _, _, _, _, _ := pickerSize(gtx)
	inner := w - 2*pad
	at := func(x, y int) f32.Point { return f32.Pt(float32(origin.X+x), float32(origin.Y+y)) }
	press := func(kind pointer.Kind, p f32.Point) {
		r.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary})
		frame()
	}

	// Правый нижний угол квадрата: насыщенность 1, яркость 0, чёрный.
	press(pointer.Press, at(pad+inner-1, pad+svH-1))
	if m.pick.hsv.S < 0.99 || m.pick.hsv.V > 0.02 {
		t.Errorf("правый нижний угол квадрата: %+v", m.pick.hsv)
	}
	// Левый верхний: белый.
	press(pointer.Move, at(pad+1, pad+1))
	if m.pick.hsv.S > 0.02 || m.pick.hsv.V < 0.98 {
		t.Errorf("левый верхний угол квадрата: %+v", m.pick.hsv)
	}
	// Середина квадрата.
	press(pointer.Move, at(pad+inner/2, pad+svH/2))
	if s, v := m.pick.hsv.S, m.pick.hsv.V; s < 0.45 || s > 0.55 || v < 0.45 || v > 0.55 {
		t.Errorf("середина квадрата: %+v", m.pick.hsv)
	}
	press(pointer.Release, at(pad+inner/2, pad+svH/2))
	mid := m.pick.hsv.rgb()
	if m.color != mid || !m.colorPicked || m.customColor != mid || !m.hasCustom {
		t.Errorf("выбранный цвет не стал цветом рисования: %v %v %v", m.color, m.customColor, mid)
	}
	if got := settings.Load().CustomColor; got != settings.FormatColor(mid) {
		t.Errorf("цвет не запомнился: %q, ожидали %q", got, settings.FormatColor(mid))
	}

	// Полоса оттенка: треть ширины это зелёный (120°).
	hueY := pad + svH + pad/2 + hueH/2
	press(pointer.Press, at(pad+inner/3, hueY))
	if h := m.pick.hsv.H; h < 118 || h > 122 {
		t.Errorf("оттенок на трети полосы: %v, ожидали около 120", h)
	}
}

// TestRenderPicker рисует окошко и проверяет цвета в углах квадрата и на полосе. С
// SCRAWL_TEST_OUT=путь.png результат сохраняется.
func TestRenderPicker(t *testing.T) {
	const W, H = 900, 500
	m := newPickerMonitor(t, W, H)
	m.pick.hsv = hsv{H: 60, S: 0.5, V: 0.5}
	bar := image.Rect(100, 130, 800, 180)
	win := image.Pt(W, H)

	hw, err := headless.NewWindow(W, H)
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	defer hw.Release()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(win)}
	m.layoutPicker(gtx, win, bar, m.pickAnchor)
	if err := hw.Frame(&ops); err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	if err := hw.Screenshot(out); err != nil {
		t.Fatal(err)
	}

	_, _, pad, svH, hueH, _ := pickerSize(gtx)
	w, _, _, _, _, _ := pickerSize(gtx)
	origin := pickerOrigin(gtx, win, bar, m.pickAnchor)
	inner := w - 2*pad
	px := func(x, y int) color.RGBA { return out.RGBAAt(origin.X+x, origin.Y+y) }
	near := func(got color.RGBA, want color.NRGBA, tol int) bool {
		d := func(a, b uint8) int {
			if a > b {
				return int(a - b)
			}
			return int(b - a)
		}
		return d(got.R, want.R) <= tol && d(got.G, want.G) <= tol && d(got.B, want.B) <= tol
	}
	// Оттенок 60 (жёлтый): справа вверху чистый жёлтый, слева вверху белый, внизу чёрный.
	if c := px(pad+inner-6, pad+6); !near(c, color.NRGBA{R: 235, G: 235, B: 20}, 45) {
		t.Errorf("правый верхний угол квадрата не жёлтый: %v", c)
	}
	if c := px(pad+6, pad+6); !near(c, color.NRGBA{R: 240, G: 240, B: 240}, 25) {
		t.Errorf("левый верхний угол квадрата не белый: %v", c)
	}
	if c := px(pad+inner/2, pad+svH-1); !near(c, color.NRGBA{}, 45) {
		t.Errorf("низ квадрата не чёрный: %v", c)
	}
	// Полоса оттенка (метка стоит на 60°, то есть на шестой части): красный в начале, зелёный
	// на трети, синий на двух третях; границы кусков градиента точные.
	hueY := pad + svH + pad/2 + hueH/2
	if c := px(pad+2, hueY); !near(c, color.NRGBA{R: 255}, 90) {
		t.Errorf("начало полосы не красное: %v", c)
	}
	if c := px(pad+inner/3, hueY); !near(c, color.NRGBA{G: 255}, 60) {
		t.Errorf("треть полосы не зелёная: %v", c)
	}
	if c := px(pad+inner*2/3, hueY); !near(c, color.NRGBA{B: 255}, 60) {
		t.Errorf("две трети полосы не синие: %v", c)
	}

	if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
		data, err := encodePNG(out)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
