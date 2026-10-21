// Package screencast gets screen frames through the xdg-desktop-portal ScreenCast
// and PipeWire. Go handles the D-Bus part, and the libpipewire client
// (internal/pwcapture) reads the frame from the fd handed out by the portal.
package screencast

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/pwcapture"
)

const (
	portalDest = "org.freedesktop.portal.Desktop"
	portalPath = "/org/freedesktop/portal/desktop"
	scIface    = "org.freedesktop.portal.ScreenCast"
	reqIface   = "org.freedesktop.portal.Request"
	sessIface  = "org.freedesktop.portal.Session"
)

// AppID is the application identifier for the portal. The file AppID+".desktop"
// (see packaging/) must match it: without a registered app_id
// xdg-desktop-portal does not issue a restore_token and shows the dialog every time.
const AppID = "io.github.jexme.Scrawl"

// Portal is a connection to the session bus for ScreenCast calls.
type Portal struct {
	conn   *dbus.Conn
	sender string
	n      atomic.Int64

	mu      sync.Mutex
	waiters map[dbus.ObjectPath]chan *dbus.Signal

	// remoteMu serializes OpenPipeWireRemote calls.
	remoteMu sync.Mutex

	// Registered reports whether the portal accepted the app_id registration.
	// If not (an old portal), restore_token will not work.
	Registered bool

	// OldPortal means a portal older than 1.21: it has no Registry interface at all. This is
	// not a failure but a version quirk: such a portal takes the app_id from the process's
	// systemd scope.
	OldPortal bool

	// IncludeCursor asks the portal to draw the mouse cursor right into the frame. Set
	// before OpenAll: the cursor mode is chosen when the session is created.
	IncludeCursor bool
}

// ScreenCast portal cursor modes (cursor_mode).
const (
	cursorHidden   uint32 = 1 // the cursor is not in the frame
	cursorEmbedded uint32 = 2 // the cursor is drawn into the frame
)

// cursorMode returns the cursor_mode value for SelectSources.
func cursorMode(include bool) uint32 {
	if include {
		return cursorEmbedded
	}
	return cursorHidden
}

type stream struct {
	ID    uint32
	Props map[string]dbus.Variant
}

// Session is a running ScreenCast session. While it is alive, a new frame is fetched quickly.
type Session struct {
	p       *Portal
	handle  string
	streams []stream
}

// Frame is a single frame of one monitor in RGBA.
type Frame struct {
	NodeID uint32
	X, Y   int // monitor position in logical coordinates, if the portal provided it
	Img    *image.RGBA
}

// New connects to the session bus and registers AppID with the portal.
func New() (*Portal, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	// Registration must come before any other portal calls on this connection. The timeout is
	// short: if the portal hangs, it is better to work without registration than to hang.
	regCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	regErr := conn.Object(portalDest, portalPath).
		CallWithContext(regCtx, "org.freedesktop.host.portal.Registry.Register", 0, AppID, map[string]dbus.Variant{}).Err
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
		OldPortal:  isMissingRegistry(regErr),
	}
	go p.dispatch(ch)
	return p, nil
}

// isMissingRegistry reports that the portal answered "no such method", meaning it is
// older than 1.21 and does not know Registry, rather than hung.
func isMissingRegistry(err error) bool {
	var derr dbus.Error
	if !errors.As(err, &derr) {
		return false
	}
	switch derr.Name {
	case "org.freedesktop.DBus.Error.UnknownMethod",
		"org.freedesktop.DBus.Error.UnknownInterface",
		"org.freedesktop.DBus.Error.UnknownObject":
		return true
	}
	return false
}

// dispatch distributes Response signals to the pending requests. This allows
// running several sessions in parallel.
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

// Close closes the D-Bus connection.
func (p *Portal) Close() error { return p.conn.Close() }

// TokenPath returns the path to the restore_token file.
func TokenPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "scrawl", "screencast.token")
}

// LoadTokens reads the saved restore_tokens, one per monitor.
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

// SaveTokens saves the restore_tokens so the next launch needs no dialog.
func SaveTokens(tokens []string) error {
	path := TokenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(tokens, "\n")+"\n"), 0o600)
}

// request calls a ScreenCast method that returns a Request and waits for its Response.
func (p *Portal) request(method string, timeout time.Duration, lead []any, extra map[string]dbus.Variant) (map[string]dbus.Variant, error) {
	if !strings.Contains(method, ".") {
		method = scIface + "." + method // a short name is treated as a ScreenCast method
	}
	token := fmt.Sprintf("se%d_%d", os.Getpid(), p.n.Add(1))
	opts := map[string]dbus.Variant{"handle_token": dbus.MakeVariant(token)}
	for k, v := range extra {
		opts[k] = v
	}
	reqPath := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", p.sender, token))

	// The wait is registered before the call, otherwise the Response could be missed.
	w := make(chan *dbus.Signal, 1)
	p.mu.Lock()
	p.waiters[reqPath] = w
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.waiters, reqPath)
		p.mu.Unlock()
	}()

	call := p.conn.Object(portalDest, portalPath).Call(method, 0, append(lead, opts)...)
	if call.Err != nil {
		return nil, fmt.Errorf("%s: %w", method, call.Err)
	}
	select {
	case sig := <-w:
		code, _ := sig.Body[0].(uint32)
		res, _ := sig.Body[1].(map[string]dbus.Variant)
		if code != 0 {
			return nil, fmt.Errorf(i18n.T("%s: the portal returned code %d (cancelled or refused)"), method, code)
		}
		return res, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf(i18n.T("%s: timeout %v"), method, timeout)
	}
}

// NewSession starts a session. With a non-empty restore the screen selection dialog is not
// shown. The new restore_token (if the portal returned one) is returned as the second value.
func (p *Portal) NewSession(restore string) (*Session, string, error) {
	res, err := p.request("CreateSession", 10*time.Second, nil, map[string]dbus.Variant{
		"session_handle_token": dbus.MakeVariant(fmt.Sprintf("sess%d_%d", os.Getpid(), p.n.Add(1))),
	})
	if err != nil {
		return nil, "", err
	}
	handle, _ := res["session_handle"].Value().(string)
	if handle == "" {
		return nil, "", errors.New(i18n.T("no session_handle in the response"))
	}
	sh := dbus.ObjectPath(handle)

	sel := map[string]dbus.Variant{
		"types":        dbus.MakeVariant(uint32(1)), // monitor
		"multiple":     dbus.MakeVariant(true),
		"cursor_mode":  dbus.MakeVariant(cursorMode(p.IncludeCursor)),
		"persist_mode": dbus.MakeVariant(uint32(2)), // remember the choice
	}
	if restore != "" {
		sel["restore_token"] = dbus.MakeVariant(restore)
	}
	if _, err = p.request("SelectSources", 10*time.Second, []any{sh}, sel); err != nil {
		return nil, "", err
	}

	// The selection dialog waits for the user, so without a token the timeout is long. With a
	// token there should be no dialog: if the portal is silent, it is better to report an
	// error.
	startTimeout := 120 * time.Second
	if restore != "" {
		startTimeout = 15 * time.Second
	}
	startedAt := time.Now()
	res, err = p.request("Start", startTimeout, []any{sh, ""}, nil)
	if err != nil {
		return nil, "", err
	}
	if os.Getenv("SCREENCAST_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] Start took %s, restore passed=%v, result: %v\n",
			time.Since(startedAt).Round(time.Millisecond), restore != "", res)
	}
	s := &Session{p: p, handle: handle}
	if err := res["streams"].Store(&s.streams); err != nil {
		return nil, "", fmt.Errorf(i18n.T("parsing streams: %w"), err)
	}
	if len(s.streams) == 0 {
		return nil, "", errors.New(i18n.T("the portal returned no streams"))
	}
	newToken, _ := res["restore_token"].Value().(string)
	return s, newToken, nil
}

// Close closes the session.
func (s *Session) Close() {
	s.p.conn.Object(portalDest, dbus.ObjectPath(s.handle)).Call(sessIface+".Close", 0)
}

// Streams returns the number of streams (monitors) in the session.
func (s *Session) Streams() int { return len(s.streams) }

func (s *Session) openRemote() (*os.File, error) {
	s.p.remoteMu.Lock()
	defer s.p.remoteMu.Unlock()
	var fd dbus.UnixFD
	call := s.p.conn.Object(portalDest, portalPath).
		Call(scIface+".OpenPipeWireRemote", 0, dbus.ObjectPath(s.handle), map[string]dbus.Variant{})
	if err := call.Store(&fd); err != nil {
		return nil, fmt.Errorf("OpenPipeWireRemote: %w", err)
	}
	return os.NewFile(uintptr(fd), "pipewire-remote"), nil
}

// Grab gets one frame from every monitor in parallel. A frame that does not
// arrive by deadline counts as a failure: Mutter delivers frames only when the
// monitor is redrawn, and on a static screen the stream may stay silent.
func (s *Session) Grab(deadline time.Duration) ([]Frame, error) {
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
			frames[i], errs[i] = grab(st, files[i], deadline)
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

// pair extracts (int32, int32) from a stream property of the form "(ii)".
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

// grab reads one frame from a PipeWire node. remote is closed: the PipeWire library
// gets its own copy of the fd and closes it itself.
func grab(st stream, remote *os.File, timeout time.Duration) (Frame, error) {
	f := Frame{NodeID: st.ID}
	f.X, f.Y, _ = pair(st.Props, "position")

	fd, err := dupFD(remote)
	if err != nil {
		return f, fmt.Errorf("dup fd: %w", err)
	}
	img, err := pwcapture.Capture(fd, st.ID, timeout)
	if err != nil {
		return f, fmt.Errorf(i18n.T("node %d: %w"), st.ID, err)
	}
	f.Img = img
	return f, nil
}
