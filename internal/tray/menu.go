package tray

import (
	"github.com/godbus/dbus/v5"

	"github.com/jexme/scrawl/internal/i18n"
)

// Номера пунктов меню; 0 — корень.
const (
	idShot     int32 = 1
	idSettings int32 = 2
	idSep      int32 = 3
	idQuit     int32 = 4
)

// Handlers — действия пунктов меню и левого клика по значку.
type Handlers struct {
	Shot     func() // сделать снимок
	Settings func() // открыть настройки
	Quit     func() // закрыть значок (сам Run завершится после этого)
}

// layout — узел меню в формате com.canonical.dbusmenu: (ia{sv}av).
type layout struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

type item struct {
	id    int32
	label string // пусто у разделителя
}

func items() []item {
	return []item{
		{idShot, i18n.T("Take screenshot")},
		{idSettings, i18n.T("Settings")},
		{idSep, ""},
		{idQuit, i18n.T("Quit")},
	}
}

func itemProps(it item) map[string]dbus.Variant {
	if it.label == "" {
		return map[string]dbus.Variant{"type": dbus.MakeVariant("separator"), "visible": dbus.MakeVariant(true)}
	}
	return map[string]dbus.Variant{
		"label":   dbus.MakeVariant(it.label),
		"enabled": dbus.MakeVariant(true),
		"visible": dbus.MakeVariant(true),
	}
}

// menu реализует com.canonical.dbusmenu.
type menu struct {
	h        Handlers
	revision uint32
}

// GetLayout возвращает меню целиком: пунктов мало, поэтому глубина не важна.
func (m *menu) GetLayout(parent int32, depth int32, props []string) (uint32, layout, *dbus.Error) {
	root := layout{ID: 0, Props: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}}
	for _, it := range items() {
		root.Children = append(root.Children, dbus.MakeVariant(layout{ID: it.id, Props: itemProps(it), Children: []dbus.Variant{}}))
	}
	if parent == 0 {
		return m.revision, root, nil
	}
	for _, it := range items() {
		if it.id == parent {
			return m.revision, layout{ID: it.id, Props: itemProps(it), Children: []dbus.Variant{}}, nil
		}
	}
	return 0, layout{}, dbus.MakeFailedError(errNoItem(parent))
}

type groupItem struct {
	ID    int32
	Props map[string]dbus.Variant
}

func (m *menu) GetGroupProperties(ids []int32, props []string) ([]groupItem, *dbus.Error) {
	var out []groupItem
	for _, it := range items() {
		for _, id := range ids {
			if id == it.id {
				out = append(out, groupItem{it.id, itemProps(it)})
			}
		}
	}
	return out, nil
}

func (m *menu) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	for _, it := range items() {
		if it.id == id {
			if v, ok := itemProps(it)[name]; ok {
				return v, nil
			}
		}
	}
	return dbus.Variant{}, dbus.MakeFailedError(errNoItem(id))
}

// Event принимает нажатия: «clicked» запускает действие пункта.
func (m *menu) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventID == "clicked" {
		m.activate(id)
	}
	return nil
}

type menuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

func (m *menu) EventGroup(events []menuEvent) ([]int32, *dbus.Error) {
	for _, e := range events {
		if e.EventID == "clicked" {
			m.activate(e.ID)
		}
	}
	return []int32{}, nil
}

func (m *menu) AboutToShow(id int32) (bool, *dbus.Error) { return false, nil }

func (m *menu) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
}

func (m *menu) activate(id int32) {
	var f func()
	switch id {
	case idShot:
		f = m.h.Shot
	case idSettings:
		f = m.h.Settings
	case idQuit:
		f = m.h.Quit
	}
	if f != nil {
		go f() // вызов пришёл из обработчика D-Bus: не задерживаем ответ
	}
}

type errNoItem int32

func (e errNoItem) Error() string { return i18n.T("no such menu item") }
