package screencast

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	permStoreDest = "org.freedesktop.impl.portal.PermissionStore"
	permStorePath = "/org/freedesktop/impl/portal/PermissionStore"
)

// GrantScreenshotPermission grants the application permission to take screenshots in the
// xdg-desktop-portal permission store. The fallback capture path needs it:
// after app_id is registered, the portal stops treating the application as "unknown" and
// refuses without a record. Call only on an explicit user command.
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
