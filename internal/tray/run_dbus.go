//go:build !windows

package tray

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

// Run показывает значок, пока не вызван h.Quit или не закрыт канал stop (nil: без него). Если наблюдатель появится позже
// (расширение включили после запуска) или перезапустится, значок регистрируется заново.
func Run(h Handlers, stop <-chan struct{}) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	quit := make(chan struct{})
	userQuit := h.Quit
	h.Quit = func() {
		if userQuit != nil {
			userQuit()
		}
		select {
		case <-quit:
		default:
			close(quit)
		}
	}

	it, err := Start(conn, h)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if it == nil {
			return err
		}
	}

	if err := conn.AddMatchSignal(dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"), dbus.WithMatchArg(0, watcherName)); err == nil {
		sig := make(chan *dbus.Signal, 8)
		conn.Signal(sig)
		for {
			select {
			case <-quit:
				return nil
			case <-stop:
				return nil
			case s := <-sig:
				if s == nil {
					return nil // соединение закрыто
				}
				if s.Name == "org.freedesktop.DBus.NameOwnerChanged" && len(s.Body) == 3 {
					if owner, _ := s.Body[2].(string); owner != "" {
						it.register()
					}
				}
			}
		}
	}
	select {
	case <-quit:
	case <-stop:
	}
	return nil
}
