// Package i18n переводит тексты интерфейса. Исходный язык русский: в коде тексты
// записаны по-русски и обёрнуты в T, а английский берётся из каталога en. Текст,
// которого нет в каталоге, показывается как есть, поэтому забытая строка
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
)

var current atomic.Value // string: RU или EN

func init() { current.Store(RU) }

// SetLanguage выбирает язык: "ru", "en" или "auto" (по LC_ALL, LC_MESSAGES,
// LANGUAGE и LANG: русская локаль даёт русский, любая другая английский).
// Неизвестное значение считается "auto".
func SetLanguage(code string) {
	switch code {
	case RU, EN:
		current.Store(code)
	default:
		current.Store(detect(os.Getenv))
	}
}

// Language возвращает действующий язык: "ru" или "en".
func Language() string { return current.Load().(string) }

// detect определяет язык по переменным окружения.
func detect(getenv func(string) string) string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		v := strings.ToLower(getenv(name))
		if v == "" || v == "c" || v == "posix" {
			continue
		}
		if strings.HasPrefix(v, "ru") {
			return RU
		}
		return EN
	}
	return EN
}

// T возвращает перевод русского текста на действующий язык.
func T(ru string) string {
	if Language() == EN {
		if en, ok := en[ru]; ok {
			return en
		}
	}
	return ru
}

// Tf переводит формат и подставляет аргументы.
func Tf(format string, args ...any) string { return fmt.Sprintf(T(format), args...) }

// Errorf переводит формат и создаёт ошибку; %w работает как в fmt.Errorf.
func Errorf(format string, args ...any) error { return fmt.Errorf(T(format), args...) }
