// Package settings stores user settings in the JSON file
// ~/.config/scrawl/config.json. The file can be edited by hand; missing and invalid
// values are replaced with defaults (numbers outside the allowed range are clamped
// to its bounds), and keys the program does not know (from a newer version or added
// by hand) are preserved on write.
//
// The schema is the Settings struct: the file key comes from the field's json tag,
// the default value from Default, validation from normalize. A new setting is
// a new field plus a line in Default/normalize.
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

	"github.com/jexme/scrawl/internal/i18n"
)

// DoubleClick defines what a double click inside the selected area does.
type DoubleClick string

const (
	DoubleClickOff      DoubleClick = "off"       // nothing
	DoubleClickCopy     DoubleClick = "copy"      // copy to the clipboard (default)
	DoubleClickCopySave DoubleClick = "copy_save" // copy and save to a file
)

// Enter defines what the Enter key does in the overlay.
type Enter string

const (
	EnterCopySave Enter = "copy_save" // copy and save to a file (default)
	EnterCopy     Enter = "copy"      // only copy to the clipboard
	EnterSave     Enter = "save"      // only save to a file
)

// Targets reports where to send the screenshot: the clipboard and/or a file.
func (e Enter) Targets() (clipboard, file bool) {
	return e != EnterSave, e != EnterCopy
}

// Settings holds all the settings.
type Settings struct {
	DoubleClick DoubleClick `json:"double_click"`
	Enter       Enter       `json:"enter"`
	// Notifications shows notifications after a screenshot and after collection actions.
	// Errors and hints about why an action failed are always shown.
	Notifications bool `json:"notifications"`
	// RememberRegion remembers the screenshot area and offers it on the next launch.
	RememberRegion bool `json:"remember_region"`
	// FilePrefix is the start of the screenshot file name. Empty means DefaultFilePrefix.
	FilePrefix string `json:"file_prefix,omitempty"`
	// DimPercent is the dimming outside the selected area, 0-100 %.
	DimPercent int `json:"dim_percent"`
	// The selection border: line thickness, dash length and gap between dashes in
	// screen pixels, colour as #RRGGBB.
	BorderWidth float64 `json:"border_width"`
	BorderDash  float64 `json:"border_dash"`
	BorderGap   float64 `json:"border_gap"`
	BorderColor string  `json:"border_color"`
	// HandleRadius is the radius of the corner handles in screen pixels; 0 hides them.
	HandleRadius float64 `json:"handle_radius"`
	// ShowSize shows the "W × H" label next to the selected area.
	ShowSize bool `json:"show_size"`
	// CustomColor is the custom colour from the toolbar palette, #RRGGBB; empty means none chosen.
	// Written by the program itself when a colour is picked in the overlay.
	CustomColor string `json:"custom_color,omitempty"`
	// ShowLoupe shows a magnifier with enlarged pixels while the area is being selected or
	// the border is dragged.
	ShowLoupe bool `json:"show_loupe"`
	// DefaultTool is the tool that is enabled right after the area is selected
	// (one of ToolNames).
	DefaultTool string `json:"default_tool"`
	// Tools holds the default colour and thickness for each tool.
	Tools ToolStyles `json:"tools"`
	// Palette holds the toolbar colours, #RRGGBB.
	Palette [PaletteSize]string `json:"palette"`
	// MarkerOpacity is the marker opacity, 5-100 %.
	MarkerOpacity int `json:"marker_opacity"`
	// Glue holds the "before/after" collage settings.
	Glue Glue `json:"glue"`
	// Format is the format of saved files; the clipboard always gets PNG.
	Format FileFormat `json:"format"`
	// JPEGQuality is the JPEG quality, 1-100.
	JPEGQuality int `json:"jpeg_quality"`
	// KeepOpen keeps the overlay open after copying or saving so editing can continue.
	// Leave with Esc.
	KeepOpen bool `json:"keep_open"`
	// Language is the interface language: "auto" (follow the system), "ru" or "en".
	Language string `json:"language"`
	// CheckUpdates asks GitHub about a new version when the tray icon and the settings
	// window start. The program's only access to the internet; off by default.
	CheckUpdates bool `json:"check_updates"`
	// Keys holds the overlay action keys; empty means the defaults.
	Keys Keys `json:"keys"`
	// IncludeCursor draws the mouse cursor in the screenshot (only for PipeWire capture).
	IncludeCursor bool `json:"include_cursor"`
	// SaveDir is the directory for saving screenshots. Empty means ~/Pictures/Screenshots.
	SaveDir string `json:"save_dir,omitempty"`
}

// Default returns the default settings.
func Default() Settings {
	return Settings{DoubleClick: DoubleClickCopy, Enter: EnterCopySave, Notifications: true, RememberRegion: true, DimPercent: 86,
		BorderWidth: 1, BorderDash: 4.6, BorderGap: 2, BorderColor: "#FFFFFF", HandleRadius: 3, ShowSize: true, ShowLoupe: true, DefaultTool: "select", Palette: DefaultPalette, MarkerOpacity: 43, Glue: DefaultGlue(), Format: FormatPNG, JPEGQuality: 90, Language: i18n.Auto}
}

// Settings directory names: the current one and the one from the program's former name
// (screen-editor).
const (
	dirName       = "scrawl"
	legacyDirName = "screen-editor"
)

func configFile(dirName string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, dirName, "config.json")
}

// Path returns the path to the settings file.
func Path() string { return configFile(dirName) }

// LegacyPath returns the path to the settings file of the program's former name.
// Settings are read from it while the new file does not exist; it is never changed or deleted.
func LegacyPath() string { return configFile(legacyDirName) }

// valid checks the double click value.
func (d DoubleClick) valid() bool {
	return d == DoubleClickOff || d == DoubleClickCopy || d == DoubleClickCopySave
}

func (e Enter) valid() bool { return e == EnterCopySave || e == EnterCopy || e == EnterSave }

// normalize replaces invalid values with defaults.
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
	if canon, ok := i18n.Normalize(s.Language); ok {
		s.Language = canon // "zh-CN" and "pt_BR" are stored as zh and pt
	} else {
		s.Language = def.Language
	}
	s.Keys.normalize()
	if c, err := ParseColor(s.CustomColor); err == nil {
		s.CustomColor = FormatColor(c)
	} else {
		s.CustomColor = ""
	}
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

// keyOf returns the file key for a struct field, or "" if the field is not written to the file.
func keyOf(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// readRaw reads the file as a set of keys with undecoded values. No file, not a JSON
// object, or corrupted: ok == false.
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

// decodeStruct decodes each struct field by its own key: a field with an invalid
// value stays as it was. Nested structs are decoded the same way, so one error
// does not spoil neighbouring values.
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

// readConfigRaw reads the settings file, or, if it does not exist yet, the file of the
// program's former name. From the former file only the contents are returned:
// exists is false for it, so a write does not mistake it for a corrupted file at
// the new path.
func readConfigRaw() (raw map[string]json.RawMessage, exists, ok bool) {
	raw, exists, ok = readRaw(Path())
	if exists {
		return raw, exists, ok
	}
	if legacy, lexists, lok := readRaw(LegacyPath()); lexists && lok {
		return legacy, false, true
	}
	return nil, false, false
}

// Load reads the settings from the file. Each key is decoded separately: a missing key
// or one of the wrong type gets its default value and does not spoil the others.
// No file or a corrupted one: default settings.
func Load() Settings {
	s := Default()
	raw, _, ok := readConfigRaw()
	if !ok {
		return s
	}
	decodeStruct(reflect.ValueOf(&s).Elem(), raw)
	s.normalize()
	return s
}

// Save writes the settings to the file atomically: first to a temporary file, then a
// rename, so an interrupted write does not leave a broken file. Our own keys are
// overwritten, foreign ones stay as they were. An unreadable file is first saved
// alongside as config.json.bak so that hand edits are not lost.
func Save(s Settings) error {
	s.normalize()
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	merged, exists, ok := readConfigRaw()
	if exists && !ok {
		os.Rename(path, path+".bak")
	}
	if merged == nil {
		merged = map[string]json.RawMessage{}
	}
	t := reflect.TypeOf(s)
	for i := 0; i < t.NumField(); i++ {
		delete(merged, keyOf(t.Field(i))) // omitempty fields must not leave the old value behind
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

// Store holds the process-wide settings: all windows read and change the same
// values, and every change is written to disk immediately.
type Store struct {
	mu sync.Mutex
	s  Settings
}

// NewStore loads the settings from disk.
func NewStore() *Store { return &Store{s: Load()} }

// Get returns the current settings.
func (st *Store) Get() Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.s
}

// Update applies a change to the settings as they are on disk now and saves the result (if it
// changed anything). The file is read again first because the overlay and the settings window
// are separate processes: saving this process's older copy of everything would silently undo
// what the other one saved meanwhile (for example a custom colour picked in the overlay). If
// writing to disk fails, the setting still changes in memory and the error is returned.
func (st *Store) Update(change func(*Settings)) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	disk := Load()
	next := disk
	change(&next)
	next.normalize()
	st.s = next
	if next == disk {
		return nil // nothing changed: leave the file alone
	}
	return Save(next)
}

// Bounds of the selection border values.
const (
	MinBorderWidth, MaxBorderWidth = 0.5, 4
	MinBorderDash, MaxBorderDash   = 1, 20
	MaxBorderGap                   = 20
	MaxHandleRadius                = 10
)

// ParseColor parses a colour of the form #RRGGBB (the # is optional).
func ParseColor(s string) (color.NRGBA, error) {
	hex := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(hex) != 6 {
		return color.NRGBA{}, errors.New(i18n.T("a colour like #RRGGBB is needed"))
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.NRGBA{}, errors.New(i18n.T("a colour like #RRGGBB is needed"))
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, nil
}

// FormatColor writes a colour as #RRGGBB.
func FormatColor(c color.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// BorderNRGBA returns the border colour (always valid after normalize).
func (s Settings) BorderNRGBA() color.NRGBA {
	c, err := ParseColor(s.BorderColor)
	if err != nil {
		c, _ = ParseColor(Default().BorderColor)
	}
	return c
}

// MinMarkerOpacity is the lowest marker opacity in percent: below it the marker is invisible.
const MinMarkerOpacity = 5

// MarkerAlpha returns the marker opacity as a 0-255 alpha channel.
func (s Settings) MarkerAlpha() uint8 {
	return uint8(math.Round(float64(s.MarkerOpacity) * 255 / 100))
}

// DefaultFilePrefix is the start of the file name when none is set.
const DefaultFilePrefix = "scrawl"

// StampLayout is the date and time format used in the file name.
const StampLayout = "20060102-150405"

const maxFilePrefix = 60

// ValidateFilePrefix checks the start of the file name: empty (meaning the default) or
// text without path separators and control characters, at most 60 characters.
func ValidateFilePrefix(p string) error {
	if utf8.RuneCountInString(p) > maxFilePrefix {
		return fmt.Errorf(i18n.T("at most %d characters"), maxFilePrefix)
	}
	for _, r := range p {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return errors.New(i18n.T("/ \\ and control characters are not allowed"))
		}
	}
	return nil
}

// Prefix returns the start of the file name, taking the default into account.
func (s Settings) Prefix() string {
	if s.FilePrefix == "" {
		return DefaultFilePrefix
	}
	return s.FilePrefix
}

// FileFormat is the screenshot file format.
type FileFormat string

const (
	FormatPNG  FileFormat = "png"  // lossless (default)
	FormatJPEG FileFormat = "jpeg" // smaller size, lossy
)

// Ext returns the file extension including the dot.
func (s Settings) Ext() string {
	if s.Format == FormatJPEG {
		return ".jpg"
	}
	return ".png"
}

// OutputDir returns the directory for screenshots: from the settings or
// ~/Pictures/Screenshots if the setting is empty. The directory is created.
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

// FileName returns the screenshot file name for the timestamp stamp (StampLayout).
func (s Settings) FileName(stamp string) string { return s.Prefix() + "-" + stamp + s.Ext() }

// NumberedFileName returns the name of the n-th file of a set sharing one timestamp.
func (s Settings) NumberedFileName(stamp string, n int) string {
	return fmt.Sprintf("%s-%s-%d%s", s.Prefix(), stamp, n, s.Ext())
}

// ValidateSaveDir checks the save directory value: empty (meaning the default
// directory), an absolute path or a path from the home directory (~ and ~/...).
func ValidateSaveDir(dir string) error {
	if dir == "" || dir == "~" || strings.HasPrefix(dir, "~/") || filepath.IsAbs(dir) {
		return nil
	}
	return errors.New(i18n.T("the path must be absolute or start with ~/"))
}

// ExpandDir expands a leading ~ to the home directory. An empty string stays empty:
// it means the default directory.
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
