package app

import "badrename/internal/rename"

// FileRow is one entry of the file list.
type FileRow struct {
	Path   string
	Result string
}

// FormState holds the settings from all rule pages.
type FormState struct {
	Tab string
	// Whole page.
	Pattern     string
	Start       int64
	Step        int64
	Width       int
	Pad         bool
	Letters     bool
	LetterUpper bool
	ChangeExt   bool
	Extension   string
	AutoResolve bool
	// The case option is shared by all pages, like the original tool.
	NameCase int
	// Replace page.
	From string
	To   string
	// Add/Delete page. InsertAt and DeleteRange enable the two optional
	// character-position operations; they have nothing to do with the
	// file extension.
	Prefix      string
	Suffix      string
	InsertAt    bool
	InsertPos   int
	InsertText  string
	DeleteText  string
	DeleteRange bool
	DeleteStart int
	DeleteCount int
}

// RowView is one rendered line of the file table.
type RowView struct {
	Old     string
	Preview string
	Result  string
}

// RunResult reports what a rename pass did. When Conflict is set, the pass
// stopped and waits for ResolveConflict.
type RunResult struct {
	Summary  string
	Conflict string
}

// App holds the application state used by the native UI.
type App struct {
	files         []FileRow
	form          FormState
	applyAll      rename.ConflictPolicy
	conflictIdx   int
	conflictTgt   string
	done          int
	failed        int
	unchanged     int
	processedRows int
}

// NewApp creates the application state.
func NewApp() *App {
	return &App{conflictIdx: -1}
}
