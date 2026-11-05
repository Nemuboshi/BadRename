package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnsafeGeneratedNamesNeverMoveFiles(t *testing.T) {
	cases := []struct {
		name string
		form FormState
	}{
		{"whole traversal", FormState{Tab: "whole", Pattern: "../escaped"}},
		{"backslash", FormState{Tab: "whole", Pattern: `..\escaped`}},
		{"replace separator", FormState{Tab: "replace", From: "a", To: "../escaped"}},
		{"prefix separator", FormState{Tab: "adddelete", Prefix: "../"}},
		{"regex separator", FormState{Tab: "regex", From: "a", To: "../escaped"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "sub")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			src := filepath.Join(dir, "a.txt")
			if err := os.WriteFile(src, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			a := NewApp()
			a.AddPaths([]string{src})
			rows := a.SetForm(tc.form)
			if rows[0].Preview != "" {
				t.Fatalf("unsafe preview: %q", rows[0].Preview)
			}
			a.StartRename()
			if !strings.HasPrefix(a.files[0].Result, "Failed:") {
				t.Fatalf("result: %q", a.files[0].Result)
			}
			data, err := os.ReadFile(src)
			if err != nil || string(data) != "original" {
				t.Fatalf("source changed: %q %v", data, err)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 {
				t.Fatalf("unexpected root contents: %v %v", entries, err)
			}
		})
	}
}

func TestDanglingSymlinkIsConflict(t *testing.T) {
	a, dir := newTempApp(t, "a.txt")
	dest := filepath.Join(dir, "b.txt")
	if err := os.Symlink("missing", dest); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	a.SetForm(wholeForm("b"))
	res := a.StartRename()
	if res.Conflict != "b.txt" {
		t.Fatalf("expected conflict, got %+v", res)
	}
	a.ResolveConflict("skip", false)
	info, err := os.Lstat(dest)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link replaced: %v %v", info, err)
	}
}

func TestRenameWithoutOverwriteDoesNotReplaceExistingTarget(t *testing.T) {
	a, dir := newTempApp(t, "a.txt")
	dest := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(dest, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	a.renameOne(0, "b.txt", false)
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "keep" {
		t.Fatalf("target changed: %q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatalf("source lost: %v", err)
	}
	if !strings.HasPrefix(a.files[0].Result, "Failed:") {
		t.Fatalf("result: %s", a.files[0].Result)
	}
}
