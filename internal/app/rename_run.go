package app

import (
	"fmt"
	"os"
	"path/filepath"

	"badrename/internal/rename"
)

// StartRename runs the batch pass from the first row.
func (a *App) StartRename() RunResult {
	a.applyAll = rename.PolicyAsk
	a.done, a.failed, a.unchanged, a.processedRows = 0, 0, 0, 0
	return a.run(0)
}

// existsIn reports whether a name is taken in the directory of row i.
func (a *App) existsIn(i int, name string) bool {
	_, err := os.Lstat(filepath.Join(filepath.Dir(a.files[i].Path), name))
	return err == nil || !os.IsNotExist(err)
}

// run renames rows from startAt on. It stops at an Ask conflict. A broken
// rule changes nothing: newName never sees it because the buttons only
// offer a valid form, but guard anyway and treat every row as unchanged.
func (a *App) run(startAt int) RunResult {
	rule, err := a.rule()
	if err != nil {
		a.processedRows = len(a.files)
		a.unchanged += len(a.files) - startAt
		return RunResult{Summary: a.summaryText()}
	}
	for i := startAt; i < len(a.files); i++ {
		a.processedRows = i + 1
		newName := a.newName(rule, i)
		if newName == "" {
			a.unchanged++
			continue
		}
		if !validTargetName(newName) {
			a.files[i].Result = "Failed: invalid target filename"
			a.failed++
			continue
		}
		overwrite := false
		if a.existsIn(i, newName) {
			base := rename.PolicyAsk
			if a.form.Tab == "whole" && a.form.AutoResolve {
				base = rename.PolicyAutoRename
			}
			policy := base
			if a.applyAll != rename.PolicyAsk {
				policy = a.applyAll
			}
			switch policy {
			case rename.PolicyAsk:
				a.conflictIdx = i
				a.conflictTgt = newName
				return RunResult{Summary: a.summaryText(), Conflict: newName}
			case rename.PolicySkip:
				a.files[i].Result = "Skipped"
				a.unchanged++
				continue
			case rename.PolicyAutoRename:
				a.renameAuto(i, newName)
				continue
			case rename.PolicyOverwrite:
				overwrite = true
			}
		}
		a.renameOne(i, newName, overwrite)
	}
	a.processedRows = len(a.files)
	a.conflictIdx = -1
	return RunResult{Summary: a.summaryText()}
}

func (a *App) renameOne(i int, newName string, overwrite bool) {
	target := filepath.Join(filepath.Dir(a.files[i].Path), newName)
	var err error
	if !validTargetName(newName) {
		err = fmt.Errorf("invalid target filename")
	} else if overwrite {
		err = os.Rename(a.files[i].Path, target)
	} else {
		// Link creates the destination atomically without replacing an existing
		// entry. If hard links are unsupported, fail rather than risk data loss.
		err = os.Link(a.files[i].Path, target)
		if err == nil {
			err = os.Remove(a.files[i].Path)
			if err != nil {
				err = fmt.Errorf("created target but could not remove source (both retained): %w", err)
			}
		}
	}
	if err != nil {
		a.files[i].Result = fmt.Sprintf("Failed: %v", err)
		a.failed++
		return
	}
	a.files[i].Path = target
	a.files[i].Result = "OK"
	a.done++
}

// renameAuto renames row i to a free variant of want. It records a failure
// when no free name exists.
func (a *App) renameAuto(i int, want string) {
	free := rename.AutoRename(want, func(c string) bool { return a.existsIn(i, c) })
	if free == "" {
		a.files[i].Result = "Cannot rename automatically"
		a.failed++
		return
	}
	a.renameOne(i, free, false)
}

func (a *App) summaryText() string {
	return fmt.Sprintf("Files processed: %d\nRenamed:       %d\nFailed:        %d\nUnchanged:     %d",
		a.processedRows, a.done, a.failed, a.unchanged)
}

// ResolveConflict answers the conflict dialog. policy is one of overwrite,
// skip, autorename, cancel.
func (a *App) ResolveConflict(policy string, all bool) RunResult {
	i := a.conflictIdx
	a.conflictIdx = -1
	if i < 0 {
		return RunResult{Summary: a.summaryText()}
	}
	if all {
		switch policy {
		case "overwrite":
			a.applyAll = rename.PolicyOverwrite
		case "skip":
			a.applyAll = rename.PolicySkip
		case "autorename":
			a.applyAll = rename.PolicyAutoRename
		}
	}
	switch policy {
	case "cancel":
		return RunResult{Summary: a.summaryText()}
	case "skip":
		a.files[i].Result = "Skipped"
		a.unchanged++
	case "overwrite":
		a.renameOne(i, a.conflictTgt, true)
	case "autorename":
		a.renameAuto(i, a.conflictTgt)
	}
	return a.run(i + 1)
}
