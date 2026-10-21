// Package tray shows an icon in the notification area through StatusNotifierItem (the KDE
// and AppIndicator protocol): a left click takes a screenshot, the menu offers "Take
// screenshot", "Settings" and "Quit". On GNOME the icon is visible only with the
// AppIndicator extension (enabled by default in Ubuntu).
package tray

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"

	"github.com/jexme/scrawl/internal/i18n"
)

const (
	itemIface   = "org.kde.StatusNotifierItem"
	itemPath    = "/StatusNotifierItem"
	menuIface   = "com.canonical.dbusmenu"
	menuPath    = "/MenuBar"
	watcherName = "org.kde.StatusNotifierWatcher"
	watcherPath = "/StatusNotifierWatcher"
)

// item implements the methods of org.kde.StatusNotifierItem.
type itemObj struct{ h Handlers }

// Activate is called by a left click on the icon.
func (i *itemObj) Activate(x, y int32) *dbus.Error {
	if i.h.Shot != nil {
		go i.h.Shot()
	}
	return nil
}

func (i *itemObj) SecondaryActivate(x, y int32) *dbus.Error { return nil }
func (i *itemObj) ContextMenu(x, y int32) *dbus.Error       { return nil }
func (i *itemObj) Scroll(delta int32, orientation string) *dbus.Error {
	return nil
}

type toolTip struct {
	IconName string
	Icons    []pixmap
	Title    string
	Text     string
}

// Item is an icon exported on the bus.
type Item struct {
	conn *dbus.Conn
	name string
}

// Name returns the service name of the icon on the bus.
func (it *Item) Name() string { return it.name }

// Start exports the icon and the menu on the connection and registers them with the
// StatusNotifierWatcher. If there is no watcher (no AppIndicator extension), the icon
// stays exported, and the error says that it is not visible yet.
func Start(conn *dbus.Conn, h Handlers) (*Item, error) {
	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf(i18n.T("the name %s is already taken"), name)
	}

	icons := []pixmap{toPixmap(drawIcon(22)), toPixmap(drawIcon(32)), toPixmap(drawIcon(48))}
	_, err = prop.Export(conn, itemPath, map[string]map[string]*prop.Prop{itemIface: {
		"Category":            {Value: "ApplicationStatus"},
		"Id":                  {Value: "scrawl"},
		"Title":               {Value: "Scrawl"},
		"Status":              {Value: "Active"},
		"WindowId":            {Value: uint32(0)},
		"IconName":            {Value: ""},
		"IconPixmap":          {Value: icons},
		"OverlayIconName":     {Value: ""},
		"OverlayIconPixmap":   {Value: []pixmap{}},
		"AttentionIconName":   {Value: ""},
		"AttentionIconPixmap": {Value: []pixmap{}},
		"AttentionMovieName":  {Value: ""},
		"ToolTip":             {Value: toolTip{Icons: []pixmap{}, Title: "Scrawl", Text: i18n.T("Left click: take a screenshot")}},
		"ItemIsMenu":          {Value: false},
		"Menu":                {Value: dbus.ObjectPath(menuPath)},
	}})
	if err != nil {
		return nil, err
	}
	if err := conn.Export(&itemObj{h: h}, itemPath, itemIface); err != nil {
		return nil, err
	}
	_, err = prop.Export(conn, menuPath, map[string]map[string]*prop.Prop{menuIface: {
		"Version":       {Value: uint32(3)},
		"TextDirection": {Value: "ltr"},
		"Status":        {Value: "normal"},
		"IconThemePath": {Value: []string{}},
	}})
	if err != nil {
		return nil, err
	}
	if err := conn.Export(&menu{h: h}, menuPath, menuIface); err != nil {
		return nil, err
	}

	it := &Item{conn: conn, name: name}
	return it, it.register()
}

// register tells the watcher about the icon. A repeated call is safe.
func (it *Item) register() error {
	call := it.conn.Object(watcherName, watcherPath).Call(watcherName+".RegisterStatusNotifierItem", 0, it.name)
	if call.Err != nil {
		return fmt.Errorf(i18n.T("the icon is not registered (is the AppIndicator extension missing?): %w"), call.Err)
	}
	return nil
}
