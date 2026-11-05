package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewAndRenameWithExtremeSerial(t *testing.T) {
	app, dir := newTempApp(t, "a.txt", "b.txt")
	rows := app.SetForm(FormState{Tab: "whole", Pattern: "#", Start: math.MaxInt64 - 1, Step: 2})
	want := []string{"9223372036854775806.txt", "9223372036854775808.txt"}
	for i := range want {
		if rows[i].Preview != want[i] {
			t.Fatalf("preview %d: got %q want %q", i, rows[i].Preview, want[i])
		}
	}
	if result := app.StartRename(); result.Conflict != "" {
		t.Fatalf("unexpected conflict: %s", result.Conflict)
	}
	for _, name := range want {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("renamed file %s: %v", name, err)
		}
	}
}

func TestNegativeDeleteRangeDoesNotDuplicateName(t *testing.T) {
	app, dir := newTempApp(t, "abcdef.txt")
	rows := app.SetForm(FormState{Tab: "adddelete", Prefix: "fixed_", DeleteRange: true, DeleteStart: 4, DeleteCount: -2})
	if got := rows[0].Preview; got != "fixed_abcdef.txt" {
		t.Fatalf("preview: %q", got)
	}
	if result := app.StartRename(); result.Conflict != "" {
		t.Fatalf("unexpected conflict: %s", result.Conflict)
	}
	if _, err := os.Stat(filepath.Join(dir, "fixed_abcdef.txt")); err != nil {
		t.Fatal(err)
	}
}
