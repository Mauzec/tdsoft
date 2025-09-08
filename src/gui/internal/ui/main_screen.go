package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mauzec/tdsoftgui/internal/client"
)

const (
	ScreenMain ScreenID = "main"
)

func parserMembersMenu(r *Router) fyne.CanvasObject {
	return container.NewVBox()
}

func mainScreen(r *Router) fyne.CanvasObject {
	var w fyne.Window
	_ = r.GetServiceAs(&w)
	w.Resize(fyne.NewSize(800, 600))
	var cl *client.Client
	_ = r.GetServiceAs(&cl)

	content := container.NewStack()
	content.Objects = []fyne.CanvasObject{}
	setContent := func(obj fyne.CanvasObject) {
		content.Objects = []fyne.CanvasObject{obj}
		content.Refresh()
	}

	menu := container.NewHBox(
		layout.NewSpacer(),

		widget.NewButton("Parse Members", func() {
			setContent(parserMembersMenu(r))
		}),
		widget.NewButton("TODO", func() {
			setContent(widget.NewLabel("todo"))
		}),

		layout.NewSpacer(),
	)

	return container.NewVBox(
		menu,
		content,
	)
}
