package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestTranslate(t *testing.T) {
	defer SetLanguage(RU)

	SetLanguage(RU)
	if got := T("Закрыть"); got != "Закрыть" {
		t.Errorf("по-русски текст не меняется: %q", got)
	}
	SetLanguage(EN)
	if got := T("Закрыть"); got != "Close" {
		t.Errorf("по-английски: %q", got)
	}
	if got := T("текст, которого нет в каталоге"); got != "текст, которого нет в каталоге" {
		t.Errorf("непереведённый текст показывается как есть: %q", got)
	}
	if got := Tf("Подборка: %d", 3); got != "Collection: 3" {
		t.Errorf("Tf: %q", got)
	}
	err := Errorf("рендер: %w", errTest)
	if err.Error() != "render: boom" {
		t.Errorf("Errorf: %q", err)
	}
	if !errorsIs(err, errTest) {
		t.Error("Errorf должен оборачивать ошибку (%w)")
	}
	SetLanguage(RU)
	if got := Tf("Подборка: %d", 3); got != "Подборка: 3" {
		t.Errorf("Tf по-русски: %q", got)
	}
	if Language() != RU {
		t.Errorf("Language: %q", Language())
	}
}

type testErr string

func (e testErr) Error() string { return string(e) }

const errTest = testErr("boom")

func errorsIs(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func TestAutoDetect(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"русская локаль", map[string]string{"LANG": "ru_RU.UTF-8"}, RU},
		{"английская", map[string]string{"LANG": "en_US.UTF-8"}, EN},
		{"другая", map[string]string{"LANG": "de_DE.UTF-8"}, EN},
		{"LC_ALL главнее LANG", map[string]string{"LC_ALL": "en_GB.UTF-8", "LANG": "ru_RU.UTF-8"}, EN},
		{"C пропускается", map[string]string{"LC_ALL": "C", "LANG": "ru_RU.UTF-8"}, RU},
		{"LANGUAGE", map[string]string{"LANGUAGE": "ru:en"}, RU},
		{"ничего не задано", map[string]string{}, EN},
	}
	for _, c := range cases {
		if got := detect(env(c.env)); got != c.want {
			t.Errorf("%s: %q, ожидали %q", c.name, got, c.want)
		}
	}

	defer SetLanguage(RU)
	t.Setenv("LC_ALL", "ru_RU.UTF-8")
	SetLanguage("auto")
	if Language() != RU {
		t.Errorf("auto при русской локали: %q", Language())
	}
	t.Setenv("LC_ALL", "fr_FR.UTF-8")
	SetLanguage("что-то странное") // неизвестное значение считается auto
	if Language() != EN {
		t.Errorf("неизвестный код при французской локали: %q", Language())
	}
}

var verbRe = regexp.MustCompile(`%[-+#0-9.]*[a-zA-Z%]`)

// Формат и перевод должны содержать одни и те же подстановки в том же порядке,
// иначе Sprintf напечатает мусор.
func TestVerbsMatch(t *testing.T) {
	for ru, en := range en {
		a, b := verbRe.FindAllString(ru, -1), verbRe.FindAllString(en, -1)
		if strings.Join(a, " ") != strings.Join(b, " ") {
			t.Errorf("подстановки расходятся: %q %v и %q %v", ru, a, en, b)
		}
		if strings.TrimSpace(en) == "" {
			t.Errorf("пустой перевод: %q", ru)
		}
		if strings.HasPrefix(ru, " ") != strings.HasPrefix(en, " ") || strings.HasSuffix(ru, " ") != strings.HasSuffix(en, " ") {
			t.Errorf("пробелы по краям расходятся: %q и %q", ru, en)
		}
	}
}

var cyr = regexp.MustCompile(`[А-Яа-яЁё]`)

// Тексты, которые переводятся не в месте объявления, а при показе. Они обязаны
// быть в каталоге (проверяется ниже), а в коде могут стоять без T.
var translatedAtUse = []string{
	"Было", "Стало", "Шаг",
	"Выделение", "Перо", "Линия", "Стрелка", "Рамка", "Маркер", "Размытие", "Текст",
	"Копировать и/или сохранить", "Закрыть оверлей", "Вернуть область прошлого снимка", "Добавить в подборку",
	"Склеить подборку", "Отправить подборку файлами", "Отменить последнюю аннотацию", "Только копировать", "Только сохранить",
}

// Русские литералы, которые остаются как есть: технические сообщения для разработчика.
var leftRussian = map[string]bool{
	"кадр PipeWire не пришёл вовремя": true,
	"Русский": true, // название языка пишется на нём самом
}

func TestCatalogCompleteness(t *testing.T) {
	root := filepath.Join("..", "..")
	used := map[string]bool{}
	for _, k := range translatedAtUse {
		used[k] = true
	}
	var unwrapped []string

	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "third_party" || d.Name() == ".git" || d.Name() == ".deps" || d.Name() == "capturespike") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, "internal/i18n") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		wrapped := map[*ast.BasicLit]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "i18n" && (sel.Sel.Name == "T" || sel.Sel.Name == "Tf" || sel.Sel.Name == "Errorf") && len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					key, _ := strconv.Unquote(lit.Value)
					used[key] = true
					wrapped[lit] = true
				}
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING || wrapped[lit] || !cyr.MatchString(lit.Value) {
				return true
			}
			key, _ := strconv.Unquote(lit.Value)
			if _, inCatalog := en[key]; inCatalog || leftRussian[key] {
				return true
			}
			// Внутри i18n.T(...) других вызовов (например, T(x+"...")) и подписи диагностики.
			unwrapped = append(unwrapped, fset.Position(lit.Pos()).String()+" "+lit.Value)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for key := range used {
		if _, ok := en[key]; !ok {
			t.Errorf("нет английского перевода: %q", key)
		}
	}
	for key := range en {
		if !used[key] {
			t.Errorf("лишняя запись в каталоге (текста нет в коде): %q", key)
		}
	}
	// Диагностика разработчика (-v, [debug]) намеренно не переводится.
	for _, u := range unwrapped {
		if strings.Contains(u, "[debug]") || strings.Contains(u, "готовы за") || strings.Contains(u, "получены за") || strings.Contains(u, "картинка на экране через") {
			continue
		}
		t.Errorf("русский текст без T: %s", u)
	}
}
