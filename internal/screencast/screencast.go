// Package screencast получает кадры экрана через xdg-desktop-portal ScreenCast
// и PipeWire. D-Bus часть делает Go, кадр читает gst-launch-1.0 (pipewiresrc)
// из выданного порталом fd. Это временное решение для spike: для релиза
// gst-launch нужно заменить клиентом libpipewire.
package screencast

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

// AppID — идентификатор приложения для portal. Ему должен соответствовать
// файл AppID+".desktop" (см. packaging/): без зарегистрированного app_id
// xdg-desktop-portal не выдаёт restore_token и показывает диалог каждый раз.
const AppID = "io.github.jexme.ScreenEditor"

// Portal — подключение к session bus для вызовов ScreenCast.
type Portal struct {
	conn   *dbus.Conn
	sender string
	n      atomic.Int64

	mu      sync.Mutex
	waiters map[dbus.ObjectPath]chan *dbus.Signal

	// Registered сообщает, принял ли portal регистрацию app_id.
	// Если нет (старый portal), restore_token работать не будет.
	Registered bool
}

type stream struct {
	ID    uint32
	Props map[string]dbus.Variant
}

// Session — запущенная ScreenCast-сессия. Пока она жива, новый кадр берётся быстро.
type Session struct {
	p       *Portal
	handle  string
	streams []stream
}

// Frame — один кадр одного монитора в RGBA.
type Frame struct {
	NodeID uint32
	X, Y   int // позиция монитора в логических координатах, если портал её отдал
	Img    *image.RGBA
}

// New подключается к session bus и регистрирует AppID в portal.
func New() (*Portal, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	// Регистрация должна идти раньше любых других вызовов portal на этом соединении.
	regErr := conn.Object(portalDest, portalPath).
		Call("org.freedesktop.host.portal.Registry.Register", 0, AppID, map[string]dbus.Variant{}).Err
	err = conn.AddMatchSignal(dbus.WithMatchInterface(reqIface), dbus.WithMatchMember("Response"))
	if err != nil {
		return nil, err
	}
	ch := make(chan *dbus.Signal, 32)
	conn.Signal(ch)
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	p := &Portal{
		conn:       conn,
		sender:     sender,
		waiters:    make(map[dbus.ObjectPath]chan *dbus.Signal),
		Registered: regErr == nil,
	}
	go p.dispatch(ch)
	return p, nil
}

// dispatch раздаёт сигналы Response по ожидающим запросам. Это позволяет
// запускать несколько сессий параллельно.
func (p *Portal) dispatch(ch <-chan *dbus.Signal) {
	for sig := range ch {
		if sig.Name != reqIface+".Response" {
			continue
		}
		p.mu.Lock()
		w := p.waiters[sig.Path]
		p.mu.Unlock()
		if w != nil {
			w <- sig
		}
	}
}

// Close закрывает соединение с D-Bus.
func (p *Portal) Close() error { return p.conn.Close() }

// TokenPath возвращает путь к файлу с restore_token.
func TokenPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "screen-editor", "screencast.token")
}

// LoadTokens читает сохранённые restore_token, по одному на монитор.
func LoadTokens() []string {
	b, err := os.ReadFile(TokenPath())
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// SaveTokens сохраняет restore_token, чтобы следующий запуск обошёлся без диалога.
func SaveTokens(tokens []string) error {
	path := TokenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(tokens, "\n")+"\n"), 0o600)
}

// request вызывает метод ScreenCast, возвращающий Request, и ждёт его Response.
func (p *Portal) request(method string, timeout time.Duration, lead []any, extra map[string]dbus.Variant) (map[string]dbus.Variant, error) {
	token := fmt.Sprintf("se%d_%d", os.Getpid(), p.n.Add(1))
	opts := map[string]dbus.Variant{"handle_token": dbus.MakeVariant(token)}
	for k, v := range extra {
		opts[k] = v
	}
	reqPath := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", p.sender, token))

	// Ожидание регистрируется до вызова, иначе Response можно пропустить.
	w := make(chan *dbus.Signal, 1)
	p.mu.Lock()
	p.waiters[reqPath] = w
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.waiters, reqPath)
		p.mu.Unlock()
	}()

	call := p.conn.Object(portalDest, portalPath).Call(scIface+"."+method, 0, append(lead, opts)...)
	if call.Err != nil {
		return nil, fmt.Errorf("%s: %w", method, call.Err)
	}
	select {
	case sig := <-w:
		code, _ := sig.Body[0].(uint32)
		res, _ := sig.Body[1].(map[string]dbus.Variant)
		if code != 0 {
			return nil, fmt.Errorf("%s: портал вернул код %d (отмена или отказ)", method, code)
		}
		return res, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("%s: таймаут %v", method, timeout)
	}
}

// NewSession запускает сессию. С непустым restore диалог выбора экрана не
// показывается. Новый restore_token (если портал его отдал) возвращается вторым значением.
func (p *Portal) NewSession(restore string) (*Session, string, error) {
	res, err := p.request("CreateSession", 10*time.Second, nil, map[string]dbus.Variant{
		"session_handle_token": dbus.MakeVariant(fmt.Sprintf("sess%d_%d", os.Getpid(), p.n.Add(1))),
	})
	if err != nil {
		return nil, "", err
	}
	handle, _ := res["session_handle"].Value().(string)
	if handle == "" {
		return nil, "", fmt.Errorf("в ответе нет session_handle")
	}
	sh := dbus.ObjectPath(handle)

	sel := map[string]dbus.Variant{
		"types":        dbus.MakeVariant(uint32(1)), // monitor
		"multiple":     dbus.MakeVariant(true),
		"cursor_mode":  dbus.MakeVariant(uint32(1)), // курсор скрыт
		"persist_mode": dbus.MakeVariant(uint32(2)), // помнить выбор
	}
	if restore != "" {
		sel["restore_token"] = dbus.MakeVariant(restore)
	}
	if _, err = p.request("SelectSources", 10*time.Second, []any{sh}, sel); err != nil {
		return nil, "", err
	}

	// Диалог выбора может ждать ответа пользователя, поэтому таймаут большой.
	startedAt := time.Now()
	res, err = p.request("Start", 120*time.Second, []any{sh, ""}, nil)
	if err != nil {
		return nil, "", err
	}
	if os.Getenv("SCREENCAST_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] Start за %s, restore передан=%v, результат: %v\n",
			time.Since(startedAt).Round(time.Millisecond), restore != "", res)
	}
	s := &Session{p: p, handle: handle}
	if err := res["streams"].Store(&s.streams); err != nil {
		return nil, "", fmt.Errorf("разбор streams: %w", err)
	}
	if len(s.streams) == 0 {
		return nil, "", fmt.Errorf("портал не вернул потоков")
	}
	newToken, _ := res["restore_token"].Value().(string)
	return s, newToken, nil
}

// Close закрывает сессию.
func (s *Session) Close() {
	s.p.conn.Object(portalDest, dbus.ObjectPath(s.handle)).Call(sessIface+".Close", 0)
}

// Streams возвращает число потоков (мониторов) в сессии.
func (s *Session) Streams() int { return len(s.streams) }

func (s *Session) openRemote() (*os.File, error) {
	var fd dbus.UnixFD
	call := s.p.conn.Object(portalDest, portalPath).
		Call(scIface+".OpenPipeWireRemote", 0, dbus.ObjectPath(s.handle), map[string]dbus.Variant{})
	if err := call.Store(&fd); err != nil {
		return nil, fmt.Errorf("OpenPipeWireRemote: %w", err)
	}
	return os.NewFile(uintptr(fd), "pipewire-remote"), nil
}

// Grab получает по одному кадру со всех мониторов параллельно.
func (s *Session) Grab() ([]Frame, error) {
	files := make([]*os.File, len(s.streams))
	for i := range s.streams {
		f, err := s.openRemote()
		if err != nil {
			return nil, err
		}
		files[i] = f
	}

	frames := make([]Frame, len(s.streams))
	errs := make([]error, len(s.streams))
	var wg sync.WaitGroup
	for i, st := range s.streams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			frames[i], errs[i] = s.grabRetry(st, files[i])
		}()
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			return nil, e
		}
	}
	return frames, nil
}

// grabRetry делает несколько попыток получить кадр: сразу после Start узел
// PipeWire иногда ещё не отдаёт буферы, и единственная попытка зависает.
func (s *Session) grabRetry(st stream, remote *os.File) (Frame, error) {
	var (
		f   Frame
		err error
	)
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			if remote, err = s.openRemote(); err != nil {
				return f, err
			}
		}
		if f, err = grab(st, remote, 3*time.Second); err == nil {
			return f, nil
		}
	}
	return f, err
}

// pair достаёт (int32, int32) из свойства потока вида "(ii)".
func pair(props map[string]dbus.Variant, key string) (int, int, bool) {
	v, ok := props[key]
	if !ok {
		return 0, 0, false
	}
	parts, ok := v.Value().([]any)
	if !ok || len(parts) != 2 {
		return 0, 0, false
	}
	a, ok1 := parts[0].(int32)
	b, ok2 := parts[1].(int32)
	return int(a), int(b), ok1 && ok2
}

// grab читает один кадр BGRx из PipeWire-узла через gst-launch-1.0.
func grab(st stream, remote *os.File, timeout time.Duration) (Frame, error) {
	f := Frame{NodeID: st.ID}
	f.X, f.Y, _ = pair(st.Props, "position")
	w, h, sized := pair(st.Props, "size")
	if !sized {
		remote.Close()
		return f, fmt.Errorf("портал не сообщил размер потока %d", st.ID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gst-launch-1.0", "-q",
		"pipewiresrc", "fd=3", "path="+strconv.Itoa(int(st.ID)), "num-buffers=1",
		"!", "videoconvert", "!", "video/x-raw,format=BGRx", "!", "fdsink", "fd=1")
	cmd.ExtraFiles = []*os.File{remote}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		remote.Close()
		return f, err
	}
	if err := cmd.Start(); err != nil {
		remote.Close()
		return f, err
	}
	remote.Close() // копия в дочернем процессе остаётся

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if _, err := io.ReadFull(out, img.Pix); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return f, fmt.Errorf("кадр %dx%d: %w: %s", w, h, err, strings.TrimSpace(stderr.String()))
	}
	// Дочитываем остаток, чтобы Wait не завис на полном канале.
	io.Copy(io.Discard, out)
	if err := cmd.Wait(); err != nil {
		return f, fmt.Errorf("gst-launch: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	// BGRx -> RGBA: меняем местами B и R, альфа непрозрачная.
	px := img.Pix
	for i := 0; i < len(px); i += 4 {
		px[i], px[i+2] = px[i+2], px[i]
		px[i+3] = 255
	}
	f.Img = img
	return f, nil
}
