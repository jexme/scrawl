// Package collection хранит подборку скриншотов между запусками программы:
// «было» снимается в одном запуске, «стало» в другом, потом всё склеивается
// или отправляется по отдельности.
package collection

import (
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
	return filepath.Join(base, "scrawl", "collection")
}

// Add кладёт PNG в подборку и возвращает путь. Имя файла начинается со времени,
// поэтому порядок в подборке совпадает с порядком добавления.
func Add(png []byte) (string, error) {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// Часы на Windows тикают реже микросекунды: два быстрых Add получили бы одно имя,
	// и второй снимок затёр бы первый. O_EXCL не даёт перезаписать, время сдвигаем на микросекунду.
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
