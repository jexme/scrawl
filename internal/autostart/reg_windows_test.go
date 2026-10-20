//go:build windows

package autostart

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// Проверяет реестровую ветку на временном ключе, а не на настоящем Run.
func TestRegistryEnableDisable(t *testing.T) {
	oldKey, oldName := regKeyPath, regValueName
	regKeyPath, regValueName = `Software\ScrawlTest\Run`, "Scrawl"
	t.Cleanup(func() {
		registry.DeleteKey(registry.CURRENT_USER, regKeyPath)
		registry.DeleteKey(registry.CURRENT_USER, `Software\ScrawlTest`)
		regKeyPath, regValueName = oldKey, oldName
	})
	setRegistry(t, true)

	if Enabled() {
		t.Fatal("в чистом ключе автозапуска нет")
	}
	if err := Disable(); err != nil {
		t.Errorf("выключить выключенный автозапуск не ошибка: %v", err)
	}
	if err := Enable(`C:\Program Files\Scrawl\scrawl.exe`); err != nil {
		t.Fatal(err)
	}
	if !Enabled() {
		t.Error("после Enable автозапуск должен быть включён")
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	if v, _, _ := k.GetStringValue(regValueName); v != `"C:\Program Files\Scrawl\scrawl.exe" -tray` {
		t.Errorf("команда автозапуска: %q", v)
	}
	if err := Enable("scrawl.exe"); err == nil {
		t.Error("относительный путь должен отклоняться")
	}
	if err := Disable(); err != nil || Enabled() {
		t.Errorf("после Disable автозапуска быть не должно: %v", err)
	}
}
