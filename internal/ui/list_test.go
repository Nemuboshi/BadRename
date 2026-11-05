package ui

import (
	"badrename/internal/app"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestNativeTableResizePreservesPreviewWidth(t *testing.T) {
	panel := newListPanel()
	rows := []app.RowView{{Old: "original.txt", Preview: "preview.txt", Result: "OK"}}
	tt := ui.NewTester(func(c *ui.Context) { panel.view(c, rows, nil, nil, nil, nil) }, 520, 420)
	tt.Frame()
	preview, ok := tt.Find("Preview")
	if !ok {
		t.Fatal("missing Preview header")
	}
	result, ok := tt.Find("Result")
	if !ok {
		t.Fatal("missing Result header")
	}
	width := result.X - preview.X
	x, y := preview.X-12, preview.Y+preview.H/2
	tt.Press(x, y)
	tt.Move(x+150, y)
	tt.Release(x+150, y)
	if panel.table.Columns.Widths["Original name"] <= 0 {
		t.Fatal("drag did not resize Original name")
	}
	beforeScroll, ok := tt.Find("Preview")
	if !ok {
		t.Fatal("Preview header missing after drag")
	}
	tt.Scroll(x, y+60, 150, 0)
	preview, ok = tt.Find("Preview")
	if !ok {
		t.Fatal("Preview header missing after scroll")
	}
	if preview.X >= beforeScroll.X {
		t.Fatalf("table did not scroll horizontally: before %v, after %v", beforeScroll.X, preview.X)
	}
	result, ok = tt.Find("Result")
	if !ok {
		t.Fatal("Result header missing after scroll")
	}
	if got := result.X - preview.X; got != width {
		t.Fatalf("preview width changed from %v to %v", width, got)
	}
}
