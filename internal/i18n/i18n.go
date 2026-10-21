// Package i18n translates interface texts. The source language is English: texts in the code
// are written in English and wrapped in T, and the other languages come from catalogs
// (lang_ru.go, lang_de.go and so on; the key is the English text). A text missing from
// a language catalog is shown as is, that is, in English, so a forgotten string
// does not break the program, and the completeness test (i18n_test.go) finds such strings.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// Language codes of the language setting.
const (
	Auto = "auto" // follow the system language
	RU   = "ru"
	EN   = "en"
	ZH   = "zh" // Chinese (Simplified)
	ES   = "es"
	PT   = "pt" // Portuguese (Brazil)
	JA   = "ja"
	DE   = "de"
	FR   = "fr"
	KO   = "ko"
)

// LanguageInfo describes an interface language: the code for the setting and its own name.
type LanguageInfo struct {
	Code string
	Name string // the language's own name: it must be recognizable without knowing the rest of the UI
	Hint string // Latin caption for scripts that may be missing from the font
}

// Languages lists all interface languages in display order.
var Languages = []LanguageInfo{
	{RU, "Русский", ""}, {EN, "English", ""}, {ZH, "简体中文", "Chinese"}, {ES, "Español", ""},
	{PT, "Português (Brasil)", ""}, {JA, "日本語", "Japanese"}, {DE, "Deutsch", ""}, {FR, "Français", ""}, {KO, "한국어", "Korean"},
}

// catalogs are translations from English: English is the source language and has no catalog.
var catalogs = map[string]map[string]string{
	RU: ru, ZH: zh, ES: es, PT: pt, JA: ja, DE: de, FR: fr, KO: ko,
}

var current atomic.Value // string: a language code from Languages

func init() { current.Store(EN) }

// Normalize converts a language entry to a code from the setting: "auto", "ru", "en", "zh",
// "es", "pt", "ja", "de", "fr", "ko". It understands regional forms ("zh-CN",
// "pt_BR", "ru_RU.UTF-8"). The second value is false if the language is unknown.
func Normalize(code string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(code))
	if v == Auto {
		return Auto, true
	}
	if i := strings.IndexAny(v, "_-.@"); i >= 0 {
		v = v[:i]
	}
	for _, l := range Languages {
		if v == l.Code {
			return l.Code, true
		}
	}
	return "", false
}

// SetLanguage selects the language: a code from Languages or "auto" (by LC_ALL, LC_MESSAGES,
// LANGUAGE and LANG: a known locale gives its own language, any other gives English).
// An unknown value is treated as "auto".
func SetLanguage(code string) {
	if canon, ok := Normalize(code); ok && canon != Auto {
		current.Store(canon)
		return
	}
	current.Store(detect(os.Getenv))
}

// Language returns the effective language: a code from Languages.
func Language() string { return current.Load().(string) }

// detect determines the language from environment variables.
func detect(getenv func(string) string) string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		v := getenv(name)
		if v == "" || v == "C" || v == "POSIX" || strings.EqualFold(v, "c") || strings.EqualFold(v, "posix") {
			continue
		}
		// LANGUAGE holds a colon-separated list of languages: take the first known one.
		for _, part := range strings.Split(v, ":") {
			if canon, ok := Normalize(part); ok && canon != Auto {
				return canon
			}
		}
		return EN // the locale is set, but there is no such translation
	}
	return EN
}

// T returns the translation of an English text into the effective language. With no
// translation for that language (or when the language is English): the text itself.
func T(text string) string {
	if v, ok := catalogs[Language()][text]; ok {
		return v
	}
	return text
}

// Tf translates a format and substitutes the arguments.
func Tf(format string, args ...any) string { return fmt.Sprintf(T(format), args...) }

// Errorf translates a format and creates an error; %w works as in fmt.Errorf.
func Errorf(format string, args ...any) error { return fmt.Errorf(T(format), args...) }
