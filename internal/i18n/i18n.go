// Package i18n переводит тексты интерфейса. Исходный язык английский: в коде тексты
// записаны по-английски и обёрнуты в T, а остальные языки берутся из каталогов
// (lang_ru.go, lang_de.go и так далее; ключ это английский текст). Текст, которого нет
// в каталоге языка, показывается как есть, то есть по-английски, поэтому забытая строка
// не ломает программу, а тест полноты (i18n_test.go) такие строки находит.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// Коды языков настройки language.
const (
	Auto = "auto" // по языку системы
	RU   = "ru"
	EN   = "en"
	ZH   = "zh" // китайский (упрощённый)
	ES   = "es"
	PT   = "pt" // португальский (Бразилия)
	JA   = "ja"
	DE   = "de"
	FR   = "fr"
	KO   = "ko"
)

// LanguageInfo описывает язык интерфейса: код для настройки и название на нём самом.
type LanguageInfo struct {
	Code string
	Name string // как язык называется сам: его нужно узнать, даже не зная остального интерфейса
	Hint string // латинская подпись для письменностей, которых может не быть в шрифте
}

// Languages — все языки интерфейса в порядке показа.
var Languages = []LanguageInfo{
	{RU, "Русский", ""}, {EN, "English", ""}, {ZH, "简体中文", "Chinese"}, {ES, "Español", ""},
	{PT, "Português (Brasil)", ""}, {JA, "日本語", "Japanese"}, {DE, "Deutsch", ""}, {FR, "Français", ""}, {KO, "한국어", "Korean"},
}

// catalogs — переводы с английского: английский язык исходный и каталога не имеет.
var catalogs = map[string]map[string]string{
	RU: ru, ZH: zh, ES: es, PT: pt, JA: ja, DE: de, FR: fr, KO: ko,
}

var current atomic.Value // string: код языка из Languages

func init() { current.Store(EN) }

// Normalize приводит запись языка к коду из настройки: "auto", "ru", "en", "zh",
// "es", "pt", "ja", "de", "fr", "ko". Понимает региональные записи ("zh-CN",
// "pt_BR", "ru_RU.UTF-8"). Второе значение false, если язык неизвестен.
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

// SetLanguage выбирает язык: код из Languages или "auto" (по LC_ALL, LC_MESSAGES,
// LANGUAGE и LANG: известная локаль даёт свой язык, любая другая английский).
// Неизвестное значение считается "auto".
func SetLanguage(code string) {
	if canon, ok := Normalize(code); ok && canon != Auto {
		current.Store(canon)
		return
	}
	current.Store(detect(os.Getenv))
}

// Language возвращает действующий язык: код из Languages.
func Language() string { return current.Load().(string) }

// detect определяет язык по переменным окружения.
func detect(getenv func(string) string) string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		v := getenv(name)
		if v == "" || v == "C" || v == "POSIX" || strings.EqualFold(v, "c") || strings.EqualFold(v, "posix") {
			continue
		}
		// В LANGUAGE языки идут списком через двоеточие: берём первый известный.
		for _, part := range strings.Split(v, ":") {
			if canon, ok := Normalize(part); ok && canon != Auto {
				return canon
			}
		}
		return EN // локаль задана, но такого перевода нет
	}
	return EN
}

// T возвращает перевод английского текста на действующий язык. Нет перевода на этот
// язык (или язык английский): сам текст.
func T(text string) string {
	if v, ok := catalogs[Language()][text]; ok {
		return v
	}
	return text
}

// Tf переводит формат и подставляет аргументы.
func Tf(format string, args ...any) string { return fmt.Sprintf(T(format), args...) }

// Errorf переводит формат и создаёт ошибку; %w работает как в fmt.Errorf.
func Errorf(format string, args ...any) error { return fmt.Errorf(T(format), args...) }
