// screencastspike измеряет задержку получения кадра через ScreenCast portal + PipeWire.
//
// D-Bus часть (сессия, источники, старт, fd) делает Go. Сам кадр читает
// gst-launch-1.0 (pipewiresrc) из выданного fd: libpipewire-dev на машине нет.
// Накладные расходы запуска GStreamer измеряются отдельно как baseline.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest = "org.freedesktop.portal.Desktop"
	portalPath = "/org/freedesktop/portal/desktop"
	scIface    = "org.freedesktop.portal.ScreenCast"
	reqIface   = "org.freedesktop.portal.Request"
	sessIface  = "org.freedesktop.portal.Session"
)

type portal struct {
	conn   *dbus.Conn
	sigs   chan *dbus.Signal
	sender string
	n      int
}

type stream struct {
	ID    uint32
	Props map[string]dbus.Variant
}

type session struct {
	handle  string
	streams []stream
	token   string // новый restore_token
	tCreate time.Duration
	tSelect time.Duration
	tStart  time.Duration
}

type frame struct {
	stream uint32
	first  time.Duration // до первого байта
	done   time.Duration // до полного кадра
	bytes  int
	live   bool // в кадре есть ненулевые пиксели
}

func main() {
	cold := flag.Int("cold", 3, "замеров с новой сессией на каждый")
	warm := flag.Int("warm", 5, "замеров при живой сессии (новое подключение к потоку)")
	reset := flag.Bool("reset", false, "игнорировать сохранённый restore_token (покажет диалог)")
	flag.Parse()

	p, err := newPortal()
	if err != nil {
		fatal("portal: %v", err)
	}

	tokenFile := tokenPath()
	token := ""
	if !*reset {
		if b, err := os.ReadFile(tokenFile); err == nil {
			token = strings.TrimSpace(string(b))
		}
	}
	if token == "" {
		fmt.Println(">>> restore_token нет: GNOME покажет диалог выбора экрана, подтвердите его (60 с)")
	}

	base := baseline()
	fmt.Printf("baseline gst-launch (без PipeWire): %s\n\n", ms(base))

	fmt.Println("== новая сессия на каждый кадр (хоткей -> кадр, худший случай) ==")
	var coldTotals []time.Duration
	var s *session
	for i := 0; i < *cold; i++ {
		if s != nil {
			p.closeSession(s)
		}
		t0 := time.Now()
		s, err = p.newSession(token)
		if err != nil {
			fatal("сессия: %v", err)
		}
		if s.token != "" {
			token = s.token
			saveToken(tokenFile, token)
		}
		tSetup := time.Since(t0)
		frames, tGrab, err := p.grabAll(s)
		if err != nil {
			fatal("кадр: %v", err)
		}
		total := tSetup + tGrab
		coldTotals = append(coldTotals, total)
		fmt.Printf("#%d create=%s select=%s start=%s | кадр(%d)=%s | итого=%s (минус baseline %s)\n",
			i+1, ms(s.tCreate), ms(s.tSelect), ms(s.tStart), len(frames), ms(tGrab), ms(total), ms(total-base))
		report(frames)
	}
	summary("cold", coldTotals, base)

	fmt.Println("\n== живая сессия, новое подключение к потоку (демон держит сессию) ==")
	var warmTotals []time.Duration
	for i := 0; i < *warm; i++ {
		frames, tGrab, err := p.grabAll(s)
		if err != nil {
			fatal("кадр: %v", err)
		}
		warmTotals = append(warmTotals, tGrab)
		fmt.Printf("#%d кадр(%d)=%s (минус baseline %s)\n", i+1, len(frames), ms(tGrab), ms(tGrab-base))
		report(frames)
	}
	summary("warm", warmTotals, base)

	p.closeSession(s)
	fmt.Println("\nготово. Проверьте, горел ли в панели GNOME индикатор записи экрана, пока сессия была жива")
}

func newPortal() (*portal, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	err = conn.AddMatchSignal(dbus.WithMatchInterface(reqIface), dbus.WithMatchMember("Response"))
	if err != nil {
		return nil, err
	}
	ch := make(chan *dbus.Signal, 32)
	conn.Signal(ch)
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	return &portal{conn: conn, sigs: ch, sender: sender}, nil
}

// request вызывает метод ScreenCast, возвращающий Request, и ждёт его Response.
func (p *portal) request(method string, timeout time.Duration, lead []any, extra map[string]dbus.Variant) (map[string]dbus.Variant, error) {
	p.n++
	token := fmt.Sprintf("se%d_%d", os.Getpid(), p.n)
	opts := map[string]dbus.Variant{"handle_token": dbus.MakeVariant(token)}
	for k, v := range extra {
		opts[k] = v
	}
	reqPath := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", p.sender, token))

	call := p.conn.Object(portalDest, portalPath).Call(scIface+"."+method, 0, append(lead, opts)...)
	if call.Err != nil {
		return nil, fmt.Errorf("%s: %w", method, call.Err)
	}
	deadline := time.After(timeout)
	for {
		select {
		case sig := <-p.sigs:
			if sig.Path != reqPath || sig.Name != reqIface+".Response" {
				continue
			}
			code, _ := sig.Body[0].(uint32)
			res, _ := sig.Body[1].(map[string]dbus.Variant)
			if code != 0 {
				return nil, fmt.Errorf("%s: портал вернул код %d (отмена или отказ)", method, code)
			}
			return res, nil
		case <-deadline:
			return nil, fmt.Errorf("%s: таймаут %v", method, timeout)
		}
	}
}

func (p *portal) newSession(restore string) (*session, error) {
	s := &session{}

	t := time.Now()
	res, err := p.request("CreateSession", 10*time.Second, nil, map[string]dbus.Variant{
		"session_handle_token": dbus.MakeVariant(fmt.Sprintf("sess%d_%d", os.Getpid(), p.n)),
	})
	if err != nil {
		return nil, err
	}
	s.tCreate = time.Since(t)
	s.handle, _ = res["session_handle"].Value().(string)
	if s.handle == "" {
		return nil, fmt.Errorf("в ответе нет session_handle")
	}
	sh := dbus.ObjectPath(s.handle)

	sel := map[string]dbus.Variant{
		"types":        dbus.MakeVariant(uint32(1)), // monitor
		"multiple":     dbus.MakeVariant(true),
		"cursor_mode":  dbus.MakeVariant(uint32(1)), // курсор скрыт
		"persist_mode": dbus.MakeVariant(uint32(2)), // помнить выбор
	}
	if restore != "" {
		sel["restore_token"] = dbus.MakeVariant(restore)
	}
	t = time.Now()
	if _, err = p.request("SelectSources", 10*time.Second, []any{sh}, sel); err != nil {
		return nil, err
	}
	s.tSelect = time.Since(t)

	t = time.Now()
	res, err = p.request("Start", 120*time.Second, []any{sh, ""}, nil)
	if err != nil {
		return nil, err
	}
	s.tStart = time.Since(t)

	if err := res["streams"].Store(&s.streams); err != nil {
		return nil, fmt.Errorf("разбор streams: %w", err)
	}
	if len(s.streams) == 0 {
		return nil, fmt.Errorf("портал не вернул потоков")
	}
	s.token, _ = res["restore_token"].Value().(string)
	return s, nil
}

func (p *portal) closeSession(s *session) {
	p.conn.Object(portalDest, dbus.ObjectPath(s.handle)).Call(sessIface+".Close", 0)
}

func (p *portal) openRemote(s *session) (*os.File, error) {
	var fd dbus.UnixFD
	call := p.conn.Object(portalDest, portalPath).
		Call(scIface+".OpenPipeWireRemote", 0, dbus.ObjectPath(s.handle), map[string]dbus.Variant{})
	if err := call.Store(&fd); err != nil {
		return nil, fmt.Errorf("OpenPipeWireRemote: %w", err)
	}
	return os.NewFile(uintptr(fd), "pipewire-remote"), nil
}

// grabAll получает по одному кадру со всех потоков параллельно.
// Время считается от OpenPipeWireRemote до последнего готового кадра.
func (p *portal) grabAll(s *session) ([]frame, time.Duration, error) {
	start := time.Now()
	files := make([]*os.File, len(s.streams))
	for i := range s.streams {
		f, err := p.openRemote(s)
		if err != nil {
			return nil, 0, err
		}
		files[i] = f
	}

	frames := make([]frame, len(s.streams))
	errs := make([]error, len(s.streams))
	var wg sync.WaitGroup
	for i, st := range s.streams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			frames[i], errs[i] = grab(st.ID, files[i], start)
		}()
	}
	wg.Wait()
	total := time.Since(start)
	for _, e := range errs {
		if e != nil {
			return nil, 0, e
		}
	}
	return frames, total, nil
}

// grab читает один кадр BGRx из PipeWire-узла через gst-launch-1.0.
func grab(node uint32, remote *os.File, start time.Time) (frame, error) {
	f := frame{stream: node}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gst-launch-1.0", "-q",
		"pipewiresrc", "fd=3", "path="+strconv.Itoa(int(node)), "num-buffers=1",
		"!", "videoconvert", "!", "video/x-raw,format=BGRx", "!", "fdsink", "fd=1")
	cmd.ExtraFiles = []*os.File{remote}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return f, err
	}
	if err := cmd.Start(); err != nil {
		return f, err
	}
	remote.Close() // копия в дочернем процессе остаётся

	buf := make([]byte, 1<<20)
	for {
		n, err := out.Read(buf)
		if n > 0 {
			if f.bytes == 0 {
				f.first = time.Since(start)
			}
			f.bytes += n
			if !f.live {
				for i := 0; i < n; i += 997 {
					if buf[i] != 0 && i%4 != 3 {
						f.live = true
						break
					}
				}
			}
		}
		if err != nil {
			break
		}
	}
	f.done = time.Since(start)
	if err := cmd.Wait(); err != nil {
		return f, fmt.Errorf("gst-launch: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	if f.bytes == 0 {
		return f, fmt.Errorf("кадр пуст: %s", strings.TrimSpace(stderr.String()))
	}
	return f, nil
}

func baseline() time.Duration {
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		t := time.Now()
		exec.Command("gst-launch-1.0", "-q", "videotestsrc", "num-buffers=1", "!", "fakesink").Run()
		if d := time.Since(t); d < best {
			best = d
		}
	}
	return best
}

func report(frames []frame) {
	for _, f := range frames {
		fmt.Printf("     node %d: первый байт %s, кадр целиком %s, %.1f МБ, содержимое=%v\n",
			f.stream, ms(f.first), ms(f.done), float64(f.bytes)/1e6, f.live)
	}
}

func summary(name string, ds []time.Duration, base time.Duration) {
	if len(ds) == 0 {
		return
	}
	s := append([]time.Duration(nil), ds...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	med := s[len(s)/2]
	fmt.Printf("%s: min=%s медиана=%s max=%s | медиана минус baseline=%s\n",
		name, ms(s[0]), ms(med), ms(s[len(s)-1]), ms(med-base))
}

func tokenPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "screen-editor", "screencast.token")
}

func saveToken(path, token string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
		os.WriteFile(path, []byte(token), 0o600)
	}
}

func ms(d time.Duration) string { return fmt.Sprintf("%.0fms", float64(d.Microseconds())/1000) }

func fatal(f string, a ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	os.Exit(1)
}
