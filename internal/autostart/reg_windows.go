//go:build windows

package autostart

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// useRegistry: on Windows autostart lives in the registry, not in a .desktop file. Tests turn
// it off so as not to touch the real registry.
var useRegistry = true

// The autostart key and value name. Tests replace them with temporary ones.
var (
	regKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	regValueName = "Scrawl"
)

func regEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(regValueName)
	return err == nil
}

func regEnable(bin string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(regValueName, `"`+bin+`" -tray`)
}

func regDisable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(regValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}
