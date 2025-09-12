package ui

import (
	"log"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mauzec/tdsoftgui/internal/client"
)

const (
	ScreenMain ScreenID = "main"
)

// parserMembersMenu is the UI for parsing members. It is the part of mainScreen.
//
//	Services: *client.Client
func parserMembersMenu(r *Router) fyne.CanvasObject {
	var cl *client.Client
	_ = r.GetServiceAs(&cl)

	chatNameLabel := widget.NewLabel("Channel or group name")
	chatNameEntry := widget.NewEntry()
	chatNameEntry.SetPlaceHolder("@chat")

	limitMembersLabel := widget.NewLabel("Limit members to parse")
	limitMembersEntry := widget.NewEntry()
	limitMembersEntry.SetPlaceHolder("1 - 50000, default: 1000")

	outputLabel := widget.NewLabel("Output CSV file name")
	outputEntry := widget.NewEntry()
	outputEntry.SetPlaceHolder("Optional")

	limitMessagesEntry := widget.NewEntry()
	limitMessagesEntry.Disable()
	limitMessagesEntry.SetPlaceHolder("(1 - 5000), default: 10")

	parseFromMessagesCheck := widget.NewCheck(
		"Parse members from messages", nil)
	parseFromMessagesCheck.OnChanged = func(b bool) {
		if b {
			limitMessagesEntry.Enable()
		} else {
			limitMessagesEntry.Disable()
		}
	}

	parseBioCheck := widget.NewCheck("Parse users/bots bio", nil)
	addInfoCheck := widget.NewCheck(
		"Add additional users/bots info", nil)

	parseButton := widget.NewButton("Parse", func() {
		req := &client.GetMembersRequest{}

		if chatNameEntry.Text == "" {
			return
		}
		req.ChatID = chatNameEntry.Text
		if limitMembersEntry.Text == "" {
			req.Limit = 0
		} else if limit, err := strconv.Atoi(limitMembersEntry.Text); err != nil {
			req.Limit = limit
		} else {
			return
		}
		req.Output = outputEntry.Text
		if parseFromMessagesCheck.Checked {
			req.ParseFromMessages = true
			if limitMessagesEntry.Text == "" {
				req.MessagesLimit = 0
			} else if limit, err := strconv.Atoi(limitMessagesEntry.Text); err != nil {
				req.MessagesLimit = limit
			} else {
				return
			}
		}

		req.AddAdditionalInfo = addInfoCheck.Checked
		req.ParseBio = parseBioCheck.Checked

		err := cl.GetMembers(req)
		if err != nil {
			// TODO: add error message to UI
			log.Println("failed to get members:", err)
		}
	})

	return container.NewVBox(
		container.NewBorder(nil, nil, chatNameLabel, nil, chatNameEntry),
		container.NewBorder(nil, nil, limitMembersLabel, nil,
			container.New(layout.NewGridWrapLayout(
				fyne.NewSize(
					175, limitMembersEntry.MinSize().Height)),
				limitMembersEntry),
		),
		container.NewBorder(nil, nil, outputLabel, nil, outputEntry),
		container.NewBorder(nil, nil, parseFromMessagesCheck, nil, limitMessagesEntry),
		parseBioCheck,
		addInfoCheck,
		parseButton,
	)
}

// mainScreen is the main application screen, that shows after login.
//
//	Services: *client.Client, fyne.Window
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

	const maxLogEntries = 500

	logContainer := container.NewVBox()
	logScroll := container.NewVScroll(logContainer)
	logScroll.SetMinSize(fyne.NewSize(0, 180))

	addLog := func(s string) {
		label := widget.NewRichTextWithText(s)
		label.Wrapping = fyne.TextWrapWord
		if len(logContainer.Objects) >= maxLogEntries {
			// TODO: if log amount is a lot, need optimization here
			logContainer.Objects = logContainer.Objects[1:]
		}

		logContainer.Add(label)
		logScroll.ScrollToBottom()
	}

	cl.SetUserLogger(addLog)

	return container.NewBorder(
		menu, nil, nil, nil,
		container.NewBorder(
			container.NewVBox(
				content,
				widget.NewSeparator(),
			), nil, nil, nil,
			logScroll,
		),
	)
}
