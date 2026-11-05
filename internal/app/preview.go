package app

import (
	"path/filepath"
	"regexp"
	"strings"

	"badrename/internal/rename"
)

// rule builds the active rule from the form state. An invalid regex makes
// this an error with the parse message.
func (a *App) rule() (rename.Rule, error) {
	nc := rename.CaseKind(a.form.NameCase)
	switch a.form.Tab {
	case "replace":
		return rename.Rule{Kind: rename.RuleReplace, Replace: rename.ReplaceRule{
			From: a.form.From, To: a.form.To, NameCase: nc,
		}}, nil
	case "adddelete":
		r := rename.AddDeleteRule{
			Prefix: a.form.Prefix, Suffix: a.form.Suffix,
			DeleteText: a.form.DeleteText, NameCase: nc,
		}
		if a.form.InsertAt {
			r.InsertAt = &rename.Pos{At: max(a.form.InsertPos, 1), Text: a.form.InsertText}
		}
		if a.form.DeleteRange {
			r.DeleteRange = &rename.Range{Start: max(a.form.DeleteStart, 1), Count: a.form.DeleteCount}
		}
		return rename.Rule{Kind: rename.RuleAddDelete, AddDelete: r}, nil
	case "regex":
		p, err := regexp.Compile(a.form.From)
		if err != nil {
			return rename.Rule{}, err
		}
		return rename.Rule{Kind: rename.RuleRegex, Regex: rename.RegexRule{
			Pattern: p, Replacement: a.form.To, NameCase: nc,
		}}, nil
	default: // whole
		p := rename.PatternRule{
			Pattern: a.form.Pattern, Start: a.form.Start, Step: a.form.Step,
			Width: a.form.Width, Pad: a.form.Pad, Letters: a.form.Letters,
			Extension: a.form.Extension, HasExt: a.form.ChangeExt, NameCase: nc,
		}
		if a.form.LetterUpper {
			p.LetterCase = rename.LetterUpper
		}
		return rename.Rule{Kind: rename.RulePattern, Pattern: p}, nil
	}
}

// newName computes the new name of row i from an already-built rule. It
// returns "" when the name is unchanged or empty. Building the rule is
// the expensive part (a regex compiles), so callers build it once per pass
// and loop here.
func (a *App) newName(rule rename.Rule, i int) string {
	base := filepath.Base(a.files[i].Path)
	out := rename.Apply(rule, base, i)
	if out == "" || out == base {
		return ""
	}
	return out
}

// validTargetName restricts all rules to a single filename on every platform.
func validTargetName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`) && filepath.IsLocal(name)
}

// SetForm stores the form state and returns the rendered table.
func (a *App) SetForm(f FormState) []RowView {
	a.form = f
	return a.Preview()
}

// Preview renders every row with its live preview. A rule that changes
// nothing shows the original name, so no cell is left blank. A broken rule
// (an invalid regex) shows nothing; the page already reports the error.
func (a *App) Preview() []RowView {
	rule, err := a.rule()
	views := make([]RowView, len(a.files))
	for i := range a.files {
		old := filepath.Base(a.files[i].Path)
		preview := ""
		if err == nil {
			preview = a.newName(rule, i)
			if preview == "" {
				preview = old
			} else if !validTargetName(preview) {
				preview = ""
			}
		}
		views[i] = RowView{Old: old, Preview: preview, Result: a.files[i].Result}
	}
	return views
}

// RegexError returns the compile error of the form field, or "" when valid.
func (a *App) RegexError(expr string) string {
	if expr == "" {
		return ""
	}
	if _, err := regexp.Compile(expr); err != nil {
		return err.Error()
	}
	return ""
}
