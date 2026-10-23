package editor

import (
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/jexme/scrawl/internal/annotate"
)

// hintLaunches is how many first launches show the hint on the overlay.
const hintLaunches = 5

func hintCountPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "scrawl", "hint-count")
}

// NextHint counts this launch of the overlay and says whether it should show the hint "Drag to
// select an area, Enter copies and saves, Esc cancels": a new user gets it for the first few
// launches, then it goes away for good. A failure to write the counter only means the hint
// may be shown once more.
func NextHint() bool {
	path := hintCountPath()
	n := 0
	if data, err := os.ReadFile(path); err == nil {
		n, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}
	if n >= hintLaunches {
		return false
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) == nil {
		os.WriteFile(path, []byte(strconv.Itoa(n+1)+"\n"), 0o600)
	}
	return true
}

// layoutHint draws a hint near the top of the window: the first launches say what to do next.
func (m *monitor) layoutHint(gtx layout.Context, win image.Point, txt string) {
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: iconColor}.Add(gtx.Ops)
	mat := rec.Stop()
	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(1<<12, 1<<12)}
	rec = op.Record(gtx.Ops)
	dims := widget.Label{MaxLines: 1}.Layout(lgtx, m.shaper, annotate.Font, 17, txt, mat)
	label := rec.Stop()

	padX, padY := gtx.Dp(18), gtx.Dp(11)
	w, h := dims.Size.X+2*padX, dims.Size.Y+2*padY
	x := clampInt((win.X-w)/2, gtx.Dp(8), max(win.X-w-gtx.Dp(8), gtx.Dp(8)))
	y := win.Y / 8
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	pill := clip.UniformRRect(image.Rect(0, 0, w, h), min(h/2, gtx.Dp(14)))
	paint.FillShape(gtx.Ops, barBG, pill.Op(gtx.Ops))
	defer op.Offset(image.Pt(padX, padY)).Push(gtx.Ops).Pop()
	label.Add(gtx.Ops)
}
