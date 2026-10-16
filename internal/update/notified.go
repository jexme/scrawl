package update

import (
	"os"
	"path/filepath"
	"strings"
)

func notifiedPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "scrawl", "update-notified")
}

// AlreadyNotified сообщает, что о версии tag уже сказали пользователю: уведомление
// о новой версии показывается один раз, а не при каждой проверке.
func AlreadyNotified(tag string) bool {
	data, err := os.ReadFile(notifiedPath())
	return err == nil && strings.TrimSpace(string(data)) == tag
}

// MarkNotified запоминает, что о версии tag сказали.
func MarkNotified(tag string) error {
	path := notifiedPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(tag+"\n"), 0o600)
}
