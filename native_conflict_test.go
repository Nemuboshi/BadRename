package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestNativeConflictActions(t *testing.T) {
	cases := []struct {
		name, action                  string
		all                           bool
		files                         map[string]string
		rows                          []RowView
		processed, renamed, unchanged int
	}{
		{"overwrite", "Overwrite", false, map[string]string{"photo_1.txt": "first", "photo_2.txt": "second", "photo_3.txt": "third"}, []RowView{{"photo_1.txt", "photo_1.txt", "OK"}, {"photo_2.txt", "photo_2.txt", "OK"}, {"photo_3.txt", "photo_3.txt", "OK"}}, 3, 3, 0},
		{"auto rename", "Auto Rename", false, map[string]string{"photo_1.txt": "first", "photo_2.txt": "existing2", "photo_3.txt": "existing3", "photo_2 (2).txt": "second", "photo_3 (2).txt": "third"}, []RowView{{"photo_1.txt", "photo_1.txt", "OK"}, {"photo_2 (2).txt", "photo_2.txt", "OK"}, {"photo_3 (2).txt", "photo_3.txt", "OK"}}, 3, 3, 0},
		{"overwrite all", "Overwrite", true, map[string]string{"photo_1.txt": "first", "photo_2.txt": "second", "photo_3.txt": "third"}, []RowView{{"photo_1.txt", "photo_1.txt", "OK"}, {"photo_2.txt", "photo_2.txt", "OK"}, {"photo_3.txt", "photo_3.txt", "OK"}}, 3, 3, 0},
		{"auto rename all", "Auto Rename", true, map[string]string{"photo_1.txt": "first", "photo_2.txt": "existing2", "photo_3.txt": "existing3", "photo_2 (2).txt": "second", "photo_3 (2).txt": "third"}, []RowView{{"photo_1.txt", "photo_1.txt", "OK"}, {"photo_2 (2).txt", "photo_2.txt", "OK"}, {"photo_3 (2).txt", "photo_3.txt", "OK"}}, 3, 3, 0},
		{"skip all", "Skip", true, map[string]string{"photo_1.txt": "first", "second.txt": "second", "third.txt": "third", "photo_2.txt": "existing2", "photo_3.txt": "existing3"}, []RowView{{"photo_1.txt", "photo_1.txt", "OK"}, {"second.txt", "photo_2.txt", "Skipped"}, {"third.txt", "photo_3.txt", "Skipped"}}, 3, 1, 2},
		{"cancel", "Cancel", false, map[string]string{"photo_1.txt": "first", "second.txt": "second", "third.txt": "third", "photo_2.txt": "existing2", "photo_3.txt": "existing3"}, []RowView{{"photo_1.txt", "photo_1.txt", "OK"}, {"second.txt", "photo_2.txt", ""}, {"third.txt", "photo_3.txt", ""}}, 2, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range map[string]string{"first.txt": "first", "second.txt": "second", "third.txt": "third", "photo_2.txt": "existing2", "photo_3.txt": "existing3"} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			d := newDesktop()
			d.app.AddPaths([]string{filepath.Join(dir, "first.txt"), filepath.Join(dir, "second.txt"), filepath.Join(dir, "third.txt")})
			d.rules.pattern = "photo_#"
			d.refresh()
			updates := make(chan func(), 1)
			d.post = func(f func()) { updates <- f }
			tt := ui.NewTester(d.view, 906, 500)
			update := func() {
				t.Helper()
				select {
				case f := <-updates:
					f()
					tt.Frame()
				case <-time.After(5 * time.Second):
					t.Fatal("worker did not post result")
				}
			}
			click := func(label string) {
				t.Helper()
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
			}
			click("Start Rename")
			update()
			if !d.conflictOpen || d.conflict != "photo_2.txt" || !tt.HasText("Name conflict") {
				t.Fatalf("first conflict: open=%v target=%q texts=%v", d.conflictOpen, d.conflict, tt.Texts())
			}
			if tc.all {
				click("Apply to all")
				if !d.applyAll {
					t.Fatal("Apply to all was not selected")
				}
			}
			choose := func() {
				t.Helper()
				if tc.action != "Cancel" {
					click(tc.action)
					return
				}
				// The main window also has a Cancel button. Select the dialog button,
				// immediately to the right of Auto Rename, without invoking desktop Quit.
				r, ok := tt.Find("Auto Rename")
				if !ok {
					t.Fatal("missing Auto Rename button")
				}
				tt.ClickAt(r.X+r.W+25, r.Y+r.H/2)
			}
			choose()
			update()
			if !tc.all && tc.action != "Cancel" {
				if !d.conflictOpen || d.conflict != "photo_3.txt" || !tt.HasText("Name conflict") {
					t.Fatalf("second conflict: open=%v target=%q", d.conflictOpen, d.conflict)
				}
				choose()
				update()
			}
			if d.busy || d.conflictOpen || !d.summaryOpen || !tt.HasText("Summary") {
				t.Fatalf("final dialog: busy=%v conflict=%v summary=%v texts=%v", d.busy, d.conflictOpen, d.summaryOpen, tt.Texts())
			}
			wantSummary := fmt.Sprintf("Files processed: %d\nRenamed:       %d\nFailed:        0\nUnchanged:     %d", tc.processed, tc.renamed, tc.unchanged)
			if d.summary != wantSummary || !tt.HasText(wantSummary) {
				t.Errorf("summary=%q, want %q", d.summary, wantSummary)
			}
			if !reflect.DeepEqual(d.rows, tc.rows) {
				t.Errorf("rows=%+v, want %+v", d.rows, tc.rows)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			got := make(map[string]string)
			for _, entry := range entries {
				data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				got[entry.Name()] = string(data)
			}
			if !reflect.DeepEqual(got, tc.files) {
				t.Errorf("files=%v, want %v", got, tc.files)
			}
		})
	}
}
