package settingsui

import (
	"context"
	"runtime"
	"sync"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/scrawl/internal/autostart"
	"github.com/jexme/scrawl/internal/fonts"
	"github.com/jexme/scrawl/internal/hotkey"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/update"
	"github.com/jexme/scrawl/internal/version"
	"github.com/jexme/scrawl/internal/x11capture"
)

// Window actions that call external services (portal, gsettings). Tests
// replace them so as not to touch the system settings.
var (
	x11Session      = x11capture.IsSession
	onWindows       = runtime.GOOS == "windows"
	fontAvailable   = fonts.Available
	grantPermission = screencast.GrantScreenshotPermission
	setupMonitors   = screencast.SetupMonitors
	assignHotkey    = hotkey.Assign
	freeBuiltin     = hotkey.FreeBuiltin
	savedMonitors   = func() int { return len(screencast.LoadTokens()) }
	currentHotkey   = hotkey.Current
	hotkeyBinary    = hotkey.PreferredBinary
	checkUpdate     = func() (update.Result, error) { return update.Check(context.Background(), version.String()) }
	autostartOn     = autostart.Enabled
	autostartEnable = autostart.Enable
	autostartOff    = autostart.Disable
)

// task is a button that starts an action in the background, and a result line under it.
// The action may wait on the user for a long time (portal dialogs), so the window does not
// block: the state is changed from another goroutine and the window is redrawn.
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

// start runs run in the background if the action is not already in progress. run receives
// a function for reporting progress and returns the final message.
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

// redraw asks the window to redraw; does nothing before the window is open and in tests.
func (s *state) redraw() {
	if s.invalidate != nil {
		s.invalidate()
	}
}

// taskControl draws the action button and, under it, a message about progress or the result.
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
