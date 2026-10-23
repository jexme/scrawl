//go:build windows

package autostart

import (
	"errors"
	"os"

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
	return k.SetStringValue(regValueName, runCommand(bin))
}

// regHeal rewrites the Run value when it exists but differs from the command for bin.
func regHeal(bin string) (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer k.Close()
	cur, _, err := k.GetStringValue(regValueName)
	exists := err == nil
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return false, err
	}
	// %TEMP% and %TMP% can differ, and the process may see either spelling of the path.
	if !needsRepair(cur, exists, bin, os.TempDir(), os.Getenv("TEMP"), os.Getenv("TMP")) {
		return false, nil
	}
	return true, k.SetStringValue(regValueName, runCommand(bin))
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
