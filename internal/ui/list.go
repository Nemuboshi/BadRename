package ui

import (
	"badrename/internal/app"
	"github.com/egoist/mygo/ui"
)

// actionButton keeps list and footer actions aligned with the original layout.
func actionButton(c *ui.Context, label string) ui.Element {
	return ui.Button(c, label).MinWidth(80).Height(29).Padding(5, 14).FontSize(13).Border(1, ui.RGB(197, 197, 197)).Radius(4)
}

// listPanel keeps the selection and table layout between view builds.
type listPanel struct {
	selected      int
	busySelection int
	busy          bool
	table         ui.ListState
	tableBounds   ui.Handle
}

func newListPanel() *listPanel {
	return &listPanel{selected: -1}
}

// view builds the file list. Action callbacks run before the panel updates its
// selection, so the caller can read selected to identify the source row.
func (p *listPanel) view(c *ui.Context, rows []app.RowView, onAdd func(), onRemove func(), onRemoveAll func(), onMove func(int)) {
	if p.selected >= len(rows) {
		p.selected = len(rows) - 1
	}
	if p.selected < -1 {
		p.selected = -1
	}
	p.table.Selected = &p.selected
	if p.busy {
		p.busySelection = p.selected
		p.table.Selected = &p.busySelection
	}

	// Give Preview its share of the initial table width explicitly, so
	// widening Original name scrolls instead of squeezing Preview.
	previewWidth := float32(0)
	if width := p.tableBounds.Bounds(c).W; width > 0 {
		previewWidth = max(80, (width-90)/2)
	}
	columns := []ui.TableColumn{
		{Title: "Original name", MinWidth: 80},
		{Title: "Preview", Width: previewWidth, MinWidth: 80},
		{Title: "Result", Width: 90, MinWidth: 80, Fixed: true},
	}

	ui.Column(c).Grow(1).Gap(8).Children(func() {
		ui.Row(c).Height(30).Shrink(0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "File list").FontWeight(600)
		})
		ui.Column(c).Grow(1).MinHeight(0).Gap(8).Padding(8).Background(ui.RGB(255, 255, 255)).Border(1, ui.RGB(208, 208, 208)).Radius(6).Children(func() {
			ui.Row(c).Grow(1).MinHeight(0).Gap(8).AlignItems(ui.Stretch).Children(func() {
				if len(rows) == 0 {
					ui.Column(c).Grow(1).FillHeight().Center().Gap(4).Children(func() {
						ui.Text(c, "The list is empty.").TextColor(c.Theme().TextMuted)
						ui.Text(c, "Click Add to pick files, or drop files onto this window.").TextColor(c.Theme().TextMuted)
					})
				} else {
					ui.Table(c.Key("file-table"), &p.table, columns, len(rows), func(row, col int) {
						var text string
						switch col {
						case 0:
							text = rows[row].Old
						case 1:
							text = rows[row].Preview
						case 2:
							text = rows[row].Result
						}
						ui.Text(c, text).SingleLine().Shrink(1).MinWidth(0).Tooltip(text)
					}).Grow(1).FillHeight().Label("File list").Bind(&p.tableBounds)
				}

				hasSelection := !p.busy && p.selected >= 0 && p.selected < len(rows)
				canMoveUp := hasSelection && p.selected > 0 && onMove != nil
				canMoveDown := hasSelection && p.selected+1 < len(rows) && onMove != nil
				move := func(target int) {
					if !p.busy && onMove != nil && target >= 0 && target < len(rows) && p.selected >= 0 && p.selected < len(rows) && target != p.selected {
						onMove(target)
						p.selected = target
					}
				}
				ui.Column(c).Width(30).Shrink(0).Gap(6).Justify(ui.Center).Children(func() {
					ui.Button(c, "▲▲").Size(30, 26).Padding(0).FontSize(9).Border(1, ui.RGB(197, 197, 197)).Radius(3).Label("Move to top").Tooltip("Move to top").Disabled(!canMoveUp).OnClick(func() { move(0) })
					ui.Button(c, "▲").Size(30, 26).Padding(0).FontSize(9).Border(1, ui.RGB(197, 197, 197)).Radius(3).Label("Move up").Tooltip("Move up").Disabled(!canMoveUp).OnClick(func() { move(p.selected - 1) })
					ui.Button(c, "▼").Size(30, 26).Padding(0).FontSize(9).Border(1, ui.RGB(197, 197, 197)).Radius(3).Label("Move down").Tooltip("Move down").Disabled(!canMoveDown).OnClick(func() { move(p.selected + 1) })
					ui.Button(c, "▼▼").Size(30, 26).Padding(0).FontSize(9).Border(1, ui.RGB(197, 197, 197)).Radius(3).Label("Move to bottom").Tooltip("Move to bottom").Disabled(!canMoveDown).OnClick(func() { move(len(rows) - 1) })
				})
			})
			ui.Row(c).Gap(8).Justify(ui.Center).Children(func() {
				actionButton(c, "Add").Disabled(p.busy || onAdd == nil).OnClick(func() {
					if !p.busy && onAdd != nil {
						onAdd()
					}
				})
				actionButton(c, "Remove").Disabled(p.busy || p.selected < 0 || p.selected >= len(rows) || onRemove == nil).OnClick(func() {
					if !p.busy && onRemove != nil && p.selected >= 0 && p.selected < len(rows) {
						onRemove()
						p.selected = min(p.selected, len(rows)-2)
					}
				})
				actionButton(c, "Remove All").Disabled(p.busy || len(rows) == 0 || onRemoveAll == nil).OnClick(func() {
					if !p.busy && onRemoveAll != nil && len(rows) > 0 {
						onRemoveAll()
						p.selected = -1
					}
				})
			})
		})
	})
}
