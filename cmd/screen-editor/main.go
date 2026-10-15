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
)

func main() {
	setup := flag.Bool("setup", false, "заново выбрать мониторы (покажет диалоги GNOME)")
	verbose := flag.Bool("v", false, "печатать замеры времени")
	flag.Parse()

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
	frames, err := screencast.GrabAll(sessions)
	if err != nil {
		fatal("кадр: %v", err)
	}
	if *verbose {
		fmt.Printf("кадры (%d) получены за %s\n", len(frames), time.Since(t0).Round(time.Millisecond))
	}

	editor.Run(editor.Config{
		Frames: frames,
		Finish: func() {
			for _, s := range sessions {
				s.Close()
			}
			os.Exit(0)
		},
		T0:      t0,
		Verbose: *verbose,
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
