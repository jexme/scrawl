// screen-editor делает скриншот области экрана с аннотациями.
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

	"github.com/jexme/screen-editor/internal/editor"
	"github.com/jexme/screen-editor/internal/screencast"
	"github.com/jexme/screen-editor/internal/singleton"
)

func main() {
	setup := flag.Bool("setup", false, "заново выбрать мониторы (покажет диалоги GNOME)")
	verbose := flag.Bool("v", false, "печатать замеры времени")
	fresh := flag.Bool("fresh", false, "не предлагать область предыдущего скриншота")
	exitAfter := flag.Duration("exit-after", 0, "закрыться штатно через указанное время (для проверок)")
	flag.Parse()

	// Повторное нажатие хоткея не должно открывать второй оверлей поверх первого
	// (и слать параллельные запросы в portal).
	if ok, err := singleton.Acquire("screen-editor"); err == nil && !ok {
		return
	}

	p, err := screencast.New()
	if err != nil {
		fatal("portal: %v", err)
	}
	if !p.Registered {
		fmt.Fprintln(os.Stderr, "portal не принял app_id: restore_token работать не будет, диалоги покажутся при каждом запуске")
	}

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
		msg := fmt.Sprintf("Диалог %d из %d: ", i+1, n)
		if again {
			msg = "Этот монитор уже выбран. " + msg
		}
		if len(left) == 1 {
			msg += fmt.Sprintf("выберите монитор «%s» (%s)", left[0].Name, where(left[0], left))
		} else {
			msg += "выберите монитор, который ещё не выбирали"
		}
		msg += ", оставьте галочку «Запомнить этот выбор» и нажмите «Дать доступ»"
		fmt.Println(">>>", msg)
		exec.Command("notify-send", "-a", "Screen Editor", "Настройка мониторов", msg).Run()
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
	if *verbose {
		how := "PipeWire"
		if fallback {
			how = "запасной путь, Screenshot-portal"
		}
		fmt.Printf("кадры (%d) получены за %s: %s\n", len(frames), time.Since(t0).Round(time.Millisecond), how)
	}

	finish := func() {
		for _, s := range sessions {
			s.Close()
		}
		os.Exit(0)
	}
	if *exitAfter > 0 {
		time.AfterFunc(*exitAfter, finish)
	}
	editor.Run(editor.Config{
		Frames:  frames,
		Finish:  finish,
		T0:      t0,
		Verbose: *verbose,
		Fresh:   *fresh,
	})
	app.Main()
}

// where описывает положение единственного оставшегося монитора словами.
func where(m screencast.MonitorInfo, _ []screencast.MonitorInfo) string {
	if m.Primary {
		return "главный"
	}
	if m.X == 0 {
		return "слева"
	}
	return "справа"
}

// fatal печатает ошибку и показывает уведомление: при запуске по хоткею
// терминала нет, и без уведомления сбой остаётся незаметным.
func fatal(f string, a ...any) {
	msg := fmt.Sprintf(f, a...)
	fmt.Fprintln(os.Stderr, msg)
	exec.Command("notify-send", "-a", "Screen Editor", "Не удалось сделать скриншот", msg).Run()
	os.Exit(1)
}
