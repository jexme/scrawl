package editor

import (
	"image"
	"testing"

	"gioui.org/f32"

	"github.com/jexme/screen-editor/internal/screencast"
)

var bounds = image.Rect(0, 0, 1000, 800)

func TestHitHandle(t *testing.T) {
	r := image.Rect(100, 100, 300, 200)
	cases := []struct {
		name  string
		p     f32.Point
		bands bool
		want  edge
		hit   bool
	}{
		{"угол левый верхний", f32.Pt(101, 99), false, edge{-1, -1}, true},
		{"правый нижний угол", f32.Pt(299, 201), false, edge{1, 1}, true},
		{"середина правой стороны без полос не ручка", f32.Pt(300, 150), false, edge{}, false},
		{"середина верхней стороны без полос не ручка", f32.Pt(200, 100), false, edge{}, false},
		{"середина стороны с полосами", f32.Pt(300, 150), true, edge{1, 0}, true},
		{"край не у ручки без полос", f32.Pt(300, 120), false, edge{}, false},
		{"край по всей длине с полосами", f32.Pt(300, 120), true, edge{1, 0}, true},
		{"угол с полосами", f32.Pt(300, 200), true, edge{1, 1}, true},
		{"внутри", f32.Pt(200, 150), true, edge{}, false},
		{"снаружи", f32.Pt(600, 600), true, edge{}, false},
	}
	for _, c := range cases {
		got, hit := hitHandle(r, c.p, 8, c.bands)
		if hit != c.hit || got != c.want {
			t.Errorf("%s: получили %v/%v, ожидали %v/%v", c.name, got, hit, c.want, c.hit)
		}
	}
}

func TestApplyResize(t *testing.T) {
	start := image.Rect(100, 100, 300, 200)
	got := applyResize(start, edge{1, 1}, f32.Pt(400, 350), bounds)
	if want := image.Rect(100, 100, 400, 350); got != want {
		t.Errorf("правый нижний угол: %v, ожидали %v", got, want)
	}
	// Нельзя вывернуть рамку: левая сторона упирается в правую минус минимум.
	got = applyResize(start, edge{-1, 0}, f32.Pt(500, 150), bounds)
	if got.Dx() != minSelection || got.Max.X != 300 {
		t.Errorf("левая сторона за правую: %v", got)
	}
	// Выход за кадр обрезается.
	got = applyResize(start, edge{1, 1}, f32.Pt(5000, 5000), bounds)
	if got.Max != bounds.Max {
		t.Errorf("за границей кадра: %v", got)
	}
}

func TestApplyMove(t *testing.T) {
	start := image.Rect(100, 100, 300, 200)
	if got, want := applyMove(start, 50, -20, bounds), image.Rect(150, 80, 350, 180); got != want {
		t.Errorf("сдвиг: %v, ожидали %v", got, want)
	}
	got := applyMove(start, -500, 900, bounds)
	if got.Min.X != 0 || got.Max.Y != 800 || got.Size() != start.Size() {
		t.Errorf("сдвиг за границы: %v", got)
	}
}

func TestLastRegionRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	f := &screencast.Frame{X: 1920, Y: 0, Img: image.NewRGBA(image.Rect(0, 0, 2560, 1440))}

	if _, ok := loadLastRegion(f); ok {
		t.Fatal("область не должна существовать до сохранения")
	}
	want := image.Rect(10, 20, 510, 320)
	saveLastRegion(f, want)
	if got, ok := loadLastRegion(f); !ok || got != want {
		t.Fatalf("после сохранения: %v/%v, ожидали %v", got, ok, want)
	}

	// Другой монитор или другой размер кадра: область не подходит.
	other := &screencast.Frame{X: 0, Y: 0, Img: image.NewRGBA(image.Rect(0, 0, 2560, 1440))}
	if _, ok := loadLastRegion(other); ok {
		t.Error("область чужого монитора не должна загружаться")
	}
	resized := &screencast.Frame{X: 1920, Y: 0, Img: image.NewRGBA(image.Rect(0, 0, 1920, 1080))}
	if _, ok := loadLastRegion(resized); ok {
		t.Error("область при другом размере кадра не должна загружаться")
	}
}
