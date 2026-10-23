package editor

import (
	"testing"

	"github.com/jexme/scrawl/internal/testenv"
)

// The hint is shown for the first few launches only.
func TestNextHint(t *testing.T) {
	testenv.Cache(t, t.TempDir())
	for i := 1; i <= hintLaunches; i++ {
		if !NextHint() {
			t.Fatalf("launch %d must show the hint", i)
		}
	}
	for i := 0; i < 3; i++ {
		if NextHint() {
			t.Fatal("after the first launches the hint stays hidden")
		}
	}
}
