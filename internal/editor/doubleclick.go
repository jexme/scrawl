package editor

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"

	"gioui.org/f32"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/settings"
)

// doubleClickWindow is the maximum time between two presses for a double click.
const doubleClickWindow = 400 * time.Millisecond

// isDoubleClick remembers a press and reports whether it was a second consecutive click
// near the first one.
func (m *monitor) isDoubleClick(p f32.Point) bool {
	now := time.Now()
	dist := math.Hypot(float64(p.X-m.lastClickPos.X), float64(p.Y-m.lastClickPos.Y))
	dbl := !m.lastClickAt.IsZero() && now.Sub(m.lastClickAt) < doubleClickWindow && dist < float64(m.tol)*1.5
	m.lastClickPos = p
	m.lastClickAt = now
	if dbl {
		m.lastClickAt = time.Time{} // a third click does not count as a new double click
	}
	return dbl
}

// doubleClickAction performs the double-click action from the settings. It returns true if
// the click was handled: only inside the selection and not on the border. It works with
// any tool except text and stamps (those are clicked repeatedly): after an area is
// selected the default tool is enabled, and the click must not depend on it.
func (m *monitor) doubleClickAction(p f32.Point) bool {
	mode := m.store.Get().DoubleClick
	if mode == settings.DoubleClickOff || !m.hasSel || m.tool == annotate.ToolText || m.tool == annotate.ToolStamp || !inRect(p, m.sel) {
		return false
	}
	if _, onEdge := hitHandle(m.sel, p, m.tol, true); onEdge {
		return false
	}
	m.dropClickDot()
	m.export(true, mode == settings.DoubleClickCopySave)
	return true
}

// dropClickDot removes the dot that the first click of a double click left with the pen or
// marker: the other tools draw nothing on a click without movement.
func (m *monitor) dropClickDot() {
	if n := len(m.shapes); n > 0 {
		last := m.shapes[n-1]
		if (last.Tool == annotate.ToolPen || last.Tool == annotate.ToolMarker) && len(last.Pts) == 1 {
			m.shapes = m.shapes[:n-1]
		}
	}
}

// openSettingsWindow closes the overlay and opens a separate settings window:
// a full-screen overlay would cover it.
func (m *monitor) openSettingsWindow() {
	exe, err := os.Executable()
	if err == nil {
		err = exec.Command(exe, "-settings").Start()
	}
	if err != nil {
		fail(i18n.T("settings"), fmt.Errorf(i18n.T("could not open the settings window: %w"), err))
		return
	}
	m.quit()
}
