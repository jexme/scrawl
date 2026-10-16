package fonts

import (
	"testing"

	"gioui.org/font/gofont"
)

func TestNonCJKLanguagesUseOnlyGoFonts(t *testing.T) {
	for _, lang := range []string{"ru", "en", "es", "pt", "de", "fr", "auto", ""} {
		if got, want := len(Collection(lang)), len(gofont.Collection()); got != want {
			t.Errorf("%q: %d шрифтов, ожидали %d (только встроенные)", lang, got, want)
		}
		if !Available(lang) {
			t.Errorf("%q: шрифты должны быть доступны всегда", lang)
		}
	}
}

func TestCJKFallbackWhenSystemFontExists(t *testing.T) {
	for _, lang := range []string{"zh", "ja", "ko"} {
		if !Available(lang) {
			t.Logf("%s: системного шрифта нет, проверка пропущена", lang)
			continue
		}
		if got := len(Collection(lang)); got != len(gofont.Collection())+1 {
			t.Errorf("%s: %d шрифтов, ожидали встроенные и один системный", lang, got)
		}
		// Повторный вызов берёт результат из кэша.
		if len(Collection(lang)) != len(Collection(lang)) {
			t.Errorf("%s: кэш даёт разный результат", lang)
		}
	}
}

func TestCandidatesListKnownPaths(t *testing.T) {
	c := candidates("zh-cn")
	if len(c) < len(knownFiles) {
		t.Errorf("кандидатов %d, известных путей %d", len(c), len(knownFiles))
	}
}
