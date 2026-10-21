// capturespike измеряет задержку захвата экрана на GNOME/Wayland.
//
// Пробует два пути:
//  1. xdg-desktop-portal Screenshot (non-interactive), основной путь.
//  2. org.gnome.Shell.Screenshot напрямую, ожидаемо закрыт для сторонних приложений.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest   = "org.freedesktop.portal.Desktop"
	portalPath   = "/org/freedesktop/portal/desktop"
	portalShot   = "org.freedesktop.portal.Screenshot.Screenshot"
	requestIface = "org.freedesktop.portal.Request"
	shellDest    = "org.gnome.Shell.Screenshot"
	shellPath    = "/org/gnome/Shell/Screenshot"
)

type timing struct {
	portal time.Duration // вызов -> сигнал Response
	read   time.Duration // открытие файла
	decode time.Duration // png.Decode
	w, h   int
}

func main() {
	runs := flag.Int("n", 5, "number of portal measurements")
	keep := flag.Bool("keep", false, "do not delete the files created by the portal")
	flag.Parse()

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fatal("session bus: %v", err)
	}
	defer conn.Close()

	fmt.Println("== portal Screenshot ==")
	var all []timing
	for i := 0; i < *runs; i++ {
		t, err := viaPortal(conn, *keep)
		if err != nil {
			fatal("portal: %v", err)
		}
		all = append(all, t)
		tag := "cold"
		if i > 0 {
			tag = "warm"
		}
		fmt.Printf("#%d (%s) %dx%d  portal=%v open=%v decode=%v  total=%v\n",
			i+1, tag, t.w, t.h, ms(t.portal), ms(t.read), ms(t.decode), ms(t.portal+t.read+t.decode))
	}
	summarize(all)

	fmt.Println("\n== org.gnome.Shell.Screenshot (direct) ==")
	if err := viaShell(conn); err != nil {
		fmt.Printf("unavailable: %v\n", err)
	} else {
		fmt.Println("available (a rare case, can be used as a fast path)")
	}
}

func viaPortal(conn *dbus.Conn, keep bool) (timing, error) {
	var t timing

	token := fmt.Sprintf("se%d", time.Now().UnixNano())
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	reqPath := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", sender, token))

	// Подписка должна быть до вызова, иначе Response можно пропустить.
	matchOpts := []dbus.MatchOption{
		dbus.WithMatchObjectPath(reqPath),
		dbus.WithMatchInterface(requestIface),
		dbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignal(matchOpts...); err != nil {
		return t, err
	}
	defer conn.RemoveMatchSignal(matchOpts...)

	ch := make(chan *dbus.Signal, 8)
	conn.Signal(ch)
	defer conn.RemoveSignal(ch)

	start := time.Now()
	obj := conn.Object(portalDest, portalPath)
	call := obj.Call(portalShot, 0, "", map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"interactive":  dbus.MakeVariant(false),
	})
	if call.Err != nil {
		return t, call.Err
	}

	var uri string
	timeout := time.After(30 * time.Second)
wait:
	for {
		select {
		case sig := <-ch:
			if sig.Path != reqPath || sig.Name != requestIface+".Response" {
				continue
			}
			code, _ := sig.Body[0].(uint32)
			res, _ := sig.Body[1].(map[string]dbus.Variant)
			if code != 0 {
				return t, fmt.Errorf("the portal refused the request (code %d): allow screenshots in the GNOME dialog and try again", code)
			}
			u, ok := res["uri"].Value().(string)
			if !ok {
				return t, fmt.Errorf("no uri in the response")
			}
			uri = u
			break wait
		case <-timeout:
			return t, fmt.Errorf("timeout waiting for the Response")
		}
	}
	t.portal = time.Since(start)

	path, err := uriToPath(uri)
	if err != nil {
		return t, err
	}
	if !keep {
		defer os.Remove(path)
	}

	s := time.Now()
	f, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer f.Close()
	t.read = time.Since(s)

	s = time.Now()
	img, err := png.Decode(f)
	if err != nil {
		return t, err
	}
	t.decode = time.Since(s)
	b := img.Bounds()
	t.w, t.h = b.Dx(), b.Dy()
	return t, nil
}

func viaShell(conn *dbus.Conn) error {
	dir, err := os.MkdirTemp("", "capturespike")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	var ok bool
	var name string
	call := conn.Object(shellDest, dbus.ObjectPath(shellPath)).
		Call(shellDest+".Screenshot", 0, false, false, filepath.Join(dir, "s.png"))
	if err := call.Store(&ok, &name); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Shell returned false")
	}
	return nil
}

func uriToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("unexpected uri scheme: %q", uri)
	}
	return u.Path, nil
}

func summarize(all []timing) {
	if len(all) == 0 {
		return
	}
	tot := make([]time.Duration, len(all))
	for i, t := range all {
		tot[i] = t.portal + t.read + t.decode
	}
	sort.Slice(tot, func(i, j int) bool { return tot[i] < tot[j] })
	fmt.Printf("total: min=%v median=%v max=%v\n", ms(tot[0]), ms(tot[len(tot)/2]), ms(tot[len(tot)-1]))
}

func ms(d time.Duration) string { return fmt.Sprintf("%.0fms", float64(d.Microseconds())/1000) }

func fatal(f string, a ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	os.Exit(1)
}
