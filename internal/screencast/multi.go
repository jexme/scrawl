package screencast

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// MonitorInfo описывает логический монитор по данным GNOME Mutter.
type MonitorInfo struct {
	X, Y    int
	Name    string // производитель и модель, например "AOC 2276WM"
	Primary bool
}

// Monitors возвращает логические мониторы по данным GNOME Mutter.
// Второе значение false, если узнать не удалось (не GNOME).
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

// MonitorCount возвращает число логических мониторов.
func (p *Portal) MonitorCount() (int, bool) {
	m, ok := p.Monitors()
	return len(m), ok
}

// positions возвращает позиции мониторов, которые отдаёт сессия.
func (s *Session) positions() [][2]int {
	var out [][2]int
	for _, st := range s.streams {
		x, y, _ := pair(st.Props, "position")
		out = append(out, [2]int{x, y})
	}
	return out
}

// OpenAll запускает по сессии на каждый монитор. GNOME отдаёт в диалоге один
// монитор за сессию, поэтому на каждый монитор нужен свой restore_token.
//
// Если restore пуст, показываются диалоги по числу мониторов; перед каждым
// вызывается prompt(i, n, again, left): again сообщает, что выбранный монитор
// уже был и диалог показывается повторно, left — мониторы, которые ещё не
// выбраны (если их удалось определить). Возвращаются сессии и актуальные
// токены (их нужно сохранить через SaveTokens).
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
				return fail(fmt.Errorf("выбран один и тот же монитор несколько раз подряд"))
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
		// Если диалог вернул сразу несколько мониторов, остальные не нужны.
		if len(seen) >= n {
			break
		}
	}
	return sessions, tokens, nil
}

// restoreAll поднимает сессии по сохранённым токенам. Запросы к portal идут
// строго по одному: старт по токену занимает миллисекунды, а параллельные
// вызовы замечены в зависании xdg-desktop-portal.
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

// GrabAll получает кадры со всех мониторов всех сессий. Сначала быстрый путь
// через PipeWire (десятки миллисекунд); если за deadline кадр не пришёл
// (статичный экран), снимается Screenshot-portal и режется по мониторам.
// Второе значение true, если понадобился запасной путь.
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
		return nil, true, fmt.Errorf("PipeWire: %v; Screenshot: %w. Если portal отказал, выдайте разрешение командой: scrawl -grant-screenshot", failed, err)
	}
	return frames, true, nil
}
