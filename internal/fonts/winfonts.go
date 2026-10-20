package fonts

import (
	"os"
	"path/filepath"
	"runtime"
)

// winFiles — системные шрифты Windows с иероглифами и хангылем по языкам, лучшие первыми.
var winFiles = map[string][]string{
	"zh-cn": {"msyh.ttc", "simsun.ttc", "msjh.ttc"},
	"ja":    {"YuGothR.ttc", "meiryo.ttc", "msgothic.ttc"},
	"ko":    {"malgun.ttf", "gulim.ttc"},
}

// windowsCandidates перечисляет шрифты из папки Fonts системы. На других платформах пусто.
func windowsCandidates(tag string) []candidate {
	if runtime.GOOS != "windows" {
		return nil
	}
	root := os.Getenv("WINDIR")
	if root == "" {
		root = os.Getenv("SystemRoot")
	}
	if root == "" {
		root = `C:\Windows`
	}
	var out []candidate
	for _, name := range winFiles[tag] {
		out = append(out, candidate{filepath.Join(root, "Fonts", name), 0})
	}
	return out
}
