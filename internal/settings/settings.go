// Package settings хранит пользовательские настройки в JSON-файле
// ~/.config/screen-editor/config.json. Файл можно править руками; отсутствующие
// и неверные значения заменяются значениями по умолчанию, а ключи, которых
// программа не знает (от новой версии или дописанные руками), при записи
// сохраняются.
//
// Схема задаётся структурой Settings: ключ файла берётся из json-тега поля,
// значение по умолчанию из Default, проверка в normalize. Новая настройка
// это новое поле и строка в Default/normalize.
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	def := Default()
	if !s.DoubleClick.valid() {
		s.DoubleClick = def.DoubleClick
	}
	if ValidateSaveDir(s.SaveDir) != nil {
		s.SaveDir = def.SaveDir
	}
}

// keyOf возвращает ключ файла для поля структуры, "" если поле в файл не пишется.
func keyOf(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// readRaw читает файл как набор ключей со значениями без разбора. Нет файла,
// он не JSON-объект или повреждён: ok == false.
func readRaw(path string) (raw map[string]json.RawMessage, exists, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, false
	}
	if json.Unmarshal(data, &raw) != nil || raw == nil {
		return nil, true, false
	}
	return raw, true, true
}

// Load читает настройки из файла. Каждый ключ разбирается отдельно: отсутствующий
// или неверный по типу ключ получает значение по умолчанию и не портит остальные.
// Нет файла или он повреждён: настройки по умолчанию.
func Load() Settings {
	s := Default()
	raw, _, ok := readRaw(Path())
	if !ok {
		return s
	}
	v := reflect.ValueOf(&s).Elem()
	for i := 0; i < v.NumField(); i++ {
		key := keyOf(v.Type().Field(i))
		r, found := raw[key]
		if key == "" || !found {
			continue
		}
		nv := reflect.New(v.Field(i).Type())
		if json.Unmarshal(r, nv.Interface()) == nil {
			v.Field(i).Set(nv.Elem())
		}
	}
	s.normalize()
	return s
}

// Save записывает настройки в файл атомарно: сначала во временный файл, потом
// переименование, чтобы оборвавшаяся запись не оставила битый файл. Свои ключи
// перезаписываются, чужие остаются как были. Нечитаемый файл перед этим
// сохраняется рядом как config.json.bak, чтобы не потерять правки руками.
func Save(s Settings) error {
	s.normalize()
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	merged, exists, ok := readRaw(path)
	if exists && !ok {
		os.Rename(path, path+".bak")
	}
	if merged == nil {
		merged = map[string]json.RawMessage{}
	}
	t := reflect.TypeOf(s)
	for i := 0; i < t.NumField(); i++ {
		delete(merged, keyOf(t.Field(i))) // поля с omitempty не должны оставлять старое значение
	}
	own, err := json.Marshal(s)
	if err != nil {
		return err
	}
	var ownRaw map[string]json.RawMessage
	if err := json.Unmarshal(own, &ownRaw); err != nil {
		return err
	}
	for k, r := range ownRaw {
		merged[k] = r
	}

	data, err := json.MarshalIndent(merged, "", "  ")
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
