//go:build !windows

package autostart

const useRegistry = false

func regEnabled() bool             { return false }
func regEnable(string) error       { return nil }
func regDisable() error            { return nil }
func regHeal(string) (bool, error) { return false, nil }
