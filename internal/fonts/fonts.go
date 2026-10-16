// Package fonts собирает шрифты для текста: встроенные шрифты Go плюс, для китайского,
// японского и корейского интерфейса, системный шрифт с иероглифами. Свои CJK-шрифты
// в программу не встроены: они занимают десятки мегабайт, а на рабочем столе
// с такими языками системный шрифт почти всегда есть.
package fonts

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
)

// Языки, которым нужен шрифт с иероглифами или хангылем, и их метки для fontconfig.
var cjkTags = map[string]string{"zh": "zh-cn", "ja": "ja", "ko": "ko"}

// Известные пути, если fontconfig (fc-match) недоступен.
var knownFiles = []string{
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/google-noto-cjk/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/OTF/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
	"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
}

var (
	mu    sync.Mutex
	cache = map[string][]font.FontFace{}
)

// Collection возвращает шрифты для языка lang: встроенные Go и, если язык этого
// требует и системный шрифт найден, шрифт с иероглифами как запасной. Результат кэшируется:
// файл шрифта читается один раз за запуск.
func Collection(lang string) []font.FontFace {
	base := gofont.Collection()
	tag, ok := cjkTags[lang]
	if !ok {
		return base
	}
	mu.Lock()
	defer mu.Unlock()
	extra, done := cache[lang]
	if !done {
		extra = loadSystem(tag)
		cache[lang] = extra
	}
	return append(append([]font.FontFace{}, base...), extra...)
}

// Available сообщает, найден ли системный шрифт для языка (для языков без
// особых требований всегда true).
func Available(lang string) bool {
	tag, ok := cjkTags[lang]
	if !ok {
		return true
	}
	mu.Lock()
	defer mu.Unlock()
	extra, done := cache[lang]
	if !done {
		extra = loadSystem(tag)
		cache[lang] = extra
	}
	return len(extra) > 0
}

// loadSystem ищет шрифт без засечек для языка и разбирает его.
func loadSystem(tag string) []font.FontFace {
	for _, c := range candidates(tag) {
		data, err := readFile(c.path)
		if err != nil {
			continue
		}
		faces, err := opentype.ParseCollection(data)
		if err != nil || len(faces) == 0 {
			continue
		}
		if c.index >= 0 && c.index < len(faces) {
			return []font.FontFace{faces[c.index]}
		}
		return []font.FontFace{faces[0]}
	}
	return nil
}

type candidate struct {
	path  string
	index int
}

// candidates перечисляет шрифты по убыванию пригодности: сначала выбор fontconfig
// для нужного языка, потом известные пути.
func candidates(tag string) []candidate {
	var out []candidate
	if fc, err := exec.LookPath("fc-match"); err == nil {
		if res, err := exec.Command(fc, "-f", "%{file}|%{index}", "sans-serif:lang="+tag).Output(); err == nil {
			if path, idx, ok := strings.Cut(string(res), "|"); ok && path != "" {
				i, _ := strconv.Atoi(strings.TrimSpace(idx))
				out = append(out, candidate{path, i})
			}
		}
	}
	for _, p := range knownFiles {
		out = append(out, candidate{p, 0})
	}
	return out
}

// Preload в фоне загружает системный шрифт для языка, чтобы его разбор (около
// 80 мс для CJK) шёл, пока программа ещё получает кадры экрана, а не после.
func Preload(lang string) {
	if _, ok := cjkTags[lang]; ok {
		go Available(lang)
	}
}
