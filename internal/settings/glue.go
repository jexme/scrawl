package settings

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Glue — настройки склейки «было/стало».
type Glue struct {
	// Подписи над картинками. Пусто: слова по умолчанию.
	Before string `json:"before,omitempty"` // первая из двух картинок
	After  string `json:"after,omitempty"`  // вторая из двух картинок
	Step   string `json:"step,omitempty"`   // слово перед номером, если картинок больше двух
}

const maxGlueLabel = 30

// Слова подписей склейки по умолчанию.
const (
	DefaultGlueBefore = "Было"
	DefaultGlueAfter  = "Стало"
	DefaultGlueStep   = "Шаг"
)

// ValidateGlueLabel проверяет подпись: пусто (по умолчанию) или короткий текст
// без управляющих символов.
func ValidateGlueLabel(s string) error {
	if utf8.RuneCountInString(s) > maxGlueLabel {
		return fmt.Errorf("не длиннее %d знаков", maxGlueLabel)
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return errors.New("нельзя использовать управляющие символы")
		}
	}
	return nil
}

func (g *Glue) normalize() {
	for _, p := range []*string{&g.Before, &g.After, &g.Step} {
		if ValidateGlueLabel(*p) != nil {
			*p = ""
		}
	}
}

// Labels возвращает подписи для n картинок: «Было» и «Стало» для двух, «Шаг N»
// для остальных (слова из настроек или по умолчанию).
func (g Glue) Labels(n int) []string {
	pick := func(custom, def string) string {
		if custom == "" {
			return def
		}
		return custom
	}
	if n == 2 {
		return []string{pick(g.Before, DefaultGlueBefore), pick(g.After, DefaultGlueAfter)}
	}
	step := pick(g.Step, DefaultGlueStep)
	labels := make([]string, n)
	for i := range labels {
		labels[i] = fmt.Sprintf("%s %d", step, i+1)
	}
	return labels
}
