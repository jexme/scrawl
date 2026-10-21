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

// AlreadyNotified reports that the user was already told about version tag: the
// notification about a new version is shown once, not on every check.
func AlreadyNotified(tag string) bool {
	data, err := os.ReadFile(notifiedPath())
	return err == nil && strings.TrimSpace(string(data)) == tag
}

// MarkNotified remembers that version tag was announced.
func MarkNotified(tag string) error {
	path := notifiedPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(tag+"\n"), 0o600)
}
