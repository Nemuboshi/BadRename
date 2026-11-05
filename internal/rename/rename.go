// Package rename is the rename engine. Pure logic, no GUI, no file system.
//
// The four rules mirror the four pages of the GUI. Documented behavior:
//
//   - Letter numbering starts at a or A. The start value offsets the
//     sequence, so a start of 3 begins at c. After z comes aa, ab, and so
//     on. Zero padding applies to numbers only.
//   - The prefix, suffix, insert, and delete operations act on the name
//     without the extension. The extension stays unchanged on that page.
//   - The replace rule acts on the whole file name, extension included.
//   - Automatic conflict rename appends " (2)", " (3)", and so on before
//     the extension.
//   - Character positions count Unicode characters, not bytes.
package rename

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// CaseKind is the case conversion applied after the rename. It comes from
// the GUI combo box.
type CaseKind int

const (
	CaseUnchanged CaseKind = iota
	CaseNameLower
	CaseExtLower
	CaseBothLower
	CaseNameUpper
	CaseExtUpper
	CaseBothUpper
)

// LetterCase is the case of the letters used by letter numbering.
type LetterCase int

const (
	LetterLower LetterCase = iota
	LetterUpper
)

// RuleKind selects which of the four rules a Rule holds.
type RuleKind int

const (
	RulePattern RuleKind = iota
	RuleReplace
	RuleAddDelete
	RuleRegex
)

// Pos is an optional 1-based character position with attached text or count.
// A nil pointer means the option is off.
type Pos struct {
	At   int
	Text string
}

// Range is an optional 1-based delete range.
type Range struct {
	Start int
	Count int
}

// PatternRule builds the new name from a template. * inserts the original
// name, # inserts the serial.
type PatternRule struct {
	Pattern    string
	Start      int64
	Step       int64
	Width      int
	Pad        bool
	Letters    bool
	LetterCase LetterCase
	Extension  string // new extension without the dot, "" when unset
	HasExt     bool
	NameCase   CaseKind
}

// ReplaceRule replaces a string in the whole file name.
type ReplaceRule struct {
	From     string
	To       string
	NameCase CaseKind
}

// AddDeleteRule adds and deletes text. It acts on the name without the
// extension.
type AddDeleteRule struct {
	Prefix      string
	Suffix      string
	InsertAt    *Pos // insert Text before character At (1-based)
	DeleteText  string
	DeleteRange *Range
	NameCase    CaseKind
}

// RegexRule replaces a regular expression match in the whole file name.
// The replacement can use $1 and ${name} group references.
type RegexRule struct {
	Pattern     *regexp.Regexp
	Replacement string
	NameCase    CaseKind
}

// Rule is one rename rule, matching the four rule pages of the GUI.
type Rule struct {
	Kind      RuleKind
	Pattern   PatternRule
	Replace   ReplaceRule
	AddDelete AddDeleteRule
	Regex     RegexRule
}

// ConflictPolicy says what to do when the target name already exists.
type ConflictPolicy int

const (
	PolicyAsk ConflictPolicy = iota
	PolicyOverwrite
	PolicySkip
	PolicyAutoRename
)

// SplitName splits a file name into stem and extension. The dot is not in
// either part. A leading dot does not start an extension, so .gitignore is
// one stem.
func SplitName(name string) (string, string) {
	i := strings.LastIndex(name, ".")
	if i <= 0 {
		return name, ""
	}
	return name[:i], name[i+1:]
}

// letters converts a number to a letter sequence: 1 -> a, 26 -> z, 27 -> aa.
func letters(n *big.Int, letterCase LetterCase) string {
	x := new(big.Int).Set(n)
	if x.Sign() <= 0 {
		x.SetInt64(1)
	}
	var digits []byte
	var remainder big.Int
	for x.Sign() > 0 {
		x.Sub(x, big.NewInt(1))
		x.QuoRem(x, big.NewInt(26), &remainder)
		digits = append(digits, byte(remainder.Int64()))
	}
	base := byte('a')
	if letterCase == LetterUpper {
		base = 'A'
	}
	var b strings.Builder
	for i := len(digits) - 1; i >= 0; i-- {
		b.WriteByte(base + digits[i])
	}
	return b.String()
}

// Serial formats the serial for file index (0-based position in the list).
func Serial(r *PatternRule, index int) string {
	value := new(big.Int).Mul(big.NewInt(r.Step), big.NewInt(int64(index)))
	value.Add(value, big.NewInt(r.Start))
	if r.Letters {
		return letters(value, r.LetterCase)
	}
	s := value.String()
	// The UI specifies at most 16 digits. Enforce that limit here too,
	// so manually entered values cannot cause an unbounded allocation.
	width := min(r.Width, 16)
	if r.Pad && len(s) < width {
		zeros := strings.Repeat("0", width-len(s))
		if value.Sign() < 0 {
			return "-" + zeros + s[1:]
		}
		return zeros + s
	}
	return s
}

// ApplyCase applies the case option to a full file name.
func ApplyCase(name string, kind CaseKind) string {
	stem, ext := SplitName(name)
	switch kind {
	case CaseNameLower:
		stem = strings.ToLower(stem)
	case CaseExtLower:
		ext = strings.ToLower(ext)
	case CaseBothLower:
		stem = strings.ToLower(stem)
		ext = strings.ToLower(ext)
	case CaseNameUpper:
		stem = strings.ToUpper(stem)
	case CaseExtUpper:
		ext = strings.ToUpper(ext)
	case CaseBothUpper:
		stem = strings.ToUpper(stem)
		ext = strings.ToUpper(ext)
	}
	return join(stem, ext)
}

func join(stem, ext string) string {
	if ext == "" {
		return stem
	}
	return stem + "." + ext
}

// insertAtChars inserts text before 1-based character position pos of s.
// A position past the end appends.
func insertAtChars(s string, pos int, text string) string {
	if pos < 1 {
		pos = 1
	}
	byteIdx := charOffset(s, pos-1)
	return s[:byteIdx] + text + s[byteIdx:]
}

// deleteRangeChars deletes count characters starting at 1-based position
// start of s.
func deleteRangeChars(s string, start, count int) string {
	if count <= 0 {
		return s
	}
	if start < 1 {
		start = 1
	}
	from := charOffset(s, start-1)
	to := from + charOffset(s[from:], count)
	return s[:from] + s[to:]
}

// charOffset returns the byte offset of the i-th rune (0-based). It returns
// len(s) when i is past the end.
func charOffset(s string, i int) int {
	if i <= 0 {
		return 0
	}
	count := 0
	for off := range s {
		if count == i {
			return off
		}
		count++
	}
	return len(s)
}

// Apply computes the new name of one file. index is the 0-based position of
// the file in the list. Only the pattern rule uses it.
func Apply(rule Rule, name string, index int) string {
	switch rule.Kind {
	case RulePattern:
		r := rule.Pattern
		if r.Pattern == "" {
			// An empty template builds no name. Treat it as no change,
			// not as ".ext".
			return name
		}
		stem, ext := SplitName(name)
		out := strings.NewReplacer("*", stem, "#", Serial(&r, index)).Replace(r.Pattern)
		if r.HasExt {
			ext = strings.TrimLeft(r.Extension, ".")
		}
		return ApplyCase(join(out, ext), r.NameCase)
	case RuleReplace:
		r := rule.Replace
		if r.From == "" {
			return name
		}
		return ApplyCase(strings.ReplaceAll(name, r.From, r.To), r.NameCase)
	case RuleAddDelete:
		r := rule.AddDelete
		stem, ext := SplitName(name)
		s := r.Prefix + stem + r.Suffix
		if r.InsertAt != nil {
			s = insertAtChars(s, r.InsertAt.At, r.InsertAt.Text)
		}
		if r.DeleteText != "" {
			s = strings.ReplaceAll(s, r.DeleteText, "")
		}
		if r.DeleteRange != nil {
			s = deleteRangeChars(s, r.DeleteRange.Start, r.DeleteRange.Count)
		}
		return ApplyCase(join(s, ext), r.NameCase)
	case RuleRegex:
		r := rule.Regex
		out := r.Pattern.ReplaceAllString(name, r.Replacement)
		return ApplyCase(out, r.NameCase)
	}
	return name
}

// AutoRename builds a name that does not collide, for the AutoRename
// policy. It appends " (2)", " (3)", and so on before the extension. The
// exists callback reports whether a candidate name is taken. It returns ""
// when no free name is found within the limit.
func AutoRename(name string, exists func(string) bool) string {
	stem, ext := SplitName(name)
	for n := 2; n < 10000; n++ {
		candidate := join(stem+" ("+strconv.Itoa(n)+")", ext)
		if !exists(candidate) {
			return candidate
		}
	}
	return ""
}
