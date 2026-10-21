package screencast

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jexme/scrawl/internal/i18n"
)

// MonitorInfo describes a logical monitor, from GNOME Mutter data.
type MonitorInfo struct {
	X, Y    int
	Name    string // manufacturer and model, for example "AOC 2276WM"
	Primary bool
}

// Monitors returns the logical monitors from GNOME Mutter data.
// The second value is false if they could not be determined (not GNOME).
func (p *Portal) Monitors() ([]MonitorInfo, bool) {
	call := p.conn.Object("org.gnome.Mutter.DisplayConfig", "/org/gnome/Mutter/DisplayConfig").
		Call("org.gnome.Mutter.DisplayConfig.GetCurrentState", 0)
	if call.Err != nil || len(call.Body) < 3 {
		return nil, false
	}
	logical, ok := call.Body[2].([][]any)
	if !ok {
		return nil, false
	}
	var out []MonitorInfo
	for _, lm := range logical {
		if len(lm) < 6 {
			continue
		}
		x, _ := lm[0].(int32)
		y, _ := lm[1].(int32)
		primary, _ := lm[4].(bool)
		info := MonitorInfo{X: int(x), Y: int(y), Primary: primary}
		if mons, ok := lm[5].([][]any); ok && len(mons) > 0 && len(mons[0]) >= 3 {
			vendor, _ := mons[0][1].(string)
			product, _ := mons[0][2].(string)
			info.Name = strings.TrimSpace(vendor + " " + product)
		}
		out = append(out, info)
	}
	return out, len(out) > 0
}

// MonitorCount returns the number of logical monitors.
func (p *Portal) MonitorCount() (int, bool) {
	m, ok := p.Monitors()
	return len(m), ok
}

// positions returns the monitor positions reported by the session.
func (s *Session) positions() [][2]int {
	var out [][2]int
	for _, st := range s.streams {
		x, y, _ := pair(st.Props, "position")
		out = append(out, [2]int{x, y})
	}
	return out
}

// OpenAll starts one session per monitor. GNOME's dialog hands out one
// monitor per session, so each monitor needs its own restore_token.
//
// If restore is empty, one dialog per monitor is shown; before each one
// prompt(i, n, again, left) is called: again says that the chosen monitor
// was already picked and the dialog is being shown again, left lists the monitors that
// are not yet chosen (if they could be determined). It returns the sessions and the current
// tokens (they must be saved via SaveTokens).
func (p *Portal) OpenAll(restore []string, prompt func(i, n int, again bool, left []MonitorInfo)) ([]*Session, []string, error) {
	if len(restore) > 0 {
		return p.restoreAll(restore)
	}

	monitors, ok := p.Monitors()
	n := len(monitors)
	if !ok || n < 1 {
		n = 1
	}
	var (
		sessions []*Session
		tokens   []string
		seen     = map[[2]int]bool{}
	)
	fail := func(err error) ([]*Session, []string, error) {
		for _, s := range sessions {
			s.Close()
		}
		return nil, nil, err
	}
	retries := 0
	for i := 0; i < n; i++ {
		if prompt != nil {
			var left []MonitorInfo
			for _, m := range monitors {
				if !seen[[2]int{m.X, m.Y}] {
					left = append(left, m)
				}
			}
			prompt(i, n, retries > 0, left)
		}
		s, tok, err := p.NewSession("")
		if err != nil {
			return fail(err)
		}
		dup := false
		for _, pos := range s.positions() {
			dup = dup || seen[pos]
		}
		if dup {
			s.Close()
			if retries++; retries > 2 {
				return fail(errors.New(i18n.T("the same monitor was chosen several times in a row")))
			}
			i--
			continue
		}
		for _, pos := range s.positions() {
			seen[pos] = true
		}
		retries = 0
		sessions = append(sessions, s)
		if tok != "" {
			tokens = append(tokens, tok)
		}
		// If the dialog returned several monitors at once, the rest are not needed.
		if len(seen) >= n {
			break
		}
	}
	return sessions, tokens, nil
}

// restoreAll brings up the sessions from the saved tokens. Requests to the portal go
// strictly one at a time: starting by token takes milliseconds, while parallel
// calls have been seen to hang xdg-desktop-portal.
func (p *Portal) restoreAll(restore []string) ([]*Session, []string, error) {
	var (
		sessions []*Session
		tokens   []string
	)
	for _, t := range restore {
		s, tok, err := p.NewSession(t)
		if err != nil {
			for _, open := range sessions {
				open.Close()
			}
			return nil, nil, err
		}
		sessions = append(sessions, s)
		if tok == "" {
			tok = t
		}
		tokens = append(tokens, tok)
	}
	return sessions, tokens, nil
}

// GrabAll gets frames from all monitors of all sessions. First the fast path
// through PipeWire (tens of milliseconds); if no frame arrives within deadline
// (a static screen), the Screenshot portal is used and the result is cut up by monitor.
// The second value is true if the fallback path was needed.
func (p *Portal) GrabAll(sessions []*Session, deadline time.Duration) ([]Frame, bool, error) {
	results := make([][]Frame, len(sessions))
	errs := make([]error, len(sessions))
	var wg sync.WaitGroup
	for i, s := range sessions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = s.Grab(deadline)
		}()
	}
	wg.Wait()

	var (
		all    []Frame
		failed error
	)
	for i, e := range errs {
		if e != nil {
			failed = e
			continue
		}
		all = append(all, results[i]...)
	}
	if failed == nil {
		return all, false, nil
	}

	frames, err := p.grabViaScreenshot(sessions)
	if err != nil {
		return nil, true, fmt.Errorf(i18n.T("PipeWire: %v; Screenshot: %w. If the portal refused, grant the permission with: scrawl -grant-screenshot"), failed, err)
	}
	return frames, true, nil
}
