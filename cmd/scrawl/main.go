// scrawl делает скриншот области экрана с аннотациями.
//
// Первый запуск (или с -setup) показывает диалоги GNOME по одному на монитор,
// дальше сохранённые restore_token позволяют обойтись без вопросов.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"gioui.org/app"

	"github.com/jexme/scrawl/internal/editor"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
	"github.com/jexme/scrawl/internal/settingsui"
	"github.com/jexme/scrawl/internal/singleton"
	"github.com/jexme/scrawl/internal/version"
)

func main() {
	setup := flag.Bool("setup", false, "заново выбрать мониторы (покажет диалоги GNOME)")
	verbose := flag.Bool("v", false, "печатать замеры времени")
	fresh := flag.Bool("fresh", false, "не предлагать область предыдущего скриншота")
	openSettings := flag.Bool("settings", false, "открыть окно настроек")
	grant := flag.Bool("grant-screenshot", false, "разрешить приложению запасной снимок через Screenshot portal и выйти")
	exitAfter := flag.Duration("exit-after", 0, "закрыться штатно через указанное время (для проверок)")
	showVersion := flag.Bool("version", false, "показать версию и выйти")
	flag.Parse()

	if *showVersion {
		fmt.Println("scrawl", version.String())
		return
	}

	if *grant {
		if err := screencast.GrantScreenshotPermission(); err != nil {
			fatal("не удалось выдать разрешение: %v", err)
		}
		fmt.Println("Разрешение на скриншоты выдано приложению", screencast.AppID)
		return
	}

	if *openSettings {
		// Своя блокировка: окно настроек не мешает оверлею и наоборот, но второе окно
		// настроек не открываем.
		if ok, err := singleton.Acquire("scrawl-settings"); err == nil && !ok {
			return
		}
		settingsui.Run(settings.NewStore())
		return
	}

	// Повторное нажатие хоткея не должно открывать второй оверлей поверх первого
	// (и слать параллельные запросы в portal).
	if ok, err := singleton.Acquire("scrawl"); err == nil && !ok {
		return
	}

	p, err := screencast.New()
	if err != nil {
		fatal("portal: %v", err)
	}
	if !p.Registered {
		fmt.Fprintln(os.Stderr, "portal не принял app_id: restore_token работать не будет, диалоги покажутся при каждом запуске")
	}

	p.IncludeCursor = settings.Load().IncludeCursor

	var tokens []string
	if !*setup {
		tokens = screencast.LoadTokens()
	}
	// Без регистрации portal не примет сохранённые токены и покажет диалоги, которых
	// при запуске по хоткею никто не ждёт. Чаще всего это зависший portal.
	if !p.Registered && len(tokens) > 0 {
		fatal("xdg-desktop-portal не отвечает. Перезапустите его: systemctl --user restart xdg-desktop-portal")
	}

	// Старт сессий в замер не входит: в будущем их будет держать демон.
	tSessions := time.Now()
	sessions, newTokens, err := p.OpenAll(tokens, func(i, n int, again bool, left []screencast.MonitorInfo) {
		msg := screencast.SetupMessage(i, n, again, left)
		fmt.Println(">>>", msg)
		screencast.SetupNotify(msg)
	})
	if err != nil {
		fatal("сессии: %v", err)
	}
	if len(newTokens) > 0 {
		if err := screencast.SaveTokens(newTokens); err != nil {
			fmt.Fprintf(os.Stderr, "restore_token не сохранены: %v\n", err)
		}
	}
	if *setup && len(newTokens) < len(sessions) {
		fmt.Fprintln(os.Stderr, "portal не выдал restore_token: в диалоге нужна галочка «Запомнить этот выбор», иначе диалоги будут при каждом запуске")
	}
	if n, ok := p.MonitorCount(); ok && n > len(sessions) {
		fmt.Fprintf(os.Stderr, "подключено мониторов: %d, в сессиях: %d. Запустите с -setup, чтобы добавить остальные\n", n, len(sessions))
	}
	if *verbose {
		fmt.Printf("сессии (%d) готовы за %s\n", len(sessions), time.Since(tSessions).Round(time.Millisecond))
	}

	t0 := time.Now() // момент «нажатия хоткея»
	// Обычно кадры приходят за ~100 мс; если нет, экран статичен и PipeWire молчит.
	frames, fallback, err := p.GrabAll(sessions, 250*time.Millisecond)
	if err != nil {
		fatal("кадр: %v", err)
	}
	// Кадры получены: сессии захвата больше не нужны. Пока они открыты, GNOME
	// вынужден композитить экран вместо прямого вывода (это бьёт по FPS в играх).
	for _, s := range sessions {
		s.Close()
	}
	if *verbose {
		how := "PipeWire"
		if fallback {
			how = "запасной путь, Screenshot-portal"
		}
		fmt.Printf("кадры (%d) получены за %s: %s\n", len(frames), time.Since(t0).Round(time.Millisecond), how)
	}

	finish := func() { os.Exit(0) }
	closeEditor := editor.Run(editor.Config{
		Frames:  frames,
		Finish:  finish,
		T0:      t0,
		Verbose: *verbose,
		Fresh:   *fresh,
	})
	if *exitAfter > 0 {
		time.AfterFunc(*exitAfter, closeEditor)
	}
	app.Main()
}

// fatal печатает ошибку и показывает уведомление: при запуске по хоткею
// терминала нет, и без уведомления сбой остаётся незаметным.
func fatal(f string, a ...any) {
	msg := fmt.Sprintf(f, a...)
	fmt.Fprintln(os.Stderr, msg)
	exec.Command("notify-send", "-a", "Scrawl", "Не удалось сделать скриншот", msg).Run()
	os.Exit(1)
}
