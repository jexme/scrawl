package tray

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestMenuLayout(t *testing.T) {
	m := &menu{}
	rev, root, err := m.GetLayout(0, -1, nil)
	if err != nil || rev != 0 {
		t.Fatalf("GetLayout: %v %v", rev, err)
	}
	if len(root.Children) != 4 {
		t.Fatalf("в меню 4 пункта (снимок, настройки, разделитель, выход): %d", len(root.Children))
	}
	var labels []string
	for _, c := range root.Children {
		child := c.Value().(layout)
		if child.Props["type"].Value() == "separator" {
			labels = append(labels, "-")
			continue
		}
		labels = append(labels, child.Props["label"].Value().(string))
	}
	if labels[0] != "Take screenshot" || labels[1] != "Settings" || labels[2] != "-" || labels[3] != "Quit" {
		t.Errorf("пункты меню: %v", labels)
	}
	if _, _, err := m.GetLayout(99, 0, nil); err == nil {
		t.Error("неизвестный пункт должен давать ошибку")
	}
	props, err := m.GetGroupProperties([]int32{idShot, idQuit, 99}, nil)
	if err != nil || len(props) != 2 {
		t.Errorf("GetGroupProperties: %v %v", props, err)
	}
	if v, err := m.GetProperty(idShot, "label"); err != nil || v.Value() != "Take screenshot" {
		t.Errorf("GetProperty: %v %v", v, err)
	}
}

func TestMenuEventsRunHandlers(t *testing.T) {
	var shot, settings, quit atomic.Int32
	m := &menu{h: Handlers{
		Shot:     func() { shot.Add(1) },
		Settings: func() { settings.Add(1) },
		Quit:     func() { quit.Add(1) },
	}}
	m.Event(idShot, "clicked", dbus.MakeVariant(0), 0)
	m.Event(idSettings, "clicked", dbus.MakeVariant(0), 0)
	m.Event(idSettings, "hovered", dbus.MakeVariant(0), 0) // наведение не запускает
	m.Event(idSep, "clicked", dbus.MakeVariant(0), 0)
	m.EventGroup([]menuEvent{{ID: idQuit, EventID: "clicked"}})
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && (shot.Load() == 0 || settings.Load() == 0 || quit.Load() == 0) {
		time.Sleep(5 * time.Millisecond)
	}
	if shot.Load() != 1 || settings.Load() != 1 || quit.Load() != 1 {
		t.Errorf("снимок %d, настройки %d, выход %d", shot.Load(), settings.Load(), quit.Load())
	}
	// Левый клик по значку тоже делает снимок.
	(&itemObj{h: m.h}).Activate(0, 0)
	time.Sleep(50 * time.Millisecond)
	if shot.Load() != 2 {
		t.Errorf("левый клик: снимков %d", shot.Load())
	}
}

func TestIcon(t *testing.T) {
	for _, size := range []int{22, 32, 48} {
		img := drawIcon(size)
		if img.Bounds().Dx() != size || img.Bounds().Dy() != size {
			t.Fatalf("размер значка %v", img.Bounds())
		}
		lit := 0
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA); c.A > 0 {
					lit++
				}
			}
		}
		if lit < size*size/20 || lit > size*size/2 {
			t.Errorf("значок %d px: закрашено %d из %d точек", size, lit, size*size)
		}
		pm := toPixmap(img)
		if int(pm.W) != size || len(pm.Data) != size*size*4 {
			t.Errorf("pixmap %d px: %dx%d, %d байт", size, pm.W, pm.H, len(pm.Data))
		}
	}
	// ARGB: первый закрашенный пиксель непрозрачный белый.
	pm := toPixmap(drawIcon(22))
	for i := 0; i < len(pm.Data); i += 4 {
		if pm.Data[i] == 255 {
			if pm.Data[i+1] != 255 || pm.Data[i+2] != 255 || pm.Data[i+3] != 255 {
				t.Errorf("закрашенная точка не белая: %v", pm.Data[i:i+4])
			}
			return
		}
	}
	t.Error("в значке нет закрашенных точек")
}

// TestOnSessionBus проверяет значок по-настоящему: выставляет его на шине сеанса,
// читает свойства и меню другим соединением и ищет значок у наблюдателя. Пропускается
// без шины или без StatusNotifierWatcher.
func TestOnSessionBus(t *testing.T) {
	server, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Skipf("нет шины сеанса: %v", err)
	}
	defer server.Close()
	client, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Skipf("нет шины сеанса: %v", err)
	}
	defer client.Close()

	var owner string
	if err := client.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, watcherName).Store(&owner); err != nil {
		t.Skipf("нет StatusNotifierWatcher: %v", err)
	}

	var clicked atomic.Int32
	it, err := Start(server, Handlers{Shot: func() { clicked.Add(1) }})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	obj := client.Object(it.Name(), itemPath)

	for prop, want := range map[string]string{"Id": "scrawl", "Category": "ApplicationStatus", "Status": "Active"} {
		v, err := obj.GetProperty(itemIface + "." + prop)
		if err != nil || v.Value() != want {
			t.Errorf("%s: %v, %v", prop, v, err)
		}
	}
	if v, err := obj.GetProperty(itemIface + ".Menu"); err != nil || v.Value() != dbus.ObjectPath(menuPath) {
		t.Errorf("Menu: %v %v", v, err)
	}
	var icons []pixmap
	if v, err := obj.GetProperty(itemIface + ".IconPixmap"); err != nil || v.Store(&icons) != nil || len(icons) != 3 {
		t.Errorf("IconPixmap: %v %v", v, err)
	}

	menuObj := client.Object(it.Name(), menuPath)
	var rev uint32
	var root layout
	if err := menuObj.Call(menuIface+".GetLayout", 0, int32(0), int32(-1), []string{}).Store(&rev, &root); err != nil {
		t.Fatalf("GetLayout по шине: %v", err)
	}
	if len(root.Children) != 4 {
		t.Errorf("пунктов меню по шине: %d", len(root.Children))
	}
	if v, err := menuObj.GetProperty(menuIface + ".Version"); err != nil || v.Value() != uint32(3) {
		t.Errorf("Version: %v %v", v, err)
	}

	// Клик по значку и по пункту меню доходит до обработчика.
	if err := obj.Call(itemIface+".Activate", 0, int32(0), int32(0)).Err; err != nil {
		t.Fatal(err)
	}
	if err := menuObj.Call(menuIface+".Event", 0, idShot, "clicked", dbus.MakeVariant(int32(0)), uint32(0)).Err; err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && clicked.Load() < 2 {
		time.Sleep(10 * time.Millisecond)
	}
	if clicked.Load() != 2 {
		t.Errorf("до обработчика дошло %d из 2 нажатий", clicked.Load())
	}

	// Наблюдатель знает о значке.
	var registered []string
	v, err := client.Object(watcherName, watcherPath).GetProperty(watcherName + ".RegisteredStatusNotifierItems")
	if err != nil || v.Store(&registered) != nil {
		t.Fatalf("RegisteredStatusNotifierItems: %v", err)
	}
	found := false
	for _, r := range registered {
		found = found || contains(r, it.Name())
	}
	if !found {
		t.Errorf("значок %s не найден у наблюдателя: %v", it.Name(), registered)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestIconPicture сохраняет значок на тёмном фоне для просмотра (SCRAWL_TEST_OUT).
func TestIconPicture(t *testing.T) {
	path := os.Getenv("SCRAWL_TEST_OUT")
	if path == "" {
		t.Skip("нужна переменная SCRAWL_TEST_OUT")
	}
	const scale = 6
	out := image.NewNRGBA(image.Rect(0, 0, 3*48*scale/2+20, 48*scale+20))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.NRGBA{R: 40, G: 40, B: 44, A: 255}), image.Point{}, draw.Src)
	x := 10
	for _, size := range []int{22, 32, 48} {
		img := drawIcon(size)
		for y := 0; y < size*scale/2; y++ {
			for xx := 0; xx < size*scale/2; xx++ {
				c := color.NRGBAModel.Convert(img.At(xx*2/scale, y*2/scale)).(color.NRGBA)
				if c.A > 0 {
					out.SetNRGBA(x+xx, 10+y, c)
				}
			}
		}
		x += size*scale/2 + 5
	}
	f, _ := os.Create(path + ".tray-icon.png")
	defer f.Close()
	png.Encode(f, out)
}
