package editor

import (
	"errors"
	"github.com/jexme/scrawl/internal/testenv"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jexme/scrawl/internal/collection"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
)

// sandbox подменяет HOME, кэш и внешние команды, чтобы тест не трогал буфер
// обмена и файлы пользователя. wl-copy пишет stdin в файл clipboard-<тип>.
func sandbox(t *testing.T) (home string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("нужен поддельный wl-copy (sh-скрипт), а буфер обмена на Windows ещё не написан")
	}
	home = t.TempDir()
	testenv.Home(t, home)
	testenv.Cache(t, filepath.Join(home, ".cache"))
	testenv.Config(t, filepath.Join(home, ".config"))
	// Тесты идут с поддельным wl-copy, поэтому сеанс считается Wayland, даже если
	// их запустили под xvfb-run (там WAYLAND_DISPLAY пуст, и буфер пошёл бы через xclip).
	// Окружение не трогаем: от него зависит и выбор платформы EGL.
	oldX11 := isX11
	isX11 = func() bool { return false }
	t.Cleanup(func() { isX11 = oldX11 })

	bin := t.TempDir()
	script := func(name, body string) {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// $2 — значение после --type.
	script("wl-copy", `cat > "`+home+`/clipboard-$(echo "$2" | tr '/' '_')"`)
	script("notify-send", "exit 0")
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	return home
}

func pngBytes(t *testing.T, c color.RGBA, w, h int) []byte {
	t.Helper()
	data, err := encodePNG(solid(w, h, c))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func newTestMonitor(t *testing.T) (*monitor, *bool) {
	t.Helper()
	frame := solid(400, 300, color.RGBA{R: 40, G: 90, B: 200, A: 255})
	finished := false
	m := newMonitor(Config{Fresh: true}, &screencast.Frame{Img: frame}, func() { finished = true })
	return m, &finished
}

func TestGlueCollection(t *testing.T) {
	home := sandbox(t)
	if _, err := collection.Add(pngBytes(t, color.RGBA{R: 220, G: 60, B: 60, A: 255}, 300, 200)); err != nil {
		t.Fatal(err)
	}
	m, finished := newTestMonitor(t)
	m.queued = collection.Count()
	// Текущая область станет вторым кадром: «Стало».
	m.sel, m.hasSel = image.Rect(10, 10, 250, 170), true

	m.glueCollection()
	if !*finished {
		t.Fatal("после склейки программа должна завершиться")
	}
	if collection.Count() != 0 {
		t.Error("подборка должна очиститься после склейки")
	}
	files, _ := filepath.Glob(filepath.Join(home, "Pictures", "Screenshots", "*.png"))
	if len(files) != 1 {
		t.Fatalf("ожидали один склеенный файл, получили %v", files)
	}
	f, err := os.Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	// Две картинки одной высоты (160) рядом: ширина больше высоты, есть подписи сверху.
	_, labelH := composeMetrics(settings.DefaultGlue())
	if cfg.Width < 2*200 || cfg.Height < 160+labelH {
		t.Errorf("склейка слишком мала: %dx%d", cfg.Width, cfg.Height)
	}
	clip, err := os.ReadFile(filepath.Join(home, "clipboard-image_png"))
	if err != nil || len(clip) < 100 {
		t.Errorf("в буфер обмена должна попасть картинка: %v, %d байт", err, len(clip))
	}
}

func TestGlueNeedsTwo(t *testing.T) {
	sandbox(t)
	m, finished := newTestMonitor(t)
	m.glueCollection()
	if *finished {
		t.Error("без картинок склейка не должна завершать программу")
	}
}

func TestSendSeparately(t *testing.T) {
	home := sandbox(t)
	for _, c := range []color.RGBA{{R: 220, G: 60, B: 60, A: 255}, {R: 60, G: 170, B: 80, A: 255}} {
		if _, err := collection.Add(pngBytes(t, c, 200, 100)); err != nil {
			t.Fatal(err)
		}
	}
	m, finished := newTestMonitor(t)
	m.queued = collection.Count()
	m.sel, m.hasSel = image.Rect(0, 0, 100, 80), true // третьей будет текущая область

	m.sendSeparately()
	if !*finished || collection.Count() != 0 {
		t.Fatalf("завершение=%v, в подборке осталось %d", *finished, collection.Count())
	}
	files, _ := filepath.Glob(filepath.Join(home, "Pictures", "Screenshots", "*.png"))
	if len(files) != 3 {
		t.Fatalf("ожидали 3 отдельных файла, получили %v", files)
	}
	list, err := os.ReadFile(filepath.Join(home, "clipboard-text_uri-list"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(list)), "\r\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "file://") {
		t.Errorf("в буфере должен быть список из 3 URI: %q", lines)
	}
	for _, l := range lines {
		if !strings.HasSuffix(l, "-1.png") && !strings.HasSuffix(l, "-2.png") && !strings.HasSuffix(l, "-3.png") {
			t.Errorf("неожиданное имя: %s", l)
		}
	}
}

func TestAddToCollection(t *testing.T) {
	sandbox(t)
	m, finished := newTestMonitor(t)

	m.addToCollection() // без выделения: ничего не добавляется
	if collection.Count() != 0 || *finished {
		t.Fatal("без выделения добавлять нечего")
	}
	m.sel, m.hasSel = image.Rect(5, 5, 205, 105), true
	m.addToCollection()
	if collection.Count() != 1 || !*finished {
		t.Errorf("в подборке %d, завершение=%v", collection.Count(), *finished)
	}
}

// TestNotificationsSetting: успех молчит при выключенных уведомлениях, ошибки нет.
func TestNotificationsSetting(t *testing.T) {
	sandbox(t)
	log := filepath.Join(t.TempDir(), "notify.log")
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "notify-send"), []byte("#!/bin/sh\necho \"$3\" >> "+log+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	shown := func() string { data, _ := os.ReadFile(log); return string(data) }

	m, _ := newTestMonitor(t)
	m.inform("Скриншот готов", "x")
	if !strings.Contains(shown(), "Скриншот готов") {
		t.Fatalf("по умолчанию уведомление должно показываться, лог: %q", shown())
	}

	os.Remove(log)
	if err := m.store.Update(func(s *settings.Settings) { s.Notifications = false }); err != nil {
		t.Fatal(err)
	}
	m.inform("Скриншот готов", "x")
	fail("рендер", errors.New("сбой"))
	got := shown()
	if strings.Contains(got, "Скриншот готов") {
		t.Errorf("при выключенных уведомлениях успех не должен показываться: %q", got)
	}
	if !strings.Contains(got, "Ошибка: рендер") {
		t.Errorf("ошибка должна показываться всегда: %q", got)
	}
}
