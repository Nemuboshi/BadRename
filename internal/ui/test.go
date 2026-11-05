package ui

import (
	"badrename/internal/app"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestNativeRulePagesKeepValuesAndPreview(t *testing.T) {
	d := newDesktop()
	path := filepath.Join(t.TempDir(), "cat.txt")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	d.app.AddPaths([]string{path})
	d.refresh()
	tt := ui.NewTester(d.view, 906, 500)
	if err := tt.Click("Naming rule"); err != nil {
		t.Fatal(err)
	}
	tt.Type("photo_#")
	tt.Frame()
	if len(d.rows) != 1 || d.rows[0].Preview != "photo_1.txt" {
		t.Fatalf("preview: %+v", d.rows)
	}
	if err := tt.Click("Replace"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Find"); err != nil {
		t.Fatal(err)
	}
	tt.Type("cat")
	if err := tt.Click("Replace with"); err != nil {
		t.Fatal(err)
	}
	tt.Type("dog")
	tt.Frame()
	if got := d.rows[0].Preview; got != "dog.txt" {
		t.Fatalf("replace preview: %q form=%+v", got, d.rules.form())
	}
	if err := tt.Click("Whole"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := d.rules.form().Pattern; got != "photo_#" {
		t.Fatalf("whole pattern lost: %q", got)
	}
}

// Test the UI/controller boundary with real disposable files. The queued
// callbacks are applied on the tester's goroutine, like Window.Update does.
func TestNativeRenameAndResolveConflict(t *testing.T) {
	d := newDesktop()
	dir := t.TempDir()
	for _, name := range []string{"cat.txt", "dog.txt", "photo_2.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	d.openFiles = func() ([]string, error) {
		cat := filepath.Join(dir, "cat.txt")
		return []string{cat, filepath.Join(dir, "dog.txt"), cat, dir}, nil
	}
	updates := make(chan func(), 1)
	d.post = func(f func()) { updates <- f }
	applyUpdate := func() {
		t.Helper()
		select {
		case f := <-updates:
			f()
		case <-time.After(5 * time.Second):
			t.Fatal("rename worker did not publish its result")
		}
	}
	tt := ui.NewTester(d.view, 906, 500)
	if err := tt.Click("Add"); err != nil {
		t.Fatal(err)
	}
	applyUpdate()
	tt.Frame()
	if len(d.rows) != 2 || d.rows[0].Old != "cat.txt" || d.rows[1].Old != "dog.txt" {
		t.Fatalf("added rows: %+v", d.rows)
	}
	if err := tt.Click("Naming rule"); err != nil {
		t.Fatal(err)
	}
	tt.Type("photo_#")
	tt.Frame()
	if err := tt.Click("Start Rename"); err != nil {
		t.Fatal(err)
	}
	applyUpdate()
	tt.Frame()
	if !d.conflictOpen || !tt.HasText("Name conflict") {
		t.Fatalf("missing conflict: %+v", d)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "photo_1.txt")); err != nil || string(got) != "cat.txt" {
		t.Fatalf("first rename: %q %v", got, err)
	}
	if err := tt.Click("Skip"); err != nil {
		t.Fatal(err)
	}
	applyUpdate()
	tt.Frame()
	if !d.summaryOpen || !tt.HasText("Summary") {
		t.Fatalf("missing summary: %+v", d)
	}
	for _, name := range []string{"dog.txt", "photo_2.txt"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != name {
			t.Fatalf("conflict damaged %s: %q %v", name, got, err)
		}
	}
}

func TestNativeListSelectMoveAndRemove(t *testing.T) {
	d := newDesktop()
	dir := t.TempDir()
	for _, name := range []string{"first.txt", "second.txt", "third.txt"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		d.app.AddPaths([]string{path})
	}
	d.refresh()
	tt := ui.NewTester(d.view, 906, 500)
	if err := tt.Click("second.txt"); err != nil {
		t.Fatal(err)
	}
	if d.list.selected != 1 {
		t.Fatalf("selected row: %d", d.list.selected)
	}
	if err := tt.Click("Move down"); err != nil {
		t.Fatal(err)
	}
	if d.list.selected != 2 || d.rows[2].Old != "second.txt" {
		t.Fatalf("move: selected=%d rows=%+v", d.list.selected, d.rows)
	}
	if err := tt.Click("Remove"); err != nil {
		t.Fatal(err)
	}
	if len(d.rows) != 2 || d.list.selected != 1 || d.rows[1].Old != "third.txt" {
		t.Fatalf("remove: selected=%d rows=%+v", d.list.selected, d.rows)
	}
	if err := tt.Click("Remove All"); err != nil {
		t.Fatal(err)
	}
	if len(d.rows) != 0 || d.list.selected != -1 {
		t.Fatalf("remove all: selected=%d rows=%+v", d.list.selected, d.rows)
	}
}

func TestNativeFileDropFilteringAndModalGuard(t *testing.T) {
	d := newDesktop()
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.txt"), filepath.Join(dir, "second.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	d.rules.pattern = "item_#"
	d.addDroppedFiles([]string{first, dir, first, filepath.Join(dir, "missing")})
	if len(d.rows) != 1 || d.rows[0].Preview != "item_1.txt" {
		t.Fatalf("drop result: %+v", d.rows)
	}
	for _, open := range []*bool{&d.busy, &d.conflictOpen, &d.summaryOpen, &d.helpOpen} {
		*open = true
		d.addDroppedFiles([]string{second})
		*open = false
		if len(d.rows) != 1 {
			t.Fatalf("drop accepted while busy or modal: %+v", d.rows)
		}
	}
	d.addDroppedFiles([]string{second})
	if len(d.rows) != 2 || d.rows[1].Preview != "item_2.txt" {
		t.Fatalf("second drop result: %+v", d.rows)
	}
}

func TestNativeFileDialogCancelAndError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		d := newDesktop()
		updates := make(chan func(), 1)
		d.post = func(f func()) { updates <- f }
		d.openFiles = func() ([]string, error) {
			if fail {
				return nil, errors.New("dialog unavailable")
			}
			return nil, nil
		}
		tt := ui.NewTester(d.view, 906, 500)
		if err := tt.Click("Add"); err != nil {
			t.Fatal(err)
		}
		select {
		case f := <-updates:
			f()
		case <-time.After(5 * time.Second):
			t.Fatal("file picker did not publish its result")
		}
		tt.Frame()
		if d.busy || len(d.rows) != 0 || d.summaryOpen != fail {
			t.Fatalf("fail=%v: busy=%v rows=%v summaryOpen=%v", fail, d.busy, d.rows, d.summaryOpen)
		}
		if fail && !tt.HasText("dialog unavailable") {
			t.Fatalf("dialog error not shown: %v", tt.Texts())
		}
	}
}

func TestNativeRegexValidation(t *testing.T) {
	d := newDesktop()
	path := filepath.Join(t.TempDir(), "example.txt")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	d.app.AddPaths([]string{path})
	d.refresh()
	tt := ui.NewTester(d.view, 906, 500)
	if err := tt.Click("Regex"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Find (regular expression)"); err != nil {
		t.Fatal(err)
	}
	tt.Type("[")
	tt.Frame()
	if !tt.HasText("Invalid expression:") || d.rows[0].Preview != "" {
		t.Fatalf("invalid regex: rows=%+v, texts=%v", d.rows, tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("example")
	if err := tt.Click("Replace with"); err != nil {
		t.Fatal(err)
	}
	tt.Type("renamed")
	tt.Frame()
	if tt.HasText("Invalid expression:") || d.rows[0].Preview != "renamed.txt" {
		t.Fatalf("corrected regex: rows=%+v, texts=%v", d.rows, tt.Texts())
	}
}

func TestNativeTabsKeyboardNavigation(t *testing.T) {
	p := newRulePanel()
	tt := ui.NewTester(p.view, 340, 460)
	if err := tt.Click("Whole"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	if p.tab != 1 {
		t.Fatalf("right selected tab %d", p.tab)
	}
	tt.Key(0, ui.KeyEnd)
	if p.tab != 3 {
		t.Fatalf("end selected tab %d", p.tab)
	}
	tt.Key(0, ui.KeyHome)
	if p.tab != 0 {
		t.Fatalf("home selected tab %d", p.tab)
	}
}

func TestNativeNumberSpinner(t *testing.T) {
	p := newRulePanel()
	tt := ui.NewTester(p.view, 340, 460)
	if err := tt.Click("Increase Start at"); err != nil {
		t.Fatal(err)
	}
	if p.form().Start != 2 {
		t.Fatalf("start: %d", p.form().Start)
	}
	if err := tt.Click("Decrease Digits"); err != nil {
		t.Fatal(err)
	}
	if p.form().Width != 1 {
		t.Fatalf("digits lower bound: %d", p.form().Width)
	}
	if err := tt.Click("Start at"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("10000000000000000")
	if got := p.form().Start; got != 10000000000000000 {
		t.Fatalf("large start truncated: %d", got)
	}
}

func TestNativeIntegerBoundsAndSpinner(t *testing.T) {
	p := newRulePanel()
	tt := ui.NewTester(p.view, 340, 460)
	set := func(label, value string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type(value)
		tt.Frame()
	}
	set("Start at", "9007199254740993")
	set("Step", "-9007199254740993")
	set("Start at", "9007199254740993.0")
	if p.form().Start != 9007199254740993 || p.form().Step != -9007199254740993 {
		t.Fatalf("lost precision: %+v", p.form())
	}
	set("Start at", "9007199254740993e0")
	if p.form().Start != 9007199254740993 {
		t.Fatalf("exponent lost precision: %d", p.form().Start)
	}
	if err := tt.Click("Increase Start at"); err != nil {
		t.Fatal(err)
	}
	if p.form().Start != 9007199254740994 {
		t.Fatalf("large increment: %d", p.form().Start)
	}
	set("Start at", "9223372036854775807")
	if err := tt.Click("Increase Start at"); err != nil {
		t.Fatal(err)
	}
	if p.form().Start != 9223372036854775807 {
		t.Fatalf("maximum wrapped: %d", p.form().Start)
	}
	set("Step", "-9223372036854775808")
	if err := tt.Click("Decrease Step"); err != nil {
		t.Fatal(err)
	}
	if p.form().Step != -9223372036854775808 {
		t.Fatalf("minimum wrapped: %d", p.form().Step)
	}
	if err := tt.Click("Increase Step"); err != nil {
		t.Fatal(err)
	}
	if p.form().Step != -9223372036854775807 {
		t.Fatalf("minimum increment: %d", p.form().Step)
	}
}

func TestNativeInvalidNumbersBlockRename(t *testing.T) {
	d := newDesktop()
	d.rules.pattern = "new_#"
	d.post = func(func()) { t.Error("invalid input started a worker") }
	tt := ui.NewTester(d.view, 906, 500)
	set := func(label, value string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type(value)
		tt.Frame()
	}
	set("Start at", "0")
	set("Start at", "9223372036854775808")
	if !tt.HasText("Start at is outside the supported integer range.") {
		t.Fatalf("missing error: %v", tt.Texts())
	}
	path := filepath.Join(t.TempDir(), "cat.txt")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	d.addDroppedFiles([]string{path})
	tt.Frame()
	if len(d.rows) != 1 || d.rows[0].Preview != "" {
		t.Fatalf("invalid preview: %+v", d.rows)
	}
	_ = tt.Click("Start Rename")
	d.runRename("")
	if d.busy {
		t.Fatal("invalid input started rename")
	}
	set("Start at", "0")
	if d.rows[0].Preview != "new_0.txt" {
		t.Fatalf("preview did not recover: %+v", d.rows)
	}
	set("Start at", "9007199254740993")
	if d.rows[0].Preview != "new_9007199254740993.txt" {
		t.Fatalf("large preview: %+v", d.rows)
	}
	set("Step", "-9223372036854775809")
	if d.rows[0].Preview != "" || !tt.HasText("Step is outside the supported integer range.") {
		t.Fatalf("step error: %+v %v", d.rows, tt.Texts())
	}
	set("Step", "1")
	if err := tt.Click("Add/Delete"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Delete a range"); err != nil {
		t.Fatal(err)
	}
	set("Delete count", "-0.5")
	if d.rows[0].Preview != "" || !tt.HasText("Delete count cannot be negative.") {
		t.Fatal("negative fraction accepted")
	}
	set("Delete count", "-2")
	if d.rows[0].Preview != "" || !tt.HasText("Delete count cannot be negative.") {
		t.Fatalf("negative count: %+v %v", d.rows, tt.Texts())
	}
	_ = tt.Click("Start Rename")
	d.runRename("")
	if d.busy {
		t.Fatal("negative count started rename")
	}
	if err := tt.Click("Delete a range"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if d.rows[0].Preview == "" {
		t.Fatal("disabled range still blocks preview")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("original file changed: %v", err)
	}
}

func TestConflictDialogCannotBeDismissedByBackdrop(t *testing.T) {
	d := newDesktop()
	d.conflict = "exists.txt"
	d.conflictOpen = true
	tt := ui.NewTester(d.view, 906, 500)
	tt.ClickAt(5, 5)
	if !d.conflictOpen || !tt.HasText("Name conflict") {
		t.Fatal("backdrop dismissed an unresolved conflict")
	}
}

func TestInformationalDialogsRequireButton(t *testing.T) {
	for _, help := range []bool{false, true} {
		d := newDesktop()
		d.summaryOpen, d.helpOpen = !help, help
		tt := ui.NewTester(d.view, 906, 500)
		tt.ClickAt(5, 5)
		if help && !d.helpOpen || !help && !d.summaryOpen {
			t.Fatalf("dialog dismissed on backdrop, help=%v", help)
		}
		button := "OK"
		if help {
			button = "Close"
		}
		if err := tt.Click(button); err != nil {
			t.Fatal(err)
		}
		if d.summaryOpen || d.helpOpen {
			t.Fatalf("dialog remained open after %s", button)
		}
	}
}

func TestNativeBusyDoesNotOpenHelp(t *testing.T) {
	d := newDesktop()
	d.busy = true
	tt := ui.NewTester(d.view, 906, 500)
	_ = tt.Click("Help")
	if d.helpOpen {
		t.Fatal("help opened during background work")
	}
}

func TestNativeBusyListDoesNotChangeSelection(t *testing.T) {
	panel := newListPanel()
	panel.selected = 0
	panel.busy = true
	rows := []app.RowView{{Old: "a"}, {Old: "b"}}
	called := false
	tt := ui.NewTester(func(c *ui.Context) {
		panel.view(c, rows, nil, nil, func() { called = true }, func(int) { called = true })
	}, 520, 420)
	_ = tt.Click("Remove All")
	_ = tt.Click("Move down")
	if called || panel.selected != 0 {
		t.Fatalf("busy list changed: called=%v selection=%d", called, panel.selected)
	}
}
