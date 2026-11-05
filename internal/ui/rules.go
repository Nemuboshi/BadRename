package ui

import (
	"badrename/internal/app"
	"errors"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// rulePanel owns all four pages, including inputs on pages that are hidden.
// Its methods are called on the UI thread.
type rulePanel struct {
	numberText                                        map[string]*string
	numberErrors                                      map[string]bool
	numberInvalid                                     map[string]bool
	inputHandles                                      map[string]*ui.Handle
	tab                                               int
	pattern                                           string
	start, step, width                                int64
	pad, letters, letterUpper, changeExt, autoResolve bool
	extension                                         string
	nameCase                                          int
	replaceFrom, replaceTo                            string
	regexFrom, regexTo                                string
	prefix, suffix                                    string
	insertAt                                          bool
	insertPos                                         int64
	insertText, deleteText                            string
	deleteRange                                       bool
	deleteStart, deleteCount                          int64
}

func newRulePanel() *rulePanel {
	return &rulePanel{
		start: 1, step: 1, width: 1, pad: true,
		insertPos: 1, deleteStart: 1, deleteCount: 1,
	}
}

// parseNumber keeps decimal integers exact while preserving truncation of fractional input.
func parseNumber(text string) (int64, bool, bool) {
	text = strings.TrimSpace(text)
	n, err := strconv.ParseInt(text, 10, 64)
	if err == nil {
		return n, false, false
	}
	if errors.Is(err, strconv.ErrRange) {
		return 0, true, false
	}
	f, err := strconv.ParseFloat(text, 64)
	if errors.Is(err, strconv.ErrRange) {
		return 0, true, false
	}
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false, true
	}
	if f == 0 {
		return 0, false, false
	}
	r, ok := new(big.Rat).SetString(text)
	if !ok {
		return 0, false, true
	}
	integer := new(big.Int).Quo(r.Num(), r.Denom())
	if !integer.IsInt64() {
		return 0, true, false
	}
	return integer.Int64(), false, false
}

func safeInt(n int64) int {
	if strconv.IntSize == 32 {
		return int(min(int64(math.MaxInt32), max(int64(math.MinInt32), n)))
	}
	return int(n)
}

func (p *rulePanel) negativeDeleteCount() bool {
	if p.deleteCount < 0 {
		return true
	}
	text := p.numberText["delete-count"]
	if text == nil {
		return false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(*text), 64)
	if err != nil || !math.Signbit(f) {
		return false
	}
	if f != 0 {
		return true
	}
	mantissa, _, _ := strings.Cut(strings.ToLower(*text), "e")
	return strings.ContainsAny(mantissa, "123456789")
}

func (p *rulePanel) validationError() string {
	if p.tab == 0 {
		for _, key := range []string{"start", "step", "width"} {
			if p.numberInvalid[key] {
				return "Enter a valid number."
			}
		}
		if p.numberErrors["start"] {
			return "Start at is outside the supported integer range."
		}
		if p.numberErrors["step"] {
			return "Step is outside the supported integer range."
		}
		if p.numberErrors["width"] {
			return "Digits is outside the supported integer range."
		}
	}
	if p.tab == 2 {
		if (p.insertAt && p.numberInvalid["insert-pos"]) || (p.deleteRange && (p.numberInvalid["delete-start"] || p.numberInvalid["delete-count"])) {
			return "Enter a valid number."
		}
		if p.insertAt && p.numberErrors["insert-pos"] {
			return "Insert position is outside the supported integer range."
		}
		if p.deleteRange {
			if p.numberErrors["delete-start"] {
				return "Delete start is outside the supported integer range."
			}
			if p.numberErrors["delete-count"] {
				return "Delete count is outside the supported integer range."
			}
			if p.negativeDeleteCount() {
				return "Delete count cannot be negative."
			}
		}
	}
	return ""
}

// form collects the active rule and the shared case option.
func (p *rulePanel) form() app.FormState {
	tabs := [...]string{"whole", "replace", "adddelete", "regex"}
	tab := p.tab
	if tab < 0 || tab >= len(tabs) {
		tab = 0
	}
	from, to := p.replaceFrom, p.replaceTo
	if tab == 3 {
		from, to = p.regexFrom, p.regexTo
	}
	return app.FormState{
		Tab: tabs[tab], Pattern: p.pattern, Start: p.start,
		Step: p.step, Width: max(1, safeInt(p.width)),
		Pad: p.pad, Letters: p.letters, LetterUpper: p.letterUpper,
		ChangeExt: p.changeExt, Extension: p.extension, AutoResolve: p.autoResolve,
		NameCase: p.nameCase, From: from, To: to, Prefix: p.prefix, Suffix: p.suffix,
		InsertAt: p.insertAt, InsertPos: safeInt(p.insertPos), InsertText: p.insertText,
		DeleteText: p.deleteText, DeleteRange: p.deleteRange,
		DeleteStart: safeInt(p.deleteStart), DeleteCount: safeInt(p.deleteCount),
	}
}

func (p *rulePanel) view(c *ui.Context) {
	ui.Column(c).Grow(1).MinHeight(0).Gap(8).Children(func() {
		names := []string{"Whole", "Replace", "Add/Delete", "Regex"}
		tabs := ui.TabsBase(c.Key("rule-tabs"), &p.tab, len(names))
		tabs.List.Gap(2).FontSize(13).BorderColor(ui.RGB(208, 208, 208)).BorderWidth(0, 0, 1, 0).Children(func() {
			for i, name := range names {
				tab := tabs.Tab(i).Padding(5, 12).Radius(4, 4, 0, 0).TextColor(ui.RGB(68, 68, 68))
				if i == p.tab {
					tab.Background(ui.RGB(255, 255, 255)).BorderColor(ui.RGB(208, 208, 208)).BorderWidth(1, 1, 0, 1).TextColor(ui.RGB(0, 0, 0)).FontWeight(600)
				} else if tab.Hovered() {
					tab.Background(ui.RGB(232, 232, 232))
				}
				tab.Children(func() { ui.Text(c, name).SingleLine() })
			}
		})
		message := p.validationError()
		if message == "" && p.tab == 2 && !p.deleteRange && p.negativeDeleteCount() {
			message = "Delete count cannot be negative."
		}
		if message != "" {
			ui.Text(c, message).FontSize(12).TextColor(ui.RGB(204, 0, 0))
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(12).Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(208, 208, 208)).Radius(6).Children(func() {
			ui.Column(c).Gap(5).FontSize(13).Children(func() {
				switch p.tab {
				case 1:
					p.replaceView(c)
				case 2:
					p.addDeleteView(c)
				case 3:
					p.regexView(c)
				default:
					p.wholeView(c)
				}
			})
		})
	})
}

func ruleFieldBackground(disabled bool) ui.Color {
	if disabled {
		return ui.RGB(240, 240, 240)
	}
	return ui.RGB(255, 255, 255)
}

func styleRuleSelect(selectBox ui.Element, disabled bool) ui.Element {
	background := ruleFieldBackground(disabled)
	if !disabled && selectBox.Hovered() {
		background = ui.RGB(232, 232, 232)
	}
	return selectBox.Padding(4, 6).FontSize(13).Background(background).Border(1, ui.RGB(197, 197, 197)).Radius(3)
}

func (p *rulePanel) wholeView(c *ui.Context) {
	patternHandle := p.inputHandle("pattern")
	ui.Text(c, "Naming rule:").OnClick(func() { patternHandle.Focus(c) })
	ui.TextInput(c.Key("pattern"), &p.pattern).Bind(patternHandle).Label("Naming rule").Placeholder("Type A_# to produce A_<number>").Padding(4, 6).FontSize(13).Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(197, 197, 197)).Radius(3)
	ui.Text(c, "Use * to insert the original file name\nUse # to insert a number or letter at that position").FontSize(12).TextColor(ui.RGB(119, 119, 119))
	p.numberRow(c, "Start at", "start", &p.start, math.MinInt64, math.MaxInt64)
	p.numberRow(c, "Step", "step", &p.step, math.MinInt64, math.MaxInt64)
	p.numberRow(c, "Digits", "width", &p.width, 1, 16)
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Checkbox(c.Key("pad"), &p.pad, "Pad short numbers").Gap(5).FontSize(13)
		ui.Checkbox(c.Key("letters"), &p.letters, "Letter numbering").Gap(5).FontSize(13)
	})
	letterCase := "Lowercase"
	if p.letterUpper {
		letterCase = "Uppercase"
	}
	letterSelect := ui.Select(c.Key("letter-case"), &letterCase, []string{"Lowercase", "Uppercase"}).Label("Letter case").Disabled(!p.letters).MinWidth(0).Width(95)
	if styleRuleSelect(letterSelect, !p.letters).Changed() {
		p.letterUpper = letterCase == "Uppercase"
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Checkbox(c.Key("change-ext"), &p.changeExt, "Change extension to").Gap(5).FontSize(13)
		ui.TextInput(c.Key("extension"), &p.extension).Label("Extension").Disabled(!p.changeExt).Width(90).Padding(4, 6).FontSize(13).Background(ruleFieldBackground(!p.changeExt)).Border(1, ui.RGB(197, 197, 197)).Radius(3)
	})
	caseHandle := p.inputHandle("name-case")
	ui.Text(c, "File name options").OnClick(func() { caseHandle.Focus(c) })
	cases := []string{"No change", "Lowercase file name", "Lowercase extension", "Lowercase name and extension", "Uppercase file name", "Uppercase extension", "Uppercase name and extension"}
	index := p.nameCase
	if index < 0 || index >= len(cases) {
		index = 0
	}
	selected := cases[index]
	nameSelect := ui.Select(c.Key("name-case"), &selected, cases).Bind(caseHandle).Label("File name options")
	if styleRuleSelect(nameSelect, false).Changed() {
		for i, option := range cases {
			if selected == option {
				p.nameCase = i
				break
			}
		}
	}
	ui.Checkbox(c.Key("auto-resolve"), &p.autoResolve, "Resolve name conflicts automatically").Gap(5).FontSize(13)
}

func (p *rulePanel) replaceView(c *ui.Context) {
	ui.Text(c, "Replace characters in the file name")
	ui.Text(c, "The extension is included.").FontSize(12).TextColor(ui.RGB(119, 119, 119))
	p.ruleInput(c, "Find", "replace-from", &p.replaceFrom)
	p.ruleInput(c, "Replace with", "replace-to", &p.replaceTo)
}

func (p *rulePanel) addDeleteView(c *ui.Context) {
	p.ruleInput(c, "Add before name", "prefix", &p.prefix)
	p.ruleInput(c, "Add after name", "suffix", &p.suffix)
	ui.Checkbox(c.Key("insert-at"), &p.insertAt, "Insert at a position").Gap(5).FontSize(13)
	insertOpacity := float32(0.5)
	if p.insertAt {
		insertOpacity = 1
	}
	ui.Column(c).Opacity(insertOpacity).Padding(0, 0, 0, 20).Gap(5).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "From char")
			p.numberInput(c, "From char", "insert-pos", &p.insertPos, 1, 9007199254740991, 60)
			ui.Text(c, "onwards, insert")
		})
		ui.TextInput(c.Key("insert-text"), &p.insertText).Label("Insert text").Padding(4, 6).FontSize(13).Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(197, 197, 197)).Radius(3)
	})
	p.ruleInput(c, "Delete from name", "delete-text", &p.deleteText)
	ui.Checkbox(c.Key("delete-range"), &p.deleteRange, "Delete a range").Gap(5).FontSize(13)
	deleteOpacity := float32(0.5)
	if p.deleteRange {
		deleteOpacity = 1
	}
	ui.Row(c).Opacity(deleteOpacity).Padding(0, 0, 0, 20).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "From char")
		p.numberInput(c, "Delete from char", "delete-start", &p.deleteStart, 1, 9007199254740991, 52)
		ui.Text(c, "delete")
		p.numberInput(c, "Delete count", "delete-count", &p.deleteCount, 1, 9007199254740991, 52)
		ui.Text(c, "chars")
	})
}

func (p *rulePanel) regexView(c *ui.Context) {
	fromHandle := p.inputHandle("regex-from")
	ui.Text(c, "Find (regular expression)").OnClick(func() { fromHandle.Focus(c) })
	ui.TextInput(c.Key("regex-from"), &p.regexFrom).Bind(fromHandle).Label("Find (regular expression)").Padding(4, 6).FontSize(13).Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(197, 197, 197)).Radius(3)
	note := "Use $1 or \u0024{name} in the replacement for capture\ngroups."
	noteColor := ui.RGB(119, 119, 119)
	if p.regexFrom != "" {
		if _, err := regexp.Compile(p.regexFrom); err != nil {
			note = "Invalid expression: " + err.Error()
			noteColor = ui.RGB(204, 0, 0)
		}
	}
	ui.Text(c, note).FontSize(12).TextColor(noteColor)
	toHandle := p.inputHandle("regex-to")
	ui.Text(c, "Replace with").OnClick(func() { toHandle.Focus(c) })
	ui.TextInput(c.Key("regex-to"), &p.regexTo).Bind(toHandle).Label("Replace with").Padding(4, 6).FontSize(13).Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(197, 197, 197)).Radius(3)
	ui.Text(c, `Example: find (\w+)_(\d+) and replace with ${2}_$1`).FontSize(12).TextColor(ui.RGB(119, 119, 119))
}

func (p *rulePanel) inputHandle(key string) *ui.Handle {
	if p.inputHandles == nil {
		p.inputHandles = make(map[string]*ui.Handle)
	}
	if p.inputHandles[key] == nil {
		p.inputHandles[key] = new(ui.Handle)
	}
	return p.inputHandles[key]
}

func (p *rulePanel) ruleInput(c *ui.Context, label, key string, value *string) {
	handle := p.inputHandle(key)
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).Width(108).TextAlign(ui.End).OnClick(func() { handle.Focus(c) })
		ui.TextInput(c.Key(key), value).Bind(handle).Label(label).Grow(1).MinWidth(0).Padding(4, 6).FontSize(13).Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(197, 197, 197)).Radius(3)
	})
}

func (p *rulePanel) numberRow(c *ui.Context, label, key string, value *int64, lo, hi int64) {
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).Width(42).TextAlign(ui.End).OnClick(func() { p.inputHandle(key).Focus(c) })
		p.numberInput(c, label, key, value, lo, hi, 70)
	})
}

// numberInput retains editable text while parsing integers exactly and truncating
// finite fractional values. Out-of-range values are reported by validation.
func (p *rulePanel) numberInput(c *ui.Context, label, key string, value *int64, lo, hi int64, width float32) {
	if p.numberText == nil {
		p.numberText = make(map[string]*string)
	}
	text := p.numberText[key]
	if text == nil {
		text = new(string)
		*text = strconv.FormatInt(*value, 10)
		p.numberText[key] = text
	}
	adjust := func(delta int64) {
		if p.numberErrors[key] || p.numberInvalid[key] {
			return
		}
		if delta > 0 && *value < hi {
			*value++
		} else if delta < 0 && *value > lo {
			*value--
		}
		*value = min(hi, max(lo, *value))
		*text = strconv.FormatInt(*value, 10)
	}
	ui.Row(c.Key(key+"-spinner")).Width(width).Height(28).Shrink(0).AlignItems(ui.Stretch).
		Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(197, 197, 197)).Radius(3).Children(func() {
		input := ui.TextInput(c.Key(key), text).Bind(p.inputHandle(key)).Label(label).Grow(1).MinWidth(0).Padding(4, 3).Background(ui.RGB(255, 255, 255)).Border(0, ui.Color{}).FontSize(13)
		if input.Changed() {
			n, outOfRange, invalid := parseNumber(*text)
			if p.numberErrors == nil {
				p.numberErrors = make(map[string]bool)
				p.numberInvalid = make(map[string]bool)
			}
			p.numberErrors[key] = outOfRange || (strconv.IntSize == 32 && key != "start" && key != "step" && (n < math.MinInt32 || n > math.MaxInt32))
			p.numberInvalid[key] = invalid
			if !p.numberErrors[key] && !invalid {
				*value = n
			}
		}
		if input.Shortcut(0, ui.KeyUp) {
			adjust(1)
		}
		if input.Shortcut(0, ui.KeyDown) {
			adjust(-1)
		}
		ui.Column(c).Width(16).Shrink(0).Children(func() {
			up := ui.ButtonBase(c.Key(key + "-up")).Label("Increase " + label).Height(13).Children(func() {
				ui.Text(c, "▲").FontSize(8).TextColor(ui.RGB(100, 100, 100))
			})
			if up.Clicked() {
				adjust(1)
			}
			down := ui.ButtonBase(c.Key(key + "-down")).Label("Decrease " + label).Height(13).Children(func() {
				ui.Text(c, "▼").FontSize(8).TextColor(ui.RGB(100, 100, 100))
			})
			if down.Clicked() {
				adjust(-1)
			}
		})
	})
}
