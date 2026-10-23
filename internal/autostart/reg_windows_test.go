//go:build windows

package autostart

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// Tests the registry branch on a temporary key, not on the real Run.
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
	if v, _, _ := k.GetStringValue(regValueName); v != `"C:\Program Files\Scrawl\scrawl.exe" -tray -autostart` {
		t.Errorf("команда автозапуска: %q", v)
	}
	if err := Enable("scrawl.exe"); err == nil {
		t.Error("относительный путь должен отклоняться")
	}
	if err := Disable(); err != nil || Enabled() {
		t.Errorf("после Disable автозапуска быть не должно: %v", err)
	}
}

// Heal rewrites an existing entry that points elsewhere and never switches autostart on.
// Runs on a temporary key; it needs Windows, so Linux can only compile it.
func TestRegistryHeal(t *testing.T) {
	oldKey, oldName := regKeyPath, regValueName
	regKeyPath, regValueName = `Software\ScrawlTest\Heal`, "Scrawl"
	t.Cleanup(func() {
		registry.DeleteKey(registry.CURRENT_USER, regKeyPath)
		registry.DeleteKey(registry.CURRENT_USER, `Software\ScrawlTest`)
		regKeyPath, regValueName = oldKey, oldName
	})
	setRegistry(t, true)

	const bin = `C:\Apps\Scrawl\scrawl.exe`
	if changed, err := Heal(bin); err != nil || changed || Enabled() {
		t.Fatalf("a disabled autostart must stay disabled: changed=%v err=%v enabled=%v", changed, err, Enabled())
	}
	if err := Enable(`C:\Old\scrawl.exe`); err != nil {
		t.Fatal(err)
	}
	if changed, err := Heal(bin); err != nil || !changed {
		t.Fatalf("an entry for another path must be rewritten: changed=%v err=%v", changed, err)
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	if v, _, _ := k.GetStringValue(regValueName); v != runCommand(bin) {
		t.Errorf("after Heal: %q", v)
	}
	if changed, err := Heal(bin); err != nil || changed {
		t.Errorf("a second Heal must change nothing: changed=%v err=%v", changed, err)
	}
}
