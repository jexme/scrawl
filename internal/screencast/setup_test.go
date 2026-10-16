package screencast

import (
	"strings"
	"testing"
)

func TestSetupMessage(t *testing.T) {
	one := []MonitorInfo{{Name: "AOC 2276WM", X: 1920}}
	msg := SetupMessage(1, 2, false, one)
	for _, want := range []string{"Диалог 2 из 2", "«AOC 2276WM»", "справа", "Запомнить этот выбор"} {
		if !strings.Contains(msg, want) {
			t.Errorf("в подсказке нет %q: %s", want, msg)
		}
	}
	if msg := SetupMessage(0, 2, true, []MonitorInfo{{}, {}}); !strings.HasPrefix(msg, "Этот монитор уже выбран.") || !strings.Contains(msg, "который ещё не выбирали") {
		t.Errorf("повторный выбор: %s", msg)
	}
	if got := where(MonitorInfo{Primary: true}); got != "главный" {
		t.Errorf("главный монитор: %q", got)
	}
	if got := where(MonitorInfo{X: 0}); got != "слева" {
		t.Errorf("слева: %q", got)
	}
}
