// scrawl делает скриншот области экрана с аннотациями.
//
// Первый запуск (или с -setup) показывает диалоги GNOME по одному на монитор,
// дальше сохранённые restore_token позволяют обойтись без вопросов.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"gioui.org/app"

	"github.com/jexme/scrawl/internal/editor"
	"github.com/jexme/scrawl/internal/fonts"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
	"github.com/jexme/scrawl/internal/settingsui"
	"github.com/jexme/scrawl/internal/singleton"
	"github.com/jexme/scrawl/internal/tray"
	"github.com/jexme/scrawl/internal/update"
	"github.com/jexme/scrawl/internal/version"
	"github.com/jexme/scrawl/internal/x11capture"
)

func main() {
	i18n.SetLanguage(settings.Load().Language) // до флагов: у них тоже есть описания
	fonts.Preload(i18n.Language())             // шрифт для иероглифов грузится, пока идёт захват
	setup := flag.Bool("setup", false, i18n.T("заново выбрать мониторы (покажет диалоги GNOME)"))
	verbose := flag.Bool("v", false, i18n.T("печатать замеры времени"))
	fresh := flag.Bool("fresh", false, i18n.T("не предлагать область предыдущего скриншота"))
	openSettings := flag.Bool("settings", false, i18n.T("открыть окно настроек"))
	grant := flag.Bool("grant-screenshot", false, i18n.T("разрешить приложению запасной снимок через Screenshot portal и выйти"))
	exitAfter := flag.Duration("exit-after", 0, i18n.T("закрыться штатно через указанное время (для проверок)"))
	trayMode := flag.Bool("tray", false, i18n.T("держать значок в области уведомлений (для запуска при входе в систему)"))
	showVersion := flag.Bool("version", false, i18n.T("показать версию и выйти"))
	flag.Parse()

	if *showVersion {
		fmt.Println("scrawl", version.String())
		return
	}

	if *trayMode {
		runTray(*exitAfter)
		return
	}

	if *grant && x11capture.IsSession() {
		fmt.Println(i18n.T("в сеансе X11 разрешение не нужно: захват идёт напрямую"))
		return
	}
	if *grant {
		if err := screencast.GrantScreenshotPermission(); err != nil {
			fatal(i18n.T("не удалось выдать разрешение: %v"), err)
		}
		fmt.Println(i18n.T("Разрешение на скриншоты выдано приложению"), screencast.AppID)
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

	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		// Захват экрана, буфер обмена и горячая клавиша для этой системы ещё не написаны.
		fatal("%s", i18n.T("захват экрана пока есть только в Linux (GNOME, Wayland) и Windows; окно настроек работает: scrawl -settings"))
	}

	var (
		frames []screencast.Frame
		t0     time.Time
	)
	if runtime.GOOS == "windows" {
		frames, t0 = grabWindows(*verbose)
	} else if x11capture.IsSession() {
		frames, t0 = grabX11(*setup, *verbose)
	} else {
		frames, t0 = grabPortal(*setup, *verbose)
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

// grabX11 снимает экран напрямую из X11: без портала, диалогов и токенов.
func grabX11(setup, verbose bool) ([]screencast.Frame, time.Time) {
	if setup {
		fmt.Fprintln(os.Stderr, i18n.T("в сеансе X11 мониторы выбирать не нужно: захват идёт напрямую"))
	}
	t0 := time.Now() // момент «нажатия хоткея»
	frames, err := x11capture.GrabAll()
	if err != nil {
		fatal(i18n.T("кадр: %v"), err)
	}
	if verbose {
		fmt.Printf("кадры (%d) получены за %s: X11\n", len(frames), time.Since(t0).Round(time.Millisecond))
	}
	return frames, t0
}

// grabPortal снимает экран через ScreenCast portal (Wayland): сначала поднимает
// сессии по сохранённым токенам, потом берёт по кадру с каждого монитора.
func grabPortal(setup, verbose bool) ([]screencast.Frame, time.Time) {
	p, err := screencast.New()
	if err != nil {
		fatal("portal: %v", err)
	}
	if p.OldPortal {
		// Старый portal узнаёт приложение по systemd-scope: без него токены не работают.
		if err := p.EnterAppScope(); err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("portal не принял app_id: restore_token работать не будет, диалоги покажутся при каждом запуске"))
		}
	} else if !p.Registered {
		fmt.Fprintln(os.Stderr, i18n.T("portal не принял app_id: restore_token работать не будет, диалоги покажутся при каждом запуске"))
	}

	p.IncludeCursor = settings.Load().IncludeCursor

	var tokens []string
	if !setup {
		tokens = screencast.LoadTokens()
	}
	// Без регистрации portal не примет сохранённые токены и покажет диалоги, которых
	// при запуске по хоткею никто не ждёт. Чаще всего это зависший portal.
	// У portal старше 1.21 Registry нет вовсе: это не зависание, токены он принимает
	// по app_id из systemd-scope, поэтому идём дальше.
	if !p.Registered && !p.OldPortal && len(tokens) > 0 {
		fatal("%s", i18n.T("xdg-desktop-portal не отвечает. Перезапустите его: systemctl --user restart xdg-desktop-portal"))
	}

	// Старт сессий в замер не входит: в будущем их будет держать демон.
	tSessions := time.Now()
	sessions, newTokens, err := p.OpenAll(tokens, func(i, n int, again bool, left []screencast.MonitorInfo) {
		msg := screencast.SetupMessage(i, n, again, left)
		fmt.Println(">>>", msg)
		screencast.SetupNotify(msg)
	})
	if err != nil {
		fatal(i18n.T("сессии: %v"), err)
	}
	if len(newTokens) > 0 {
		if err := screencast.SaveTokens(newTokens); err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("restore_token не сохранены: %v\n"), err)
		}
	}
	if setup && len(newTokens) < len(sessions) {
		fmt.Fprintln(os.Stderr, i18n.T("portal не выдал restore_token: в диалоге нужна галочка «Запомнить этот выбор», иначе диалоги будут при каждом запуске"))
	}
	if n, ok := p.MonitorCount(); ok && n > len(sessions) {
		fmt.Fprintf(os.Stderr, i18n.T("подключено мониторов: %d, в сессиях: %d. Запустите с -setup, чтобы добавить остальные\n"), n, len(sessions))
	}
	if verbose {
		fmt.Printf("сессии (%d) готовы за %s\n", len(sessions), time.Since(tSessions).Round(time.Millisecond))
	}

	t0 := time.Now() // момент «нажатия хоткея»
	// Обычно кадры приходят за ~100 мс; если нет, экран статичен и PipeWire молчит.
	frames, fallback, err := p.GrabAll(sessions, 250*time.Millisecond)
	if err != nil {
		fatal(i18n.T("кадр: %v"), err)
	}
	// Кадры получены: сессии захвата больше не нужны. Пока они открыты, GNOME
	// вынужден композитить экран вместо прямого вывода (это бьёт по FPS в играх).
	for _, s := range sessions {
		s.Close()
	}
	if verbose {
		how := "PipeWire"
		if fallback {
			how = i18n.T("запасной путь, Screenshot-portal")
		}
		fmt.Printf("кадры (%d) получены за %s: %s\n", len(frames), time.Since(t0).Round(time.Millisecond), how)
	}

	return frames, t0
}

// fatal печатает ошибку и показывает уведомление: при запуске по хоткею
// терминала нет, и без уведомления сбой остаётся незаметным.
func fatal(f string, a ...any) {
	msg := fmt.Sprintf(f, a...)
	fmt.Fprintln(os.Stderr, msg)
	notify(i18n.T("Не удалось сделать скриншот"), msg)
	os.Exit(1)
}

// runTray держит значок в области уведомлений: снимок и настройки запускаются
// отдельными процессами, поэтому значок остаётся лёгким и не держит захват экрана.
func runTray(exitAfter time.Duration) {
	if ok, err := singleton.Acquire("scrawl-tray"); err == nil && !ok {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		fatal("%v", err)
	}
	spawn := func(args ...string) {
		cmd := exec.Command(exe, args...)
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		go cmd.Wait() // чтобы не оставлять зомби-процесс
	}
	var stop chan struct{}
	if exitAfter > 0 {
		stop = make(chan struct{})
		time.AfterFunc(exitAfter, func() { close(stop) })
	}
	if settings.Load().CheckUpdates {
		go watchUpdates()
	}
	err = tray.Run(tray.Handlers{
		Shot:     func() { spawn() },
		Settings: func() { spawn("-settings") },
	}, stop)
	if err != nil {
		fatal("%s: %v", i18n.T("значок"), err)
	}
}

// watchUpdates проверяет новую версию при старте значка и потом раз в сутки; о
// каждой новой версии сообщает один раз. Работает, только если проверка включена в настройках.
func watchUpdates() {
	for {
		if res, err := update.Check(context.Background(), version.String()); err == nil && res.Newer && !update.AlreadyNotified(res.Latest) {
			notify(i18n.T("Доступна новая версия"), res.Message())
			update.MarkNotified(res.Latest)
		}
		time.Sleep(24 * time.Hour)
		if !settings.Load().CheckUpdates {
			return // выключили в настройках
		}
	}
}
