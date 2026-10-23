package annotate

import (
	"testing"

	"gioui.org/f32"
)

// A long freehand stroke must not be dropped just because its first two points are close
// together (a slow start of the stroke): that made the marker vanish on release.
func TestFreehandStrokeIsKept(t *testing.T) {
	long := []f32.Point{f32.Pt(100, 100), f32.Pt(101, 100), f32.Pt(140, 120), f32.Pt(220, 130), f32.Pt(300, 90)}
	for _, tool := range []Tool{ToolPen, ToolMarker} {
		if (Shape{Tool: tool, Pts: long}).Degenerate() {
			t.Errorf("tool %v: a long stroke with a slow start is dropped", tool)
		}
		if (Shape{Tool: tool, Pts: nil}).Degenerate() != true {
			t.Errorf("tool %v: a stroke without points must be dropped", tool)
		}
	}
}

// A click with the marker, without moving, leaves no mark.
func TestMarkerClickLeavesNothing(t *testing.T) {
	if !(Shape{Tool: ToolMarker, Pts: []f32.Point{f32.Pt(10, 10)}}).Degenerate() {
		t.Error("a marker click without moving must be dropped")
	}
}

// Shapes defined by two points are dropped when they are too small.
func TestTinyTwoPointShapesAreDropped(t *testing.T) {
	tiny := []f32.Point{f32.Pt(100, 100), f32.Pt(101, 101)}
	for _, tool := range []Tool{ToolLine, ToolArrow, ToolRect} {
		if !(Shape{Tool: tool, Pts: tiny}).Degenerate() {
			t.Errorf("tool %v: a tiny shape is kept", tool)
		}
	}
	big := []f32.Point{f32.Pt(100, 100), f32.Pt(180, 140)}
	for _, tool := range []Tool{ToolLine, ToolArrow, ToolRect, ToolMarker, ToolPen} {
		if (Shape{Tool: tool, Pts: big}).Degenerate() {
			t.Errorf("tool %v: a normal shape is dropped", tool)
		}
	}
}
