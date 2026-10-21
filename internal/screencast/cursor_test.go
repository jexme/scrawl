package screencast

import "testing"

func TestCursorMode(t *testing.T) {
	// The values are set by the ScreenCast portal specification: 1 is hidden, 2 is in the frame.
	if got := cursorMode(false); got != 1 {
		t.Errorf("без курсора cursor_mode = 1, получили %d", got)
	}
	if got := cursorMode(true); got != 2 {
		t.Errorf("с курсором cursor_mode = 2, получили %d", got)
	}
}
