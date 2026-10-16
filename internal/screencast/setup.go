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
		return i18n.T("главный")
	}
	if m.X == 0 {
		return i18n.T("слева")
	}
	return i18n.T("справа")
}

// SetupMessage объясняет, что выбрать в очередном диалоге GNOME при настройке
// мониторов: i — номер диалога с нуля, n — их число, again — предыдущий выбор
// повторил уже выбранный монитор, left — мониторы, которые ещё не выбраны.
func SetupMessage(i, n int, again bool, left []MonitorInfo) string {
	msg := fmt.Sprintf(i18n.T("Диалог %d из %d: "), i+1, n)
	if again {
		msg = i18n.T("Этот монитор уже выбран. ") + msg
	}
	if len(left) == 1 {
		msg += fmt.Sprintf(i18n.T("выберите монитор «%s» (%s)"), left[0].Name, where(left[0]))
	} else {
		msg += i18n.T("выберите монитор, который ещё не выбирали")
	}
	return msg + i18n.T(", оставьте галочку «Запомнить этот выбор» и нажмите «Дать доступ»")
}

// SetupNotify показывает уведомление о настройке мониторов: диалоги GNOME
// появляются поверх всего, и пользователю нужно подсказать, что в них выбрать.
func SetupNotify(msg string) {
	exec.Command("notify-send", "-a", "Scrawl", i18n.T("Настройка мониторов"), msg).Run()
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
		return "", fmt.Errorf(i18n.T("сессии: %w"), err)
	}
	for _, s := range sessions {
		s.Close()
	}
	if len(tokens) < len(sessions) {
		return "", errors.New(i18n.T("portal не выдал токен восстановления: в диалоге нужна галочка «Запомнить этот выбор», иначе диалоги будут при каждом запуске"))
	}
	if err := SaveTokens(tokens); err != nil {
		return "", fmt.Errorf(i18n.T("токены не сохранены: %w"), err)
	}
	msg := fmt.Sprintf(i18n.T("Мониторов настроено: %d"), len(sessions))
	if n, ok := p.MonitorCount(); ok && n > len(sessions) {
		msg += fmt.Sprintf(i18n.T(" из %d подключённых, выберите остальные ещё раз"), n)
	}
	return msg, nil
}
