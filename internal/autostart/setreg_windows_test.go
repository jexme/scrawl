//go:build windows

package autostart

import "testing"

func setRegistry(t *testing.T, on bool) {
	old := useRegistry
	useRegistry = on
	t.Cleanup(func() { useRegistry = old })
}
