// Package fonts assembles fonts for text: the built-in Go fonts plus, for a Chinese,
// Japanese or Korean interface, a system font with ideographs. Our own CJK fonts
// are not embedded in the program: they take tens of megabytes, and a desktop
// with such languages almost always has a system font.
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

// Languages that need a font with ideographs or hangul, and their fontconfig tags.
var cjkTags = map[string]string{"zh": "zh-cn", "ja": "ja", "ko": "ko"}

// Known paths, in case fontconfig (fc-match) is unavailable.
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

// Collection returns fonts for the language lang: the built-in Go ones and, if the
// language requires it and a system font is found, the font with ideographs as a fallback.
// The result is cached: the font file is read once per launch.
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

// Available reports whether a system font is found for the language (always true
// for languages without special requirements).
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

// loadSystem finds a sans-serif font for the language and parses it.
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

// candidates lists fonts in decreasing order of suitability: first the fontconfig choice
// for the needed language, then the known paths.
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
	out = append(out, windowsCandidates(tag)...)
	for _, p := range knownFiles {
		out = append(out, candidate{p, 0})
	}
	return out
}

// Preload loads the system font for the language in the background so that its parsing (about
// 80 ms for CJK) runs while the program is still receiving screen frames, not after.
func Preload(lang string) {
	if _, ok := cjkTags[lang]; ok {
		go Available(lang)
	}
}
