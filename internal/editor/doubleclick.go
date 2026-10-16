package editor

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"

	"gioui.org/f32"

	"github.com/jexme/scrawl/internal/annotate"
	"github.com/jexme/scrawl/internal/settings"
)

// doubleClickWindow — максимальное время между двумя нажатиями для двойного клика.
const doubleClickWindow = 400 * time.Millisecond

// isDoubleClick запоминает нажатие и сообщает, был ли это второй клик подряд
// рядом с первым.
func (m *monitor) isDoubleClick(p f32.Point) bool {
	now := time.Now()
	dist := math.Hypot(float64(p.X-m.lastClickPos.X), float64(p.Y-m.lastClickPos.Y))
	dbl := !m.lastClickAt.IsZero() && now.Sub(m.lastClickAt) < doubleClickWindow && dist < float64(m.tol)*1.5
	m.lastClickPos = p
	m.lastClickAt = now
	if dbl {
		m.lastClickAt = time.Time{} // третий клик не считается новым двойным
	}
	return dbl
}

// doubleClickAction выполняет действие двойного клика из настроек. Возвращает
// true, если клик обработан: только внутри выделения, с инструментом
// «выделение» и не на рамке.
func (m *monitor) doubleClickAction(p f32.Point) bool {
	mode := m.store.Get().DoubleClick
	if mode == settings.DoubleClickOff || !m.hasSel || m.tool != annotate.ToolSelect || !inRect(p, m.sel) {
		return false
	}
	if _, onEdge := hitHandle(m.sel, p, m.tol, true); onEdge {
		return false
	}
	m.export(true, mode == settings.DoubleClickCopySave)
	return true
}

// openSettingsWindow закрывает оверлей и открывает отдельное окно настроек:
// полноэкранный оверлей закрыл бы его собой.
func (m *monitor) openSettingsWindow() {
	exe, err := os.Executable()
	if err == nil {
		err = exec.Command(exe, "-settings").Start()
	}
	if err != nil {
		fail("настройки", fmt.Errorf("не удалось открыть окно настроек: %w", err))
		return
	}
	m.quit()
}
