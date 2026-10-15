package collection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddListClear(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
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
