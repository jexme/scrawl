package settingsui

import (
	"os"
	"path/filepath"
	"strings"
)

// lastTabPath is the file that remembers the tab on which the settings window was closed.
func lastTabPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "scrawl", "settings-tab")
}

// rememberedTab returns the number of the remembered tab, or 0 if there is none or that
// tab no longer exists.
func rememberedTab(tabs []settingsTab) int {
	data, err := os.ReadFile(lastTabPath())
	if err != nil {
		return 0
	}
	id := strings.TrimSpace(string(data))
	for i, t := range tabs {
		if t.id == id {
			return i
		}
	}
	return 0
}

// saveTab remembers the tab; a write error does not matter.
func saveTab(id string) {
	path := lastTabPath()
	if os.MkdirAll(filepath.Dir(path), 0o700) == nil {
		os.WriteFile(path, []byte(id+"\n"), 0o600)
	}
}
