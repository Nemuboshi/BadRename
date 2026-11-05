package app

import (
	"os"
	"path/filepath"
	"slices"
)

// AddPaths adds dropped or picked file paths to the list. It returns the
// number of files actually added. Duplicates and directories are skipped.
func (a *App) AddPaths(paths []string) int {
	added := 0
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		// Compare the path only: a row may carry a Result from an
		// earlier run.
		if slices.ContainsFunc(a.files, func(r FileRow) bool { return r.Path == abs }) {
			continue
		}
		a.files = append(a.files, FileRow{Path: abs})
		added++
	}
	return added
}

// RemoveRow deletes the row at index i.
func (a *App) RemoveRow(i int) {
	if i >= 0 && i < len(a.files) {
		a.files = slices.Delete(a.files, i, i+1)
	}
}

// RemoveAll clears the list.
func (a *App) RemoveAll() {
	a.files = nil
}

// MoveRow moves a row to a new position. The other rows shift.
func (a *App) MoveRow(from, to int) {
	if from < 0 || to < 0 || from >= len(a.files) || to >= len(a.files) {
		return
	}
	row := a.files[from]
	a.files = slices.Delete(a.files, from, from+1)
	a.files = slices.Insert(a.files, to, row)
}
