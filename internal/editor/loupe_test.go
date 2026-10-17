package editor

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
)

func TestLoupeFocus(t *testing.T) {
	bounds := image.Rect(0, 0, 1920, 1080)
	sel := image.Rect(100, 200, 500, 600)
	for _, tc := range []struct {
		name     string
		e        edge
		creating bool
		hover    f32.Point
		want     f32.Point
	}{
		{"левый верхний угол прилипает к стороне, а не к курсору", edge{-1, -1}, false, f32.Pt(98.4, 203.6), f32.Pt(100, 200)},
		{"правая сторона: по Y курсор округляется", edge{1, 0}, false, f32.Pt(503, 300.6), f32.Pt(500, 301)},
		{"нижняя сторона: по X курсор округляется", edge{0, 1}, false, f32.Pt(250.4, 604), f32.Pt(250, 600)},
		{"новая рамка берёт целую часть, как rectOf", edge{}, true, f32.Pt(300.9, 400.9), f32.Pt(300, 400)},
		{"новая рамка у края не выходит за кадр", edge{}, true, f32.Pt(-5, 2000), f32.Pt(0, 1080)},
	} {
		if got := loupeFocus(sel, tc.e, tc.creating, tc.hover, bounds); got != tc.want {
			t.Errorf("%s: %v, ожидали %v", tc.name, got, tc.want)
		}
	}
}

func TestLoupeActive(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	newM := func() *monitor { return &monitor{store: settings.NewStore()} }
	for _, tc := range []struct {
		name  string
		setup func(m *monitor)
		want  bool
	}{
		{"целятся: рамки нет, курсор над окном", func(m *monitor) { m.hoverIn = true }, true},
		{"курсор ушёл с этого монитора", func(m *monitor) {}, false},
		{"первое выделение тянут", func(m *monitor) { m.dragging = true }, true},
		{"готовую рамку растягивают", func(m *monitor) { m.hasSel, m.adjust = true, adjustResize }, true},
		{"область есть, курсор просто над окном", func(m *monitor) { m.hasSel, m.hoverIn = true, true }, false},
		{"область двигают целиком", func(m *monitor) { m.hasSel, m.adjust, m.hoverIn = true, adjustMove, true }, false},
	} {
		m := newM()
		tc.setup(m)
		if got := m.loupeActive(); got != tc.want {
			t.Errorf("%s: %v, ожидали %v", tc.name, got, tc.want)
		}
	}

	// Выключенная в настройках лупа не показывается ни в одном режиме.
	m := newM()
	if err := m.store.Update(func(c *settings.Settings) { c.ShowLoupe = false }); err != nil {
		t.Fatal(err)
	}
	m.hoverIn, m.dragging = true, true
	if m.loupeActive() {
		t.Error("show_loupe=false должен выключать лупу")
	}
}

func TestLoupeDir(t *testing.T) {
	start := f32.Pt(100, 100)
	for _, tc := range []struct {
		name     string
		e        edge
		creating bool
		hover    f32.Point
		want     image.Point
	}{
		{"новая рамка вправо-вниз", edge{}, true, f32.Pt(200, 150), image.Pt(1, 1)},
		{"новая рамка влево-вверх", edge{}, true, f32.Pt(50, 20), image.Pt(-1, -1)},
		{"новая рамка влево-вниз", edge{}, true, f32.Pt(50, 150), image.Pt(-1, 1)},
		{"левый верхний угол готовой рамки", edge{-1, -1}, false, f32.Pt(0, 0), image.Pt(-1, -1)},
		{"правая сторона: по Y лупа по центру", edge{1, 0}, false, f32.Pt(0, 0), image.Pt(1, 0)},
	} {
		if got := loupeDir(tc.e, tc.creating, start, tc.hover); got != tc.want {
			t.Errorf("%s: %v, ожидали %v", tc.name, got, tc.want)
		}
	}
}

func TestLoupePlacement(t *testing.T) {
	win := image.Pt(1000, 800)
	const side, gap = 160, 30
	for _, tc := range []struct {
		name string
		cur  image.Point
		dir  image.Point
		want image.Point // левый верхний угол
	}{
		{"справа снизу от курсора", image.Pt(100, 100), image.Pt(1, 1), image.Pt(130, 130)},
		{"слева сверху от курсора", image.Pt(500, 400), image.Pt(-1, -1), image.Pt(500-gap-side, 400-gap-side)},
		{"справа, по вертикали по центру", image.Pt(500, 400), image.Pt(1, 0), image.Pt(530, 400-side/2)},
		{"у правого края переворачивается влево", image.Pt(950, 100), image.Pt(1, 1), image.Pt(950-gap-side, 130)},
		{"у нижнего края переворачивается вверх", image.Pt(100, 780), image.Pt(1, 1), image.Pt(130, 780-gap-side)},
		{"у левого края переворачивается вправо", image.Pt(20, 400), image.Pt(-1, -1), image.Pt(50, 400-gap-side)},
		{"в правом нижнем углу", image.Pt(990, 790), image.Pt(1, 1), image.Pt(990-gap-side, 790-gap-side)},
	} {
		r := loupePlacement(tc.cur, side, gap, win, tc.dir)
		if r.Min != tc.want || r.Dx() != side || r.Dy() != side {
			t.Errorf("%s: %v, ожидали угол %v", tc.name, r, tc.want)
		}
		if !r.In(image.Rectangle{Max: win}) {
			t.Errorf("%s: лупа %v вышла за окно", tc.name, r)
		}
	}
	// Окно меньше лупы: прижимаем к нулю, а не уходим в минус.
	if r := loupePlacement(image.Pt(5, 5), side, gap, image.Pt(100, 100), image.Pt(1, 1)); r.Min != (image.Point{}) {
		t.Errorf("в маленьком окне: %v", r)
	}
}

// TestRenderLoupe рисует лупу над кадром в виде шахматки 1×1 px: и для готовой рамки, и для
// первого выделения. Проверяет, что пиксели увеличены без сглаживания, фокус стоит в центре,
// а лупа не закрывает выделение. С SCRAWL_TEST_OUT=путь.png результаты сохраняются.
func TestRenderLoupe(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const W, H = 700, 500
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			c := color.RGBA{R: 230, G: 230, B: 230, A: 255}
			if (x+y)%2 == 0 {
				c = color.RGBA{R: 20, G: 90, B: 160, A: 255}
			}
			src.SetRGBA(x, y, c)
		}
	}
	for _, tc := range []struct {
		name  string
		setup func(m *monitor)
	}{
		{"resize", func(m *monitor) {
			m.sel, m.adjust, m.adjEdge, m.hover = image.Rect(100, 80, 260, 200), adjustResize, edge{1, 1}, f32.Pt(260.2, 199.8)
		}},
		{"aiming", func(m *monitor) {
			m.hoverIn, m.hover = true, f32.Pt(300.6, 200.3)
		}},
		{"first-selection", func(m *monitor) {
			m.sel, m.dragging, m.selStart, m.hover = image.Rect(350, 260, 450, 330), true, f32.Pt(450, 330), f32.Pt(350.4, 260.7)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &monitor{frame: &screencast.Frame{Img: src}, store: settings.NewStore()}
			tc.setup(m)

			win, err := headless.NewWindow(W, H)
			if err != nil {
				t.Skipf("headless рендер недоступен: %v", err)
			}
			defer win.Release()
			var ops op.Ops
			clipArea := clip.Rect(src.Bounds()).Push(&ops)
			paint.NewImageOp(src).Add(&ops)
			paint.PaintOp{}.Add(&ops)
			clipArea.Pop()
			gtx := layout.Context{Ops: &ops, Metric: identity, Constraints: layout.Exact(image.Pt(W, H))}
			m.layoutLoupe(gtx, image.Pt(W, H), 1, 1)
			if err := win.Frame(&ops); err != nil {
				t.Skipf("headless рендер недоступен: %v", err)
			}
			out := image.NewRGBA(image.Rect(0, 0, W, H))
			if err := win.Screenshot(out); err != nil {
				t.Fatal(err)
			}

			side := int(loupeCells * loupeZoom)
			start := m.selStart
			if m.loupeAiming() {
				start = m.hover // целятся: области ещё нет, лупа справа снизу
			}
			dir := loupeDir(m.adjEdge, m.dragging || m.loupeAiming(), start, m.hover)
			box := loupePlacement(image.Pt(int(m.hover.X), int(m.hover.Y)), side, gtx.Dp(28), image.Pt(W, H), dir)
			if box.Overlaps(m.sel) {
				t.Errorf("лупа %v закрывает выделение %v", box, m.sel)
			}
			// Границы ячеек лежат на кратных loupeZoom от левого верхнего угла лупы: пробы внутри
			// соседних ячеек, вдали от рамки, различаются, а внутри одной ячейки совпадают
			// (при сглаживании на границе были бы промежуточные цвета).
			z := int(loupeZoom)
			a := out.RGBAAt(box.Min.X+z+z/2, box.Min.Y+z+z/2)
			b := out.RGBAAt(box.Min.X+z+z/2+z, box.Min.Y+z+z/2)
			if a == b {
				t.Errorf("соседние пиксели кадра в лупе одного цвета: %v", a)
			}
			if a2 := out.RGBAAt(box.Min.X+z+z/2+1, box.Min.Y+z+z/2+1); a2 != a {
				t.Errorf("пиксель кадра в лупе не залит ровно: %v и %v", a, a2)
			}
			// Кадр вне лупы не тронут.
			if got, want := out.RGBAAt(5, 5), src.RGBAAt(5, 5); got != want {
				t.Errorf("вне лупы пиксель изменился: %v, было %v", got, want)
			}
			// Крестик в центре лупы красный.
			cx, cy := box.Min.X+box.Dx()/2, box.Min.Y+box.Dy()/2
			if c := out.RGBAAt(cx+4, cy); c.R < 150 || c.G > 120 {
				t.Errorf("крестик в центре лупы не найден: %v", c)
			}

			if path := os.Getenv("SCRAWL_TEST_OUT"); path != "" {
				data, err := encodePNG(out)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path+"."+tc.name+".png", data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
