// Package collection stores a collection of screenshots between program runs:
// "before" is captured in one run, "after" in another, then everything is glued together
// or sent separately.
package collection

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Dir returns the collection directory in the user's cache.
func Dir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "scrawl", "collection")
}

// Add puts a PNG into the collection and returns the path. The file name starts with the time,
// so the order in the collection matches the order of addition.
func Add(png []byte) (string, error) {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// The clock on Windows ticks less often than once per microsecond: two quick Adds would
	// get the same name, and the second shot would overwrite the first. O_EXCL prevents
	// overwriting, we shift the time by a microsecond.
	for ts := time.Now(); ; ts = ts.Add(time.Microsecond) {
		path := filepath.Join(dir, ts.Format("20060102-150405.000000")+".png")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(png); err != nil {
			f.Close()
			return "", err
		}
		return path, f.Close()
	}
}

// List returns the paths to the collection images in the order they were added.
func List() ([]string, error) {
	entries, err := os.ReadDir(Dir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			paths = append(paths, filepath.Join(Dir(), e.Name()))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// Count returns the number of images in the collection (0 on any error).
func Count() int {
	paths, _ := List()
	return len(paths)
}

// Clear deletes all the collection images. Only .png files in its directory are deleted.
func Clear() error {
	paths, err := List()
	if err != nil {
		return err
	}
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
