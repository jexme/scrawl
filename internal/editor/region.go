package editor

import (
	"encoding/json"
	"image"
	"math"
	"os"
	"path/filepath"

	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/jexme/screen-editor/internal/screencast"
)

// minSelection — минимальный размер выделения в пикселях кадра.
const minSelection = 4

// edge задаёт, какие стороны рамки тянет пользователь: -1 левая/верхняя,
// 1 правая/нижняя, 0 сторона не меняется.
type edge struct{ hx, hy int }

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }

// hitHandle проверяет, попала ли точка p в ручку рамки r с допуском tol (пиксели кадра).
// С bands=true рамка захватывается по всей длине сторон (инструмент выделения),
// иначе только за четыре угловые ручки (чтобы не мешать рисованию).
func hitHandle(r image.Rectangle, p f32.Point, tol float32, bands bool) (edge, bool) {
	minX, maxX := float32(r.Min.X), float32(r.Max.X)
	minY, maxY := float32(r.Min.Y), float32(r.Max.Y)
	near := func(a, b float32) bool { return abs32(a-b) <= tol }
	inX := p.X >= minX-tol && p.X <= maxX+tol
	inY := p.Y >= minY-tol && p.Y <= maxY+tol

	var e edge
	if bands {
		switch {
		case near(p.X, minX) && inY:
			e.hx = -1
		case near(p.X, maxX) && inY:
			e.hx = 1
		}
		switch {
		case near(p.Y, minY) && inX:
			e.hy = -1
		case near(p.Y, maxY) && inX:
			e.hy = 1
		}
		return e, e != edge{}
	}

	// Без полос рамка берётся только за углы: именно они нарисованы ручками.
	corners := [4]struct {
		x, y   float32
		hx, hy int
	}{{minX, minY, -1, -1}, {maxX, minY, 1, -1}, {minX, maxY, -1, 1}, {maxX, maxY, 1, 1}}
	for _, c := range corners {
		if near(p.X, c.x) && near(p.Y, c.y) {
			return edge{c.hx, c.hy}, true
		}
	}
	return edge{}, false
}

// cornerPoints возвращает четыре угла рамки: центры ручек.
func cornerPoints(r image.Rectangle) []f32.Point {
	return []f32.Point{
		f32.Pt(float32(r.Min.X), float32(r.Min.Y)), f32.Pt(float32(r.Max.X), float32(r.Min.Y)),
		f32.Pt(float32(r.Min.X), float32(r.Max.Y)), f32.Pt(float32(r.Max.X), float32(r.Max.Y)),
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// applyResize двигает стороны рамки start, указанные в e, к точке p, не выходя
// за bounds и не делая рамку меньше minSelection.
func applyResize(start image.Rectangle, e edge, p f32.Point, bounds image.Rectangle) image.Rectangle {
	r := start
	px := clampInt(int(math.Round(float64(p.X))), bounds.Min.X, bounds.Max.X)
	py := clampInt(int(math.Round(float64(p.Y))), bounds.Min.Y, bounds.Max.Y)
	switch e.hx {
	case -1:
		r.Min.X = min(px, r.Max.X-minSelection)
	case 1:
		r.Max.X = max(px, r.Min.X+minSelection)
	}
	switch e.hy {
	case -1:
		r.Min.Y = min(py, r.Max.Y-minSelection)
	case 1:
		r.Max.Y = max(py, r.Min.Y+minSelection)
	}
	return r
}

// applyMove сдвигает рамку start на (dx, dy), не выпуская её за bounds.
func applyMove(start image.Rectangle, dx, dy int, bounds image.Rectangle) image.Rectangle {
	r := start.Add(image.Pt(dx, dy))
	if r.Min.X < bounds.Min.X {
		r = r.Add(image.Pt(bounds.Min.X-r.Min.X, 0))
	}
	if r.Max.X > bounds.Max.X {
		r = r.Add(image.Pt(bounds.Max.X-r.Max.X, 0))
	}
	if r.Min.Y < bounds.Min.Y {
		r = r.Add(image.Pt(0, bounds.Min.Y-r.Min.Y))
	}
	if r.Max.Y > bounds.Max.Y {
		r = r.Add(image.Pt(0, bounds.Max.Y-r.Max.Y))
	}
	return r
}

// cursorFor подбирает курсор для края e.
func cursorFor(e edge) pointer.Cursor {
	switch {
	case e.hx != 0 && e.hy != 0 && e.hx == e.hy:
		return pointer.CursorNorthWestSouthEastResize
	case e.hx != 0 && e.hy != 0:
		return pointer.CursorNorthEastSouthWestResize
	case e.hx != 0:
		return pointer.CursorEastWestResize
	default:
		return pointer.CursorNorthSouthResize
	}
}

// lastRegion — запомненная область предыдущего скриншота.
type lastRegion struct {
	X, Y int    // позиция монитора
	W, H int    // размер кадра монитора
	Sel  [4]int // x0, y0, x1, y1 в пикселях кадра
}

func lastRegionPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "screen-editor", "last-region.json")
}

// saveLastRegion запоминает область, чтобы следующий запуск показал её сразу.
func saveLastRegion(f *screencast.Frame, sel image.Rectangle) {
	b := f.Img.Bounds()
	data, err := json.Marshal(lastRegion{
		X: f.X, Y: f.Y, W: b.Dx(), H: b.Dy(),
		Sel: [4]int{sel.Min.X, sel.Min.Y, sel.Max.X, sel.Max.Y},
	})
	if err != nil {
		return
	}
	path := lastRegionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	os.WriteFile(path, data, 0o600)
}

// forgetLastRegion стирает запомненную область.
func forgetLastRegion() { os.Remove(lastRegionPath()) }

// rememberRegion запоминает текущую область, если это не отключено в настройках;
// при отключённой настройке старая запись стирается, чтобы не осталась лежать.
func (m *monitor) rememberRegion() {
	if !m.hasSel {
		return
	}
	if m.store.Get().RememberRegion {
		saveLastRegion(m.frame, m.sel)
	} else {
		forgetLastRegion()
	}
}

// loadLastRegion возвращает запомненную область, если она относилась к этому
// монитору с тем же размером кадра и по-прежнему допустима.
func loadLastRegion(f *screencast.Frame) (image.Rectangle, bool) {
	data, err := os.ReadFile(lastRegionPath())
	if err != nil {
		return image.Rectangle{}, false
	}
	var lr lastRegion
	if json.Unmarshal(data, &lr) != nil {
		return image.Rectangle{}, false
	}
	b := f.Img.Bounds()
	if lr.X != f.X || lr.Y != f.Y || lr.W != b.Dx() || lr.H != b.Dy() {
		return image.Rectangle{}, false
	}
	r := image.Rect(lr.Sel[0], lr.Sel[1], lr.Sel[2], lr.Sel[3]).Intersect(b)
	if r.Dx() < minSelection || r.Dy() < minSelection {
		return image.Rectangle{}, false
	}
	return r, true
}
