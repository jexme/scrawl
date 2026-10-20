package collection

import (
	"github.com/jexme/scrawl/internal/testenv"
	"os"
	"path/filepath"
	"testing"
)

func TestAddListClear(t *testing.T) {
	testenv.Cache(t, t.TempDir())
	if Count() != 0 {
		t.Fatal("подборка должна быть пустой")
	}
	var added []string
	for _, b := range []string{"one", "two", "three"} {
		p, err := Add([]byte(b))
		if err != nil {
			t.Fatal(err)
		}
		added = append(added, p)
	}
	got, err := List()
	if err != nil || len(got) != 3 {
		t.Fatalf("List: %v, %v", got, err)
	}
	for i := range added {
		if got[i] != added[i] {
			t.Errorf("порядок нарушен: %d: %s != %s", i, got[i], added[i])
		}
	}
	// Посторонние файлы не трогаем.
	other := filepath.Join(Dir(), "notes.txt")
	os.WriteFile(other, []byte("x"), 0o600)
	if err := Clear(); err != nil {
		t.Fatal(err)
	}
	if Count() != 0 {
		t.Error("после Clear подборка должна быть пустой")
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("Clear не должен удалять посторонние файлы")
	}
}

// Быстрые добавления подряд не должны затирать друг друга, даже если часы
// (на Windows они тикают реже микросекунды) дают одинаковое время.
func TestAddQuicklyKeepsEveryShotInOrder(t *testing.T) {
	testenv.Cache(t, t.TempDir())
	const n = 200
	added := make([]string, 0, n)
	for i := 0; i < n; i++ {
		p, err := Add([]byte{byte(i)})
		if err != nil {
			t.Fatal(err)
		}
		added = append(added, p)
	}
	got, err := List()
	if err != nil || len(got) != n {
		t.Fatalf("List: %d файлов из %d, %v", len(got), n, err)
	}
	for i := range added {
		if got[i] != added[i] {
			t.Fatalf("порядок нарушен: %d: %s != %s", i, got[i], added[i])
		}
	}
}
