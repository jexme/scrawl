// Package settings хранит пользовательские настройки в JSON-файле
// ~/.config/screen-editor/config.json. Файл можно править руками; отсутствующие
// и неверные значения заменяются значениями по умолчанию (числа вне допустимого
// диапазона подрезаются до его границ), а ключи, которых
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
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// DoubleClick определяет, что делает двойной клик внутри выделенной области.
type DoubleClick string

const (
	DoubleClickOff      DoubleClick = "off"       // ничего (по умолчанию)
	DoubleClickCopy     DoubleClick = "copy"      // скопировать в буфер обмена
	DoubleClickCopySave DoubleClick = "copy_save" // скопировать и сохранить в файл
)

// Enter определяет, что делает клавиша Enter в оверлее.
type Enter string

const (
	EnterCopySave Enter = "copy_save" // скопировать и сохранить в файл (по умолчанию)
	EnterCopy     Enter = "copy"      // только скопировать в буфер обмена
	EnterSave     Enter = "save"      // только сохранить в файл
)

// Targets сообщает, куда отправлять снимок: в буфер обмена и/или в файл.
func (e Enter) Targets() (clipboard, file bool) {
	return e != EnterSave, e != EnterCopy
}

// Settings — все настройки.
type Settings struct {
	DoubleClick DoubleClick `json:"double_click"`
	Enter       Enter       `json:"enter"`
	// Notifications — показывать уведомления после снимка и действий с подборкой.
	// Ошибки и подсказки, почему действие не вышло, показываются всегда.
	Notifications bool `json:"notifications"`
	// RememberRegion — запоминать область снимка и предлагать её при следующем запуске.
	RememberRegion bool `json:"remember_region"`
	// FilePrefix — начало имени файла скриншота. Пусто: DefaultFilePrefix.
	FilePrefix string `json:"file_prefix,omitempty"`
	// DimPercent — затемнение вне выделенной области, 0-100 %.
	DimPercent int `json:"dim_percent"`
	// Рамка выделения: толщина линии, длина штриха и пауза между штрихами в
	// пикселях экрана, цвет в виде #RRGGBB.
	BorderWidth float64 `json:"border_width"`
	BorderDash  float64 `json:"border_dash"`
	BorderGap   float64 `json:"border_gap"`
	BorderColor string  `json:"border_color"`
	// HandleRadius — радиус угловых ручек в пикселях экрана; 0 прячет ручки.
	HandleRadius float64 `json:"handle_radius"`
	// ShowSize — показывать подпись «W × H» у выделенной области.
	ShowSize bool `json:"show_size"`
	// DefaultTool — инструмент, который включается сразу после выбора области
	// (одно из ToolNames).
	DefaultTool string `json:"default_tool"`
	// Tools — цвет и толщина по умолчанию для каждого инструмента.
	Tools ToolStyles `json:"tools"`
	// Palette — цвета панели инструментов, #RRGGBB.
	Palette [PaletteSize]string `json:"palette"`
	// MarkerOpacity — непрозрачность маркера, 5-100 %.
	MarkerOpacity int `json:"marker_opacity"`
	// Glue — склейка «было/стало».
	Glue Glue `json:"glue"`
	// Format — формат сохраняемых файлов; в буфер обмена всегда идёт PNG.
	Format FileFormat `json:"format"`
	// JPEGQuality — качество JPEG, 1-100.
	JPEGQuality int `json:"jpeg_quality"`
	// KeepOpen — после копирования или сохранения не закрывать оверлей, а дать
	// продолжить правку. Выход по Esc.
	KeepOpen bool `json:"keep_open"`
	// IncludeCursor — рисовать курсор мыши в снимке (только при захвате через PipeWire).
	IncludeCursor bool `json:"include_cursor"`
	// SaveDir — каталог для сохранения скриншотов. Пусто: ~/Pictures/Screenshots.
	SaveDir string `json:"save_dir,omitempty"`
}

// Default возвращает настройки по умолчанию.
func Default() Settings {
	return Settings{DoubleClick: DoubleClickOff, Enter: EnterCopySave, Notifications: true, RememberRegion: true, DimPercent: 86,
		BorderWidth: 1, BorderDash: 4.6, BorderGap: 2, BorderColor: "#FFFFFF", HandleRadius: 3, ShowSize: true, DefaultTool: "select", Palette: DefaultPalette, MarkerOpacity: 43, Glue: DefaultGlue(), Format: FormatPNG, JPEGQuality: 90}
}

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

func (e Enter) valid() bool { return e == EnterCopySave || e == EnterCopy || e == EnterSave }

// normalize заменяет неверные значения на значения по умолчанию.
func (s *Settings) normalize() {
	def := Default()
	if !s.DoubleClick.valid() {
		s.DoubleClick = def.DoubleClick
	}
	if !s.Enter.valid() {
		s.Enter = def.Enter
	}
	s.DimPercent = min(max(s.DimPercent, 0), 100)
	s.BorderWidth = min(max(s.BorderWidth, MinBorderWidth), MaxBorderWidth)
	s.BorderDash = min(max(s.BorderDash, MinBorderDash), MaxBorderDash)
	s.BorderGap = min(max(s.BorderGap, 0), MaxBorderGap)
	if !validTool(s.DefaultTool) {
		s.DefaultTool = def.DefaultTool
	}
	s.Tools.normalize()
	normalizePalette(&s.Palette)
	s.Glue.normalize()
	if s.Format != FormatPNG && s.Format != FormatJPEG {
		s.Format = def.Format
	}
	s.JPEGQuality = min(max(s.JPEGQuality, 1), 100)
	s.MarkerOpacity = min(max(s.MarkerOpacity, MinMarkerOpacity), 100)
	s.HandleRadius = min(max(s.HandleRadius, 0), MaxHandleRadius)
	if c, err := ParseColor(s.BorderColor); err == nil {
		s.BorderColor = FormatColor(c)
	} else {
		s.BorderColor = def.BorderColor
	}
	if ValidateFilePrefix(s.FilePrefix) != nil {
		s.FilePrefix = def.FilePrefix
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

// decodeStruct разбирает каждое поле структуры по его ключу отдельно: поле с
// неверным значением остаётся как было. Вложенные структуры разбираются так же,
// поэтому одна ошибка не портит соседние значения.
func decodeStruct(v reflect.Value, raw map[string]json.RawMessage) {
	for i := 0; i < v.NumField(); i++ {
		key := keyOf(v.Type().Field(i))
		r, found := raw[key]
		if key == "" || !found {
			continue
		}
		field := v.Field(i)
		if field.Kind() == reflect.Struct {
			var inner map[string]json.RawMessage
			if json.Unmarshal(r, &inner) == nil && inner != nil {
				decodeStruct(field, inner)
			}
			continue
		}
		nv := reflect.New(field.Type())
		if json.Unmarshal(r, nv.Interface()) == nil {
			field.Set(nv.Elem())
		}
	}
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
	decodeStruct(reflect.ValueOf(&s).Elem(), raw)
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

// Update применяет изменение и сохраняет результат (если оно что-то изменило). Если запись на диск не
// удалась, в памяти настройка всё равно меняется, а ошибка возвращается.
func (st *Store) Update(change func(*Settings)) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	before := st.s
	change(&st.s)
	st.s.normalize()
	if st.s == before {
		return nil // ничего не изменилось: файл не трогаем
	}
	return Save(st.s)
}

// Границы значений рамки выделения.
const (
	MinBorderWidth, MaxBorderWidth = 0.5, 4
	MinBorderDash, MaxBorderDash   = 1, 20
	MaxBorderGap                   = 20
	MaxHandleRadius                = 10
)

// ParseColor разбирает цвет вида #RRGGBB (решётка необязательна).
func ParseColor(s string) (color.NRGBA, error) {
	hex := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(hex) != 6 {
		return color.NRGBA{}, errors.New("нужен цвет вида #RRGGBB")
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.NRGBA{}, errors.New("нужен цвет вида #RRGGBB")
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, nil
}

// FormatColor записывает цвет как #RRGGBB.
func FormatColor(c color.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// BorderNRGBA возвращает цвет рамки (после normalize он всегда верный).
func (s Settings) BorderNRGBA() color.NRGBA {
	c, err := ParseColor(s.BorderColor)
	if err != nil {
		c, _ = ParseColor(Default().BorderColor)
	}
	return c
}

// MinMarkerOpacity — наименьшая непрозрачность маркера в процентах: ниже его не видно.
const MinMarkerOpacity = 5

// MarkerAlpha возвращает непрозрачность маркера как альфа-канал 0-255.
func (s Settings) MarkerAlpha() uint8 {
	return uint8(math.Round(float64(s.MarkerOpacity) * 255 / 100))
}

// DefaultFilePrefix — начало имени файла, если своё не задано.
const DefaultFilePrefix = "scrawl"

// StampLayout — формат даты и времени в имени файла.
const StampLayout = "20060102-150405"

const maxFilePrefix = 60

// ValidateFilePrefix проверяет начало имени файла: пусто (значит по умолчанию) или
// текст без разделителей пути и управляющих символов, не длиннее 60 знаков.
func ValidateFilePrefix(p string) error {
	if utf8.RuneCountInString(p) > maxFilePrefix {
		return fmt.Errorf("не длиннее %d знаков", maxFilePrefix)
	}
	for _, r := range p {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return errors.New("нельзя использовать / \\ и управляющие символы")
		}
	}
	return nil
}

// Prefix возвращает начало имени файла с учётом значения по умолчанию.
func (s Settings) Prefix() string {
	if s.FilePrefix == "" {
		return DefaultFilePrefix
	}
	return s.FilePrefix
}

// FileFormat — формат файла скриншота.
type FileFormat string

const (
	FormatPNG  FileFormat = "png"  // без потерь (по умолчанию)
	FormatJPEG FileFormat = "jpeg" // меньше размер, есть потери
)

// Ext возвращает расширение файла вместе с точкой.
func (s Settings) Ext() string {
	if s.Format == FormatJPEG {
		return ".jpg"
	}
	return ".png"
}

// OutputDir возвращает каталог для скриншотов: из настроек или
// ~/Pictures/Screenshots, если в настройках пусто. Каталог создаётся.
func (s Settings) OutputDir() (string, error) {
	dir, err := ExpandDir(s.SaveDir)
	if err != nil {
		return "", err
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, "Pictures", "Screenshots")
	}
	return dir, os.MkdirAll(dir, 0o755)
}

// FileName возвращает имя файла скриншота для отметки времени stamp (StampLayout).
func (s Settings) FileName(stamp string) string { return s.Prefix() + "-" + stamp + s.Ext() }

// NumberedFileName возвращает имя n-го файла набора с общей отметкой времени.
func (s Settings) NumberedFileName(stamp string, n int) string {
	return fmt.Sprintf("%s-%s-%d%s", s.Prefix(), stamp, n, s.Ext())
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
