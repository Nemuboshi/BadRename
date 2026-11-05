package ui

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// These checks exercise MyGo scaling in a fixed logical viewport. They do
// not replace a platform window or per-monitor DPI test.
func TestNativeLayoutAtDisplayScales(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			d := newDesktop()
			tt := ui.NewTester(d.view, contentWidth, contentHeight)
			tt.SetScale(scale)
			image := tt.Image().Bounds()
			if wantW, wantH := int(math.Round(float64(contentWidth*scale))), int(math.Round(float64(contentHeight*scale))); image.Dx() != wantW || image.Dy() != wantH {
				t.Fatalf("image dimensions at scale %v: %v, want %dx%d", scale, image, wantW, wantH)
			}
			for _, name := range []string{"Whole", "Replace", "Add/Delete", "Regex", "Add", "Help"} {
				r, ok := tt.Find(name)
				if !ok || r.X < 0 || r.Y < 0 || r.X+r.W > contentWidth || r.Y+r.H > contentHeight {
					t.Fatalf("control %q outside viewport: %v found=%v", name, r, ok)
				}
			}
			for _, page := range []struct{ tab, text string }{
				{"Whole", "Resolve name conflicts automatically"},
				{"Replace", "Replace with"},
				{"Add/Delete", "Delete a range"},
				{"Regex", "Find (regular expression)"},
			} {
				if err := tt.Click(page.tab); err != nil {
					t.Fatal(err)
				}
				r, ok := tt.Find(page.text)
				if !ok || r.X < 0 || r.Y < 0 || r.X+r.W > contentWidth || r.Y+r.H > contentHeight {
					t.Fatalf("%s control %q outside viewport: %v found=%v", page.tab, page.text, r, ok)
				}
			}
		})
	}
}
