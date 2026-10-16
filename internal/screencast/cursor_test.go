package screencast

import "testing"

func TestCursorMode(t *testing.T) {
	// Значения заданы спецификацией ScreenCast portal: 1 — скрыт, 2 — в кадре.
	if got := cursorMode(false); got != 1 {
		t.Errorf("без курсора cursor_mode = 1, получили %d", got)
	}
	if got := cursorMode(true); got != 2 {
		t.Errorf("с курсором cursor_mode = 2, получили %d", got)
	}
}
