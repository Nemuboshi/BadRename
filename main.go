package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	contentWidth  = 890
	contentHeight = 479
)

func main() {
	desktop := newDesktop()
	mygo.App.WhenReady(func() {
		desktop.window = mygo.NewWindow(mygo.WindowOptions{
			Title: "BadRename", Width: contentWidth, Height: contentHeight,
			UseContentSize: true, DisableResize: true,
			Content: ui.View(desktop.view),
		})
		desktop.post = desktop.window.Update
		desktop.openFiles = func() ([]string, error) {
			return mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: desktop.window, Title: "BadRename - Select Files", Multiple: true})
		}
		desktop.window.OnClose(func(event *mygo.CloseEvent) {
			if desktop.busy {
				event.PreventDefault()
			}
		})
		desktop.window.OnFileDrop(func(event *mygo.FileDropEvent) {
			desktop.addDroppedFiles(event.Paths)
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
