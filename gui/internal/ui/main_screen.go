package ui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mauzec/tdsoft/gui/internal/client"
	"go.uber.org/zap"
)

const (
	ScreenMain ScreenID = "main"
)

// chatStatsMenu gets chat statistics. It is the part of mainScreen
//
//	Services: *client.Client
func chatStatsMenu(r *Router) fyne.CanvasObject {
	var cl *client.Client
	_ = r.GetServiceAs(&cl)

	chatNameLabel := widget.NewLabel("Channel or group name")
	chatNameEntry := widget.NewEntry()
	chatNameEntry.SetPlaceHolder("@chat")

	limitMessagesLabel := widget.NewLabel("Limit messages to parse")
	limitMessagesEntry := widget.NewEntry()
	limitMessagesEntry.SetPlaceHolder("default: 0 (all)")

	outputLabel := widget.NewLabel("Output CSV file name")
	outputEntry := widget.NewEntry()
	outputEntry.SetPlaceHolder("Optional")

	parseButton := widget.NewButton("Parse", nil)
	parseButton.OnTapped = func() {
		req := &client.GetChatStatsRequest{}

		if chatNameEntry.Text == "" {
			cl.ExtLog.Warn("chat name is empty")
			return
		}
		req.ChatID = chatNameEntry.Text
		if limitMessagesEntry.Text == "" {
			req.MessagesLimit = 0
		} else if limit, err := strconv.Atoi(limitMessagesEntry.Text); err == nil {
			req.MessagesLimit = limit
		} else {
			cl.ExtLog.Warn("convert messages limit failed", zap.String("limitMessages", limitMessagesEntry.Text), zap.Error(err))
			return
		}
		req.Output = outputEntry.Text

		func() {
			fyne.Do(func() {
				chatNameEntry.Disable()
				limitMessagesEntry.Disable()
				outputEntry.Disable()
				parseButton.Disable()
			})
		}()

		errCh := make(chan error, 1)
		go func() {
			errCh <- cl.GetChatStats(req)
		}()
		enableAll := func() {
			fyne.Do(func() {
				chatNameEntry.Enable()
				limitMessagesEntry.Enable()
				outputEntry.Enable()
				parseButton.Enable()
			})
		}
		go func() {
			if err := <-errCh; err != nil {
				cl.ExtLog.Error("getting chat stats failed", zap.Error(err))
				_ = cl.UserLog(3, "Getting chat statistics failed")
			}
			enableAll()
		}()
	}
	return container.NewVBox(
		container.NewBorder(nil, nil, chatNameLabel, nil, chatNameEntry),
		container.NewBorder(nil, nil, limitMessagesLabel, nil, limitMessagesEntry),
		container.NewBorder(nil, nil, outputLabel, nil, outputEntry),
		parseButton,
	)
}

// parserMembersMenu parses members. It is the part of mainScreen.
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

	parseButton := widget.NewButton("Parse", nil)
	parseButton.OnTapped = func() {
		req := &client.GetMembersRequest{}

		if chatNameEntry.Text == "" {
			cl.ExtLog.Warn("chat name is empty")
			return
		}
		req.ChatID = chatNameEntry.Text
		if limitMembersEntry.Text == "" {
			req.Limit = 0
		} else if limit, err := strconv.Atoi(limitMembersEntry.Text); err == nil {
			req.Limit = limit
		} else {
			cl.ExtLog.Warn("convert members limit failed", zap.String("limitMembers", limitMembersEntry.Text), zap.Error(err))
			return
		}
		req.Output = outputEntry.Text
		if parseFromMessagesCheck.Checked {
			req.ParseFromMessages = true
			if limitMessagesEntry.Text == "" {
				req.MessagesLimit = 0
			} else if limit, err := strconv.Atoi(limitMessagesEntry.Text); err == nil {
				req.MessagesLimit = limit
			} else {
				cl.ExtLog.Warn("convert messages limit failed", zap.String("limitMessages", limitMessagesEntry.Text), zap.Error(err))
				return
			}
		}

		req.AddAdditionalInfo = addInfoCheck.Checked
		req.ParseBio = parseBioCheck.Checked

		func() {
			fyne.Do(func() {
				chatNameEntry.Disable()
				limitMembersEntry.Disable()
				outputEntry.Disable()
				parseFromMessagesCheck.Disable()
				limitMessagesEntry.Disable()
				parseBioCheck.Disable()
				addInfoCheck.Disable()
				parseButton.Disable()
			})
		}()

		errCh := make(chan error, 1)
		go func() {
			errCh <- cl.GetMembers(req)
		}()
		enableAll := func() {
			fyne.Do(func() {
				chatNameEntry.Enable()
				limitMembersEntry.Enable()
				outputEntry.Enable()
				parseFromMessagesCheck.Enable()
				if parseFromMessagesCheck.Checked {
					limitMessagesEntry.Enable()
				}
				parseBioCheck.Enable()
				addInfoCheck.Enable()
				parseButton.Enable()
			})
		}
		go func() {
			if err := <-errCh; err != nil {
				cl.ExtLog.Error("getting members failed", zap.Error(err))
				_ = cl.UserLog(3, "Getting members failed")
			}
			enableAll()
		}()
	}

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
		widget.NewButton("Chat Stats", func() {
			setContent(chatStatsMenu(r))
		}),
		widget.NewButton("TODO", func() {
			setContent(widget.NewLabel("todo"))
		}),

		layout.NewSpacer(),
	)

	logGrid := NewLogGrid(widget.TextGridStyleDefault)
	cl.SetUserLogger(logGrid.Pushback)

	return container.NewBorder(
		menu, nil, nil, nil,
		container.NewBorder(
			container.NewVBox(
				content,
				widget.NewSeparator(),
			), nil, nil, nil,
			logGrid.Scroll,
		),
	)
}
