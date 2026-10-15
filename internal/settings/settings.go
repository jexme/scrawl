// Package settings хранит пользовательские настройки в JSON-файле
// ~/.config/screen-editor/config.json. Файл можно править руками; отсутствующие
// и неверные значения заменяются значениями по умолчанию.
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DoubleClick определяет, что делает двойной клик внутри выделенной области.
type DoubleClick string

const (
	DoubleClickOff      DoubleClick = "off"       // ничего (по умолчанию)
	DoubleClickCopy     DoubleClick = "copy"      // скопировать в буфер обмена
	DoubleClickCopySave DoubleClick = "copy_save" // скопировать и сохранить в файл
)

// Settings — все настройки.
type Settings struct {
	DoubleClick DoubleClick `json:"double_click"`
	// SaveDir — каталог для сохранения скриншотов. Пусто: ~/Pictures/Screenshots.
	SaveDir string `json:"save_dir,omitempty"`
}

// Default возвращает настройки по умолчанию.
func Default() Settings { return Settings{DoubleClick: DoubleClickOff} }

// Path возвращает путь к файлу настроек.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "screen-editor", "config.json")
}

// valid проверяет значение двойного клика.
func (d DoubleClick) valid() bool {
	return d == DoubleClickOff || d == DoubleClickCopy || d == DoubleClickCopySave
}

// normalize заменяет неверные значения на значения по умолчанию.
func (s *Settings) normalize() {
	if !s.DoubleClick.valid() {
		s.DoubleClick = DoubleClickOff
	}
}

// Load читает настройки из файла. Нет файла или он повреждён: настройки по умолчанию.
func Load() Settings {
	s := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		return s
	}
	var read Settings
	if json.Unmarshal(data, &read) != nil {
		return s
	}
	s = read
	s.normalize()
	return s
}

// Save записывает настройки в файл целиком и атомарно: сначала во временный файл,
// потом переименование, чтобы оборвавшаяся запись не оставила битый файл.
func Save(s Settings) error {
	s.normalize()
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Store — общие настройки процесса: все окна читают и меняют одни и те же
// значения, а каждое изменение сразу пишется на диск.
type Store struct {
	mu sync.Mutex
	s  Settings
}

// NewStore загружает настройки с диска.
func NewStore() *Store { return &Store{s: Load()} }

// Get возвращает текущие настройки.
func (st *Store) Get() Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.s
}

// Update применяет изменение и сохраняет результат. Если запись на диск не
// удалась, в памяти настройка всё равно меняется, а ошибка возвращается.
func (st *Store) Update(change func(*Settings)) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	change(&st.s)
	st.s.normalize()
	return Save(st.s)
}

// ValidateSaveDir проверяет значение папки сохранения: пусто (значит папка по
// умолчанию), абсолютный путь или путь от домашнего каталога (~ и ~/...).
func ValidateSaveDir(dir string) error {
	if dir == "" || dir == "~" || strings.HasPrefix(dir, "~/") || filepath.IsAbs(dir) {
		return nil
	}
	return errors.New("путь должен быть абсолютным или начинаться с ~/")
}

// ExpandDir раскрывает ~ в начале пути до домашнего каталога. Пустая строка
// остаётся пустой: это папка по умолчанию.
func ExpandDir(dir string) (string, error) {
	if dir != "~" && !strings.HasPrefix(dir, "~/") {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, strings.TrimPrefix(dir, "~")), nil
}
