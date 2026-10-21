package screencast

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/jexme/scrawl/internal/i18n"
)

// where описывает положение единственного оставшегося монитора словами.
func where(m MonitorInfo) string {
	if m.Primary {
		return i18n.T("primary")
	}
	if m.X == 0 {
		return i18n.T("left")
	}
	return i18n.T("right")
}

// SetupMessage объясняет, что выбрать в очередном диалоге GNOME при настройке
// мониторов: i — номер диалога с нуля, n — их число, again — предыдущий выбор
// повторил уже выбранный монитор, left — мониторы, которые ещё не выбраны.
func SetupMessage(i, n int, again bool, left []MonitorInfo) string {
	msg := fmt.Sprintf(i18n.T("Dialog %d of %d: "), i+1, n)
	if again {
		msg = i18n.T("This monitor is already chosen. ") + msg
	}
	if len(left) == 1 {
		msg += fmt.Sprintf(i18n.T("choose the monitor \"%s\" (%s)"), left[0].Name, where(left[0]))
	} else {
		msg += i18n.T("choose a monitor you have not chosen yet")
	}
	return msg + i18n.T(", keep \"Remember this choice\" ticked and press \"Share\"")
}

// SetupNotify показывает уведомление о настройке мониторов: диалоги GNOME
// появляются поверх всего, и пользователю нужно подсказать, что в них выбрать.
func SetupNotify(msg string) {
	exec.Command("notify-send", "-a", "Scrawl", i18n.T("Monitor setup"), msg).Run()
}

// SetupMonitors заново выбирает мониторы: показывает диалоги GNOME по одному на
// монитор и сохраняет токены восстановления. progress получает подсказку к
// каждому диалогу. Возвращает итог для показа пользователю; ошибка есть, если
// выбор не удалось сохранить.
func SetupMonitors(progress func(string)) (string, error) {
	p, err := New()
	if err != nil {
		return "", fmt.Errorf("portal: %w", err)
	}
	sessions, tokens, err := p.OpenAll(nil, func(i, n int, again bool, left []MonitorInfo) {
		msg := SetupMessage(i, n, again, left)
		progress(msg)
		SetupNotify(msg)
	})
	if err != nil {
		return "", fmt.Errorf(i18n.T("sessions: %w"), err)
	}
	for _, s := range sessions {
		s.Close()
	}
	if len(tokens) < len(sessions) {
		return "", errors.New(i18n.T("the portal did not issue a restore token: tick \"Remember this choice\" in the dialog, otherwise dialogs will appear on every run"))
	}
	if err := SaveTokens(tokens); err != nil {
		return "", fmt.Errorf(i18n.T("tokens not saved: %w"), err)
	}
	msg := fmt.Sprintf(i18n.T("Monitors set up: %d"), len(sessions))
	if n, ok := p.MonitorCount(); ok && n > len(sessions) {
		msg += fmt.Sprintf(i18n.T(" of %d connected, choose the rest again"), n)
	}
	return msg, nil
}
