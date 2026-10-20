package update

import (
	"context"
	"github.com/jexme/scrawl/internal/testenv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		newer, ok       bool
	}{
		{"v0.2.0", "v0.1.0", true, true},
		{"v0.1.0", "v0.1.0", false, true},
		{"v0.1.0", "v0.2.0", false, true},
		{"v1.10.0", "v1.9.9", true, true},
		{"v1.0.0", "v1.0.0-rc1", true, true},
		{"v1.0.0-rc1", "v1.0.0", false, true},
		{"v1.0.0-rc2", "v1.0.0-rc1", true, true},
		{"1.2", "v1.1.9", true, true},
		{"v2.0.0", "v1.9.9+build5", true, true},
		{"v0.2.0", "dev-1bff227", false, false},
		{"v0.2.0", "1bff227-dirty", false, false},
		{"", "v0.1.0", false, false},
		{"v1.x.0", "v1.0.0", false, false},
	}
	for _, c := range cases {
		newer, ok := Newer(c.latest, c.current)
		if newer != c.newer || ok != c.ok {
			t.Errorf("Newer(%q, %q) = %v, %v; ожидали %v, %v", c.latest, c.current, newer, ok, c.newer, c.ok)
		}
	}
}

func serve(t *testing.T, status int, body string, check func(*http.Request)) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	old := apiURL
	apiURL = srv.URL
	t.Cleanup(func() { apiURL = old; srv.Close() })
}

func TestCheck(t *testing.T) {
	var ua string
	serve(t, 200, `{"tag_name":"v0.3.0","html_url":"https://github.com/jexme/scrawl/releases/tag/v0.3.0"}`,
		func(r *http.Request) { ua = r.Header.Get("User-Agent") })
	res, err := Check(context.Background(), "v0.2.0")
	if err != nil || res.Latest != "v0.3.0" || !res.Newer || !res.Comparable || !strings.Contains(res.URL, "v0.3.0") {
		t.Fatalf("итог %+v, ошибка %v", res, err)
	}
	if ua != "scrawl/v0.2.0" {
		t.Errorf("User-Agent: %q (в запросе не должно быть ничего лишнего)", ua)
	}

	res, err = Check(context.Background(), "v0.3.0")
	if err != nil || res.Newer || !res.Comparable {
		t.Errorf("та же версия: %+v %v", res, err)
	}
	res, err = Check(context.Background(), "dev-abc1234")
	if err != nil || res.Newer || res.Comparable || res.Latest != "v0.3.0" {
		t.Errorf("сборка разработчика: %+v %v", res, err)
	}
}

func TestCheckErrors(t *testing.T) {
	for _, c := range []struct {
		status int
		body   string
		want   string
	}{
		{404, "", "релизов пока нет"},
		{500, "", "500"},
		{200, "не json", "непонятный"},
		{200, `{}`, "нет версии"},
	} {
		serve(t, c.status, c.body, nil)
		if _, err := Check(context.Background(), "v0.1.0"); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%d %q: ошибка %v, ожидали «%s»", c.status, c.body, err, c.want)
		}
	}
	// Сервер недоступен.
	apiURL = "http://127.0.0.1:1/"
	if _, err := Check(context.Background(), "v0.1.0"); err == nil || !strings.Contains(err.Error(), "нет связи") {
		t.Errorf("нет связи: %v", err)
	}
}

func TestMessage(t *testing.T) {
	for _, c := range []struct {
		r    Result
		want string
	}{
		{Result{Latest: "v0.3.0", URL: "https://x/y", Newer: true, Comparable: true}, "Доступна новая версия v0.3.0: https://x/y"},
		{Result{Latest: "v0.3.0", Comparable: true}, "У вас последняя версия (v0.3.0)"},
		{Result{Latest: "v0.3.0"}, "сборка разработчика"},
	} {
		if got := c.r.Message(); !strings.Contains(got, c.want) {
			t.Errorf("%+v: %q, ожидали «%s»", c.r, got, c.want)
		}
	}
}

func TestNotifiedOncePerVersion(t *testing.T) {
	testenv.Cache(t, t.TempDir())
	if AlreadyNotified("v0.3.0") {
		t.Fatal("в чистом кэше ничего не сообщали")
	}
	if err := MarkNotified("v0.3.0"); err != nil {
		t.Fatal(err)
	}
	if !AlreadyNotified("v0.3.0") || AlreadyNotified("v0.4.0") {
		t.Error("о v0.3.0 сообщили, о v0.4.0 ещё нет")
	}
}
