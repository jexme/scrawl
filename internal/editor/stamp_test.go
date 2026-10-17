package editor

import (
	"image"
	"image/color"
	"os"
	"testing"
	"time"

	"gioui.org/f32"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/screencast"
)

func newStampMonitor(t *testing.T) *monitor {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := image.NewRGBA(image.Rect(0, 0, 400, 300))
	m := newMonitor(Config{}, &screencast.Frame{Img: src}, func() {})
	m.sel, m.hasSel = image.Rect(20, 20, 380, 280), true
	m.tool = annotate.ToolStamp
	return m
}

// TestStampNumbers: клик ставит штамп с очередным номером, отмена освобождает последний номер,
// пока кнопка зажата, штамп можно подвинуть, за границу выделения он не выходит.
func TestStampNumbers(t *testing.T) {
	m := newStampMonitor(t)
	click := func(x, y float32) {
		m.press(f32.Pt(x, y))
		m.release(f32.Pt(x, y))
	}
	click(50, 50)
	click(100, 80)
	click(150, 120)
	if len(m.shapes) != 3 {
		t.Fatalf("штампов %d, ожидали 3", len(m.shapes))
	}
	for i, s := range m.shapes {
		if s.Tool != annotate.ToolStamp || s.Text != string(rune('1'+i)) {
			t.Errorf("штамп %d: %v %q", i, s.Tool, s.Text)
		}
	}
	if got := m.shapes[1].Pts[0]; got != f32.Pt(100, 80) {
		t.Errorf("штамп 2 стоит в %v, ожидали (100, 80)", got)
	}

	m.undo()
	click(200, 200)
	if n := len(m.shapes); n != 3 || m.shapes[2].Text != "3" {
		t.Errorf("после отмены номер 3 должен освободиться и достаться новому штампу: %d штампов, последний %q", n, m.shapes[2].Text)
	}

	// Пока кнопка зажата, штамп едет за курсором; номер остаётся.
	m.press(f32.Pt(60, 60))
	m.drag(f32.Pt(90, 100))
	m.drag(f32.Pt(1000, 100)) // за границу выделения
	m.release(f32.Pt(1000, 100))
	last := m.shapes[len(m.shapes)-1]
	if last.Text != "4" || last.Pts[0] != f32.Pt(380, 100) {
		t.Errorf("перетащенный штамп: %q в %v, ожидали 4 в (380, 100)", last.Text, last.Pts[0])
	}
	// Клик вне выделения штампа не ставит.
	n := len(m.shapes)
	click(5, 5)
	if len(m.shapes) != n {
		t.Error("клик вне выделения поставил штамп")
	}
}

// TestRenderStamps рисует штампы разных цветов и номеров: круг закрашен цветом, цифры контрастны.
// С SCRAWL_TEST_OUT=путь.png результат сохраняется.
func TestRenderStamps(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 420, 140))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = 200, 200, 210, 255
	}
	red, yellow, white, black := annotate.Palette[0], annotate.Palette[2], annotate.Palette[6], annotate.Palette[7]
	stamp := func(c color.NRGBA, w float32, x float32, num string) annotate.Shape {
		return annotate.Shape{Tool: annotate.ToolStamp, Color: c, Width: w, Pts: []f32.Point{{X: x, Y: 70}}, Text: num}
	}
	shapes := []annotate.Shape{
		stamp(red, 4, 40, "1"), stamp(yellow, 4, 110, "2"), stamp(white, 4, 180, "10"),
		stamp(black, 4, 250, "123"), stamp(annotate.Palette[4], 8, 340, "7"),
	}
	out, err := render(src, src.Bounds(), shapes, annotate.NewShaper())
	if err != nil {
		t.Skipf("headless рендер недоступен: %v", err)
	}

	count := func(cx, cy, r int, match func(color.RGBA) bool) int {
		n := 0
		for y := cy - r; y <= cy+r; y++ {
			for x := cx - r; x <= cx+r; x++ {
				if match(out.RGBAAt(x, y)) {
					n++
				}
			}
		}
		return n
	}
	light := func(c color.RGBA) bool { return c.R > 235 && c.G > 235 && c.B > 235 }
	dark := func(c color.RGBA) bool { return c.R < 60 && c.G < 60 && c.B < 60 }
	// Красный штамп: слева от цифры круг красный, в центре белые цифры.
	if c := out.RGBAAt(40-annotateRadius(4)+4, 70); c.R < 180 || c.G > 90 {
		t.Errorf("красный штамп не закрашен: %v", c)
	}
	if n := count(40, 70, 6, light); n < 8 {
		t.Errorf("на красном штампе нет белой цифры: %d светлых пикселей", n)
	}
	// Жёлтый штамп: светлый фон, цифры тёмные.
	if n := count(110, 70, 6, dark); n < 8 {
		t.Errorf("на жёлтом штампе нет тёмной цифры: %d тёмных пикселей", n)
	}
	// Белый штамп: цифры тёмные. Чёрный штамп: цифры белые.
	if n := count(180, 70, 8, dark); n < 12 {
		t.Errorf("на белом штампе нет тёмных цифр: %d", n)
	}
	if n := count(250, 70, 9, light); n < 12 {
		t.Errorf("на чёрном штампе нет белых цифр: %d", n)
	}
	// Вне круга фон не тронут.
	if got, want := out.RGBAAt(5, 5), src.RGBAAt(5, 5); got != want {
		t.Errorf("фон изменился: %v, было %v", got, want)
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

func annotateRadius(w float32) int { return int(annotate.StampRadius(w)) }

// TestStampDigitsCentered: цифры стоят в середине круга: середина их габарита не дальше двух пикселей от центра.
func TestStampDigitsCentered(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = 128, 128, 128, 255
	}
	blue := annotate.Palette[4]
	for _, num := range []string{"1", "2", "7", "8", "10", "42", "123"} {
		for _, w := range []float32{2, 4, 8, 16} {
			shape := annotate.Shape{Tool: annotate.ToolStamp, Color: blue, Width: w, Pts: []f32.Point{{X: 100, Y: 100}}, Text: num}
			out, err := render(src, src.Bounds(), []annotate.Shape{shape}, annotate.NewShaper())
			if err != nil {
				t.Skipf("headless рендер недоступен: %v", err)
			}
			r := annotate.StampRadius(w)
			minX, minY, maxX, maxY := 1<<20, 1<<20, -1, -1
			for y := 0; y < 200; y++ {
				for x := 0; x < 200; x++ {
					dx, dy := float32(x-100), float32(y-100)
					if inner := r*0.84 - 2; dx*dx+dy*dy > inner*inner { // внутри цветного круга, без ободка
						continue
					}
					if c := out.RGBAAt(x, y); c.R > 200 && c.G > 200 && c.B > 200 {
						minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
					}
				}
			}
			if maxX < 0 {
				t.Errorf("номер %q, толщина %v: цифр не видно", num, w)
				continue
			}
			cx, cy := float32(minX+maxX+1)/2, float32(minY+maxY+1)/2
			if dx, dy := cx-100, cy-100; dx < -2 || dx > 2 || dy < -2 || dy > 2 {
				t.Errorf("номер %q, толщина %v: середина цифр (%.1f, %.1f), а центр круга (100, 100)", num, w, cx, cy)
			}
		}
	}
}

// TestBlurLevelsInEditor: у размытия «−» и «+» двигают уровень 1-6 и упираются в его границы, по
// умолчанию уровень 2, а плашка называет режим.
func TestBlurLevelsInEditor(t *testing.T) {
	m := newStampMonitor(t)
	m.tool = annotate.ToolBlur
	if got := m.curWidth(); got != annotate.DefaultBlurLevel {
		t.Errorf("уровень по умолчанию %v, ожидали %d", got, annotate.DefaultBlurLevel)
	}
	for i := 0; i < 20; i++ {
		m.bumpWidth(1)
	}
	if got := m.curWidth(); got != annotate.MaxBlurLevel {
		t.Errorf("после множества «+» уровень %v, ожидали %d", got, annotate.MaxBlurLevel)
	}
	for i := 0; i < 20; i++ {
		m.bumpWidth(-1)
	}
	if got := m.curWidth(); got != annotate.MinWidth {
		t.Errorf("после множества «−» уровень %v, ожидали %d", got, annotate.MinWidth)
	}
	// Остальные инструменты по-прежнему доходят до 30.
	m.tool = annotate.ToolPen
	for i := 0; i < 50; i++ {
		m.bumpWidth(1)
	}
	if got := m.curWidth(); got != annotate.MaxWidth {
		t.Errorf("перо: %v, ожидали %d", got, annotate.MaxWidth)
	}
}

// scrollMonitor делает монитор с ручными часами для проверки прокрутки.
func scrollMonitor(t *testing.T, tool annotate.Tool) (*monitor, *time.Time) {
	t.Helper()
	m := newStampMonitor(t)
	m.tool = tool
	now := time.Unix(1000, 0)
	m.clock = func() time.Time { return now }
	return m, &now
}

// TestScrollTouchpad: тачпад шлёт мелкие события потоком, но толщина двигается порциями и не
// перескакивает с 1 на 6 за один жест.
func TestScrollTouchpad(t *testing.T) {
	m, now := scrollMonitor(t, annotate.ToolBlur)
	m.widths[annotate.ToolBlur] = 1

	// Быстрый жест: двести мелких событий по 3 единицы (минус это прокрутка вверх, то есть «больше») за 0,4 с.
	for i := 0; i < 200; i++ {
		*now = now.Add(2 * time.Millisecond)
		m.scroll(-3)
	}
	// 0,4 с между шагами не меньше четверти секунды: не больше двух шагов (раньше было пять).
	if got := m.curWidth(); got < 2 || got > 3 {
		t.Errorf("быстрый жест двинул уровень 1 -> %v, ожидали 2 или 3", got)
	}

	// Инерция после жеста: затухающий хвост (3, 2.85, 2.7... до нуля) уровень почти не двигает.
	before := m.curWidth()
	for v := float32(3); v > 0.05; v *= 0.95 {
		*now = now.Add(8 * time.Millisecond)
		m.scroll(-v)
	}
	if got := m.curWidth(); got > before+1 {
		t.Errorf("инерция увела уровень %v -> %v", before, got)
	}

	// Медленный непрерывный жест (около 100 единиц в секунду) за секунду даёт не больше четырёх шагов.
	m.widths[annotate.ToolBlur] = 1
	*now = now.Add(time.Second)
	for i := 0; i < 100; i++ {
		*now = now.Add(10 * time.Millisecond)
		m.scroll(-1)
	}
	if got := m.curWidth(); got < 2 || got > 5 {
		t.Errorf("медленный жест за секунду: уровень 1 -> %v, ожидали от 2 до 5", got)
	}
}

// TestScrollWheel: щелчок колеса (одно событие около 10 единиц) это ровно один шаг, как раньше.
func TestScrollWheel(t *testing.T) {
	m, now := scrollMonitor(t, annotate.ToolBlur)
	m.widths[annotate.ToolBlur] = 1
	for want := float32(2); want <= annotate.MaxBlurLevel; want++ {
		*now = now.Add(400 * time.Millisecond)
		m.scroll(-10)
		if got := m.curWidth(); got != want {
			t.Fatalf("щелчок вверх: уровень %v, ожидали %v", got, want)
		}
	}
	*now = now.Add(400 * time.Millisecond)
	m.scroll(-10)
	if got := m.curWidth(); got != annotate.MaxBlurLevel {
		t.Errorf("за последним уровнем: %v", got)
	}
	*now = now.Add(400 * time.Millisecond)
	m.scroll(10)
	if got := m.curWidth(); got != annotate.MaxBlurLevel-1 {
		t.Errorf("щелчок вниз: %v", got)
	}
	// Крупное событие (колесо с высоким разрешением даёт и 120) тоже один шаг.
	*now = now.Add(400 * time.Millisecond)
	m.scroll(-120)
	if got := m.curWidth(); got != annotate.MaxBlurLevel {
		t.Errorf("крупное событие: %v, ожидали один шаг до %d", got, annotate.MaxBlurLevel)
	}

	// У инструментов с тонкой шкалой щелчки идут чаще.
	m.tool = annotate.ToolPen
	m.widths[annotate.ToolPen] = 4
	for i := 0; i < 5; i++ {
		*now = now.Add(60 * time.Millisecond)
		m.scroll(-10)
	}
	if got := m.curWidth(); got != 9 {
		t.Errorf("пять быстрых щелчков пером: %v, ожидали 9", got)
	}
}

// TestScrollResets: пауза и смена направления сбрасывают накопленное, так что случайное касание
// не двигает толщину.
func TestScrollResets(t *testing.T) {
	m, now := scrollMonitor(t, annotate.ToolBlur)
	m.widths[annotate.ToolBlur] = 3
	for i := 0; i < 3; i++ {
		m.scroll(-7)
	}
	m.scroll(7)  // назад: накопленное обнуляется
	m.scroll(-7) // и снова вперёд
	if got := m.curWidth(); got != 3 {
		t.Errorf("туда-обратно двинуло толщину: %v", got)
	}
	*now = now.Add(time.Second) // долгая пауза: накопленное забыто
	for i := 0; i < 5; i++ {
		m.scroll(-7) // 35 единиц: меньше порции
	}
	if got := m.curWidth(); got != 3 {
		t.Errorf("после паузы накопленное не забылось: %v", got)
	}
	m.scroll(-7) // 42: порция набрана
	if got := m.curWidth(); got != 4 {
		t.Errorf("порция набрана, а уровень %v, ожидали 4", got)
	}
}
