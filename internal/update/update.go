// Package update проверяет, вышла ли новая версия, запросом к GitHub Releases.
// Это единственное место, где программа выходит в интернет: проверка выключена
// по умолчанию, запускается кнопкой или (если включена) при старте значка и окна
// настроек, и ничего не отправляет, кроме заголовка User-Agent с версией.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jexme/scrawl/internal/i18n"
)

// Repo — репозиторий с релизами.
const Repo = "jexme/scrawl"

// apiURL — адрес последнего релиза; в тестах подменяется.
var apiURL = "https://api.github.com/repos/" + Repo + "/releases/latest"

// Result — итог проверки.
type Result struct {
	Latest     string // тег последнего релиза, например "v0.2.0"
	URL        string // страница релиза
	Newer      bool   // Latest новее текущей версии
	Comparable bool   // текущую версию удалось сравнить (у сборок разработчика нет)
}

type release struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// Check запрашивает последний релиз и сравнивает его с текущей версией current.
func Check(ctx context.Context, current string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "scrawl/"+current)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf(i18n.T("cannot reach GitHub: %w"), err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return Result{}, fmt.Errorf("%s", i18n.T("there are no releases yet"))
	case resp.StatusCode != http.StatusOK:
		return Result{}, fmt.Errorf(i18n.T("GitHub replied with code %d"), resp.StatusCode)
	}
	var rel release
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&rel); err != nil {
		return Result{}, fmt.Errorf(i18n.T("unexpected reply from GitHub: %w"), err)
	}
	if rel.TagName == "" {
		return Result{}, fmt.Errorf("%s", i18n.T("the GitHub reply has no version"))
	}
	res := Result{Latest: rel.TagName, URL: rel.HTMLURL}
	if newer, ok := Newer(rel.TagName, current); ok {
		res.Newer, res.Comparable = newer, true
	}
	return res, nil
}

// version — разобранная версия vX.Y.Z[-предварительная].
type version struct {
	nums [3]int
	pre  string
}

func parse(v string) (version, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "+") // метаданные сборки не важны
	core, pre, _ := strings.Cut(v, "-")
	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return version{}, false
	}
	var out version
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return version{}, false
		}
		out.nums[i] = n
	}
	out.pre = pre
	return out, true
}

// Newer сообщает, что версия latest новее current. Второе значение false, если
// одну из версий нельзя разобрать (например, сборка разработчика «dev-1bff227»).
// Предварительная версия (v1.0.0-rc1) старше выпуска (v1.0.0) с теми же числами.
func Newer(latest, current string) (newer, ok bool) {
	a, okA := parse(latest)
	b, okB := parse(current)
	if !okA || !okB {
		return false, false
	}
	for i := range a.nums {
		if a.nums[i] != b.nums[i] {
			return a.nums[i] > b.nums[i], true
		}
	}
	switch {
	case a.pre == b.pre:
		return false, true
	case a.pre == "":
		return true, true // выпуск новее своей предварительной версии
	case b.pre == "":
		return false, true
	}
	return a.pre > b.pre, true
}

// Message описывает итог проверки словами для окна и уведомления.
func (r Result) Message() string {
	switch {
	case r.Newer:
		return fmt.Sprintf(i18n.T("A new version %s is available: %s"), r.Latest, r.URL)
	case r.Comparable:
		return fmt.Sprintf(i18n.T("You have the latest version (%s)"), r.Latest)
	default:
		return fmt.Sprintf(i18n.T("Latest version on GitHub: %s (you run a developer build, cannot compare)"), r.Latest)
	}
}
