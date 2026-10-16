package settingsui

import (
	"os"
	"path/filepath"
	"strings"
)

// lastTabPath — файл, где помнится вкладка, на которой закрыли окно настроек.
func lastTabPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "scrawl", "settings-tab")
}

// rememberedTab возвращает номер запомненной вкладки или 0, если её нет или такой
// вкладки больше нет.
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

// saveTab запоминает вкладку; ошибка записи не важна.
func saveTab(id string) {
	path := lastTabPath()
	if os.MkdirAll(filepath.Dir(path), 0o700) == nil {
		os.WriteFile(path, []byte(id+"\n"), 0o600)
	}
}
