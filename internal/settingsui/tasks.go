package settingsui

import (
	"sync"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/scrawl/internal/hotkey"
	"github.com/jexme/scrawl/internal/screencast"
)

// Действия окна, которые вызывают внешние службы (portal, gsettings). Тесты
// подменяют их, чтобы не трогать настройки системы.
var (
	grantPermission = screencast.GrantScreenshotPermission
	setupMonitors   = screencast.SetupMonitors
	assignHotkey    = hotkey.Assign
	currentHotkey   = hotkey.Current
	hotkeyBinary    = hotkey.PreferredBinary
)

// task — кнопка, запускающая действие в фоне, и строка с результатом под ней.
// Действие может долго ждать пользователя (диалоги portal), поэтому окно не
// блокируется: состояние меняется из другой горутины, а окно перерисовывается.
type task struct {
	btn widget.Clickable

	mu      sync.Mutex
	running bool
	msg     string
	failed  bool
}

func (t *task) status() (msg string, failed, running bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.msg, t.failed, t.running
}

func (t *task) set(msg string, failed bool) {
	t.mu.Lock()
	t.msg, t.failed = msg, failed
	t.mu.Unlock()
}

// start запускает run в фоне, если действие ещё не идёт. run получает функцию,
// которой можно сообщать о ходе работы, и возвращает итоговое сообщение.
func (s *state) start(t *task, run func(progress func(string)) (string, error)) {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running, t.msg, t.failed = true, "", false
	t.mu.Unlock()

	go func() {
		msg, err := run(func(p string) {
			t.set(p, false)
			s.redraw()
		})
		t.mu.Lock()
		t.running = false
		if err != nil {
			t.msg, t.failed = err.Error(), true
		} else {
			t.msg, t.failed = msg, false
		}
		t.mu.Unlock()
		s.redraw()
	}()
}

// redraw просит окно перерисоваться; до открытия окна и в тестах ничего не делает.
func (s *state) redraw() {
	if s.invalidate != nil {
		s.invalidate()
	}
}

// taskControl рисует кнопку действия и под ней сообщение о ходе или результате.
func (s *state) taskControl(t *task, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		msg, failed, running := t.status()
		b := material.Button(s.th, &t.btn, label)
		b.Background = borderColor
		if running {
			b.Background = mutedColor
		}
		children := []layout.FlexChild{layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx, layout.Rigid(b.Layout))
		})}
		if msg != "" {
			c := mutedColor
			if failed {
				c = errorColor
			}
			children = append(children,
				layout.Rigid(layout.Spacer{Height: 6}.Layout),
				layout.Rigid(s.label(material.Caption, msg, c)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}
