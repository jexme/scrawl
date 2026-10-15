// Package collection хранит подборку скриншотов между запусками программы:
// «было» снимается в одном запуске, «стало» в другом, потом всё склеивается
// или отправляется по отдельности.
package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Dir возвращает каталог подборки в кэше пользователя.
func Dir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "screen-editor", "collection")
}

// Add кладёт PNG в подборку и возвращает путь. Имя файла начинается со времени,
// поэтому порядок в подборке совпадает с порядком добавления.
func Add(png []byte) (string, error) {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s.png", time.Now().Format("20060102-150405.000000"))
	path := filepath.Join(dir, name)
	return path, os.WriteFile(path, png, 0o600)
}

// List возвращает пути к картинкам подборки в порядке добавления.
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

// Count возвращает число картинок в подборке (0 при любой ошибке).
func Count() int {
	paths, _ := List()
	return len(paths)
}

// Clear удаляет все картинки подборки. Удаляются только .png в её каталоге.
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
