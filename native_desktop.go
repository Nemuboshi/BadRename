package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type desktop struct {
	window                                        *mygo.Window
	post                                          func(func())
	openFiles                                     func() ([]string, error)
	app                                           *App
	rules                                         *rulePanel
	list                                          *listPanel
	rows                                          []RowView
	lastForm                                      FormState
	lastError                                     string
	busy                                          bool
	conflictOpen, summaryOpen, helpOpen, applyAll bool
	conflict, summary                             string
}

func newDesktop() *desktop {
	d := &desktop{app: NewApp(), rules: newRulePanel(), list: newListPanel()}
	d.refresh()
	return d
}

func (d *desktop) refresh() {
	d.lastForm = d.rules.form()
	d.lastError = d.rules.validationError()
	d.rows = d.app.SetForm(d.lastForm)
	if d.lastError != "" {
		for i := range d.rows {
			d.rows[i].Preview = ""
		}
	}
	if d.window != nil {
		d.window.Invalidate()
	}
}

func (d *desktop) addDroppedFiles(paths []string) {
	if d.busy || d.conflictOpen || d.summaryOpen || d.helpOpen {
		return
	}
	d.app.AddPaths(paths)
	d.refresh()
}

func (d *desktop) addFiles() {
	if d.busy || d.openFiles == nil || d.post == nil {
		return
	}
	d.busy = true
	open, post := d.openFiles, d.post
	go func() {
		paths, err := open()
		post(func() {
			d.busy = false
			if err != nil {
				d.summary = err.Error()
				d.summaryOpen = true
			} else {
				d.app.AddPaths(paths)
				d.refresh()
			}
		})
	}()
}

func (d *desktop) runRename(policy string) {
	if d.busy || d.post == nil || d.rules.validationError() != "" {
		return
	}
	d.busy = true
	all := d.applyAll
	post := d.post
	go func() {
		var result RunResult
		if policy == "" {
			result = d.app.StartRename()
		} else {
			result = d.app.ResolveConflict(policy, all)
		}
		post(func() {
			d.busy = false
			d.refresh()
			d.conflict = result.Conflict
			if result.Conflict != "" {
				d.conflictOpen = true
			} else if result.Summary != "" {
				d.summary = result.Summary
				d.summaryOpen = true
			}
		})
	}()
}

func (d *desktop) view(c *ui.Context) {
	if !d.busy && (d.rules.form() != d.lastForm || d.rules.validationError() != d.lastError) {
		d.refresh()
	}
	ui.Column(c).Fill().Disabled(d.busy).Padding(10).Gap(10).Background(ui.RGBA(243, 243, 243, 1)).Children(func() {
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Gap(10).Children(func() {
			ui.Column(c).Width(340).Shrink(0).Children(func() { d.rules.view(c) })
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				d.list.busy = d.busy
				d.list.view(c, d.rows, d.addFiles, func() {
					if d.busy {
						return
					}
					d.app.RemoveRow(d.list.selected)
					d.refresh()
				}, func() {
					if d.busy {
						return
					}
					d.app.RemoveAll()
					d.refresh()
				}, func(to int) {
					if d.busy {
						return
					}
					d.app.MoveRow(d.list.selected, to)
					d.refresh()
				})
			})
		})
		ui.Row(c).Shrink(0).Gap(10).Justify(ui.Center).Children(func() {
			actionButton(c, "Start Rename").MinWidth(110).Disabled(d.busy || len(d.rows) == 0 || d.rules.validationError() != "").OnClick(func() { d.refresh(); d.runRename("") })
			actionButton(c, "Cancel").Disabled(d.busy).OnClick(func() {
				mygo.App.Quit()
			})
			actionButton(c, "Help").Disabled(d.busy).OnClick(func() { d.helpOpen = true })
		})
	})
	d.dialogs(c)
}

func (d *desktop) dialogs(c *ui.Context) {
	if d.conflictOpen {
		dialog := ui.DialogBase(c, &d.conflictOpen, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
			panel.Width(420).Padding(18).Gap(10).Background(ui.RGB(255, 255, 255)).Radius(6)
			ui.Text(c, "Name conflict").FontWeight(600)
			ui.Text(c, fmt.Sprintf("The file name %q already exists.", d.conflict))
			ui.Text(c, "Do you want to resolve this name conflict automatically?")
			ui.Checkbox(c, &d.applyAll, "Apply to all")
			ui.Row(c).Gap(8).Children(func() {
				for _, option := range []struct{ label, policy string }{{"Overwrite", "overwrite"}, {"Skip", "skip"}, {"Auto Rename", "autorename"}, {"Cancel", "cancel"}} {
					ui.Button(c, option.label).OnClick(func() { d.conflictOpen = false; d.runRename(option.policy) })
				}
			})
		})
		if dialog.Dismissed() {
			d.conflictOpen = true
		}
	}
	if d.summaryOpen {
		dialog := ui.DialogBase(c, &d.summaryOpen, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
			panel.Width(420).Padding(18).Gap(10).Background(ui.RGB(255, 255, 255)).Radius(6)
			ui.Text(c, "Summary").FontWeight(600)
			ui.Text(c, d.summary)
			ui.Button(c, "OK").OnClick(func() { d.summaryOpen = false })
		})
		if dialog.Dismissed() {
			d.summaryOpen = true
		}
	}
	if d.helpOpen {
		dialog := ui.DialogBase(c, &d.helpOpen, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
			panel.Width(420).Padding(18).Gap(10).Background(ui.RGB(255, 255, 255)).Radius(6)
			ui.Text(c, "Help").FontWeight(600)
			ui.Text(c, "Whole page: * inserts the original name, # inserts the serial. For example, A_# produces A_<number>.")
			ui.Text(c, "Replace page: replace a string in the file name.")
			ui.Text(c, "Add/Delete page: add or remove text in the name.")
			ui.Text(c, "Regex page: replace a regular expression match. Use $1 or ${name} for capture groups.")
			ui.Button(c, "Close").OnClick(func() { d.helpOpen = false })
		})
		if dialog.Dismissed() {
			d.helpOpen = true
		}
	}
}
