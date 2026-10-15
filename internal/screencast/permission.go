package screencast

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	permStoreDest = "org.freedesktop.impl.portal.PermissionStore"
	permStorePath = "/org/freedesktop/impl/portal/PermissionStore"
)

// GrantScreenshotPermission выдаёт приложению разрешение на скриншоты в
// хранилище разрешений xdg-desktop-portal. Оно нужно запасному пути захвата:
// после регистрации app_id portal перестаёт считать приложение «неизвестным» и
// без записи отказывает. Вызывать только по явной команде пользователя.
func GrantScreenshotPermission() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	call := conn.Object(permStoreDest, permStorePath).Call(
		permStoreDest+".SetPermission", 0,
		"screenshot", true, "screenshot", AppID, []string{"yes"})
	if call.Err != nil {
		return fmt.Errorf("SetPermission: %w", call.Err)
	}
	return nil
}
