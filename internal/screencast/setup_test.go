package screencast

import (
	"strings"
	"testing"
)

func TestSetupMessage(t *testing.T) {
	one := []MonitorInfo{{Name: "AOC 2276WM", X: 1920}}
	msg := SetupMessage(1, 2, false, one)
	for _, want := range []string{"Dialog 2 of 2", "\"AOC 2276WM\"", "right", "Remember this choice"} {
		if !strings.Contains(msg, want) {
			t.Errorf("в подсказке нет %q: %s", want, msg)
		}
	}
	if msg := SetupMessage(0, 2, true, []MonitorInfo{{}, {}}); !strings.HasPrefix(msg, "This monitor is already chosen.") || !strings.Contains(msg, "you have not chosen yet") {
		t.Errorf("повторный выбор: %s", msg)
	}
	if got := where(MonitorInfo{Primary: true}); got != "primary" {
		t.Errorf("главный монитор: %q", got)
	}
	if got := where(MonitorInfo{X: 0}); got != "left" {
		t.Errorf("слева: %q", got)
	}
}
