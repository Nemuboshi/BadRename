package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestNumberSyntaxValidation(t *testing.T) {
	for _, s := range []string{"", "abc", "1e", "NaN", "Inf", "-"} {
		t.Run(s, func(t *testing.T) {
			_, _, invalid := parseNumber(s)
			if !invalid {
				t.Fatalf("accepted %q", s)
			}
		})
	}
	for _, s := range []string{"0", "1.5", "2e2", "-5"} {
		t.Run("valid_"+s, func(t *testing.T) {
			_, out, invalid := parseNumber(s)
			if out || invalid {
				t.Fatalf("rejected %q", s)
			}
		})
	}
}

func TestNativeInvalidNumberBlocksRename(t *testing.T) {
	d := newDesktop()
	d.rules.pattern = "new_#"
	d.post = func(func()) { t.Error("unexpected worker") }
	path := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	d.addDroppedFiles([]string{path})
	tt := ui.NewTester(d.view, 906, 500)
	if err := tt.Click("Start at"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("abc")
	tt.Frame()
	if !tt.HasText("Enter a valid number.") || d.rows[0].Preview != "" {
		t.Fatalf("invalid number not rejected: %v %+v", tt.Texts(), d.rows)
	}
	d.runRename("")
	if d.busy {
		t.Fatal("rename started")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
