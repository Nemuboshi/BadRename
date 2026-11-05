package app

import (
	"os"
	"path/filepath"
	"testing"

	"badrename/internal/rename"
)

func TestPatternDoesNotInterpretInsertedName(t *testing.T) {
	for _, tc := range []struct{ pattern, want string }{{"*", "a#b.txt"}, {"*_#", "a#b_1.txt"}, {"#_*_#", "1_a#b_1.txt"}} {
		t.Run(tc.pattern, func(t *testing.T) {
			r := rename.Rule{Kind: rename.RulePattern, Pattern: rename.PatternRule{Pattern: tc.pattern, Start: 1}}
			if got := rename.Apply(r, "a#b.txt", 0); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
	a, dir := newTempApp(t, "a#b.txt")
	rows := a.SetForm(wholeForm("*_#"))
	if rows[0].Preview != "a#b_001.txt" {
		t.Fatal(rows)
	}
	a.StartRename()
	if _, err := os.Stat(filepath.Join(dir, rows[0].Preview)); err != nil {
		t.Fatal(err)
	}
}
