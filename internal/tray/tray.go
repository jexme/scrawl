// Package tray показывает значок в области уведомлений через StatusNotifierItem
// (протокол KDE и AppIndicator): левый клик делает снимок, меню даёт «Снимок»,
// «Настройки» и «Выход». На GNOME значок виден только с расширением AppIndicator
// (в Ubuntu оно включено по умолчанию).
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

// item реализует методы org.kde.StatusNotifierItem.
type itemObj struct{ h Handlers }

// Activate вызывается левым кликом по значку.
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

// Item — значок, выставленный на шине.
type Item struct {
	conn *dbus.Conn
	name string
}

// Name возвращает имя службы значка на шине.
func (it *Item) Name() string { return it.name }

// Start выставляет значок и меню на соединении и регистрирует их у наблюдателя
// StatusNotifierWatcher. Если наблюдателя нет (нет расширения AppIndicator),
// значок остаётся выставленным, а ошибка говорит о том, что его пока не видно.
func Start(conn *dbus.Conn, h Handlers) (*Item, error) {
	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf(i18n.T("имя %s уже занято"), name)
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
		"ToolTip":             {Value: toolTip{Icons: []pixmap{}, Title: "Scrawl", Text: i18n.T("Левый клик: снимок экрана")}},
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

// register сообщает наблюдателю о значке. Повторный вызов безопасен.
func (it *Item) register() error {
	call := it.conn.Object(watcherName, watcherPath).Call(watcherName+".RegisterStatusNotifierItem", 0, it.name)
	if call.Err != nil {
		return fmt.Errorf(i18n.T("значок не зарегистрирован (нет расширения AppIndicator?): %w"), call.Err)
	}
	return nil
}
