package ui

import (
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mauzec/tdsoft/gui/internal/client"
	"github.com/mauzec/tdsoft/gui/internal/preferences"
	"github.com/mauzec/tdsoft/gui/internal/ui/custom"
	"github.com/mauzec/tdsoft/gui/internal/utils"
	"go.uber.org/zap"
)

const (
	ScreenMain ScreenID = "main"
)

// chatStatsMenu gets chat statistics. It is the part of mainScreen
//
//	Services: *client.Client, fyne.App
func chatStatsMenu(r *Router) fyne.CanvasObject {
	var (
		cl *client.Client
		a  fyne.App
	)
	_ = r.GetServiceAs(&cl)
	_ = r.GetServiceAs(&a)
	prefs := a.Preferences()

	header := widget.NewLabelWithStyle("Chat statistics",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true},
	)

	chatNameLabel := widget.NewLabel("Channel or group")
	chatNameEntry := widget.NewEntry()
	chatNameEntry.SetPlaceHolder("@chat or t.me/username or id")
	chatNameEntry.Validator = nil
	chatNameEntry.SetText(prefs.String(preferences.KeyUIChatStatsMenuChat))

	limitMessagesLabel := widget.NewLabel("Messages limit")
	limitMessagesEntry := custom.NewNumericalEntry()
	limitMessagesEntry.SetPlaceHolder("0..∞ (0 = all)")
	limitMessagesEntry.Validator = nil
	limitMessagesEntry.SetText(prefs.String(preferences.KeyUIChatStatsMenuLimit))

	outputLabel := widget.NewLabel("Output CSV")
	outputEntry := widget.NewEntry()
	outputEntry.SetPlaceHolder("e.g. stats.csv")
	outputEntry.Validator = nil
	outputEntry.SetText(prefs.String(preferences.KeyUIChatStatsMenuOutput))

	parseButton := widget.NewButton("Parse", nil)
	parseButton.OnTapped = func() {
		func() {
			fyne.Do(func() {
				chatNameEntry.Disable()
				limitMessagesEntry.Disable()
				outputEntry.Disable()
				parseButton.Disable()
			})
		}()
		enableAll := func() {
			fyne.Do(func() {
				chatNameEntry.Enable()
				limitMessagesEntry.Enable()
				outputEntry.Enable()
				parseButton.Enable()
			})
		}

		req := &client.GetChatStatsRequest{}

		chatKind, chat := utils.ValidateChatName(chatNameEntry.Text)
		if chatKind == utils.ChatNameEmpty {
			cl.ExtLog.Warn("bad chat name",
				zap.String("chat", chat))
			enableAll()
			return
		}
		req.ChatID = chat
		req.InviteLink = chatKind == utils.ChatNameInviteLink

		var limit int
		var err error
		if limitMessagesEntry.Text == "" {
			limitMessagesEntry.SetText(preferences.DefaultUIChatStatsMenuLimit)
		}
		if limit, err = utils.ValidateAndGetNumeric(
			limitMessagesEntry.Text, 1, math.MaxInt32,
		); err != nil {
			cl.ExtLog.Warn("bad messages limit",
				zap.String("value", limitMessagesEntry.Text),
				zap.Error(err),
			)
			enableAll()
			return
		}
		req.MessagesLimit = limit

		if outputEntry.Text == "" {
			outputEntry.SetText(
				"chat-stats-" + time.Now().Format("20060102-150405") + ".csv",
			)
		}
		req.Output = outputEntry.Text

		if err := req.Validate(); err != nil {
			cl.ExtLog.Error("validating get chat stats request failed",
				zap.Error(err),
			)
			enableAll()
			return
		}

		prefs.SetString(preferences.KeyUIChatStatsMenuChat, chatNameEntry.Text)
		prefs.SetString(preferences.KeyUIChatStatsMenuLimit, limitMessagesEntry.Text)
		prefs.SetString(preferences.KeyUIChatStatsMenuOutput, outputEntry.Text)

		errCh := make(chan error, 1)
		go func() {
			errCh <- cl.GetChatStats(req, false)
			enableAll()
		}()
	}

	form := container.New(layout.NewFormLayout(),
		chatNameLabel, chatNameEntry,
		limitMessagesLabel, limitMessagesEntry,
		outputLabel, outputEntry,
	)
	actions := container.NewCenter(container.New(
		layout.NewGridWrapLayout(func() fyne.Size {
			sz := parseButton.MinSize()
			return fyne.Size{Width: sz.Width + 25.0, Height: sz.Height}
		}()), parseButton,
	))
	return container.NewVBox(header, widget.NewSeparator(), form, actions)
}

// membersMenu parses members. It is the part of mainScreen.
//
//	Services: *client.Client, fyne.App
func membersMenu(r *Router) fyne.CanvasObject {
	var (
		cl *client.Client
		a  fyne.App
	)
	_ = r.GetServiceAs(&cl)
	_ = r.GetServiceAs(&a)
	prefs := a.Preferences()

	header := widget.NewLabelWithStyle(
		"Parse members", fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	chatNameLabel := widget.NewLabel("Channel or group")
	chatNameEntry := widget.NewEntry()
	chatNameEntry.SetPlaceHolder("@chat")
	chatNameEntry.Validator = nil
	chatNameEntry.SetText(prefs.String(preferences.KeyUIMembersMenuChat))

	limitMembersLabel := widget.NewLabel("Members limit")
	limitMembersEntry := custom.NewNumericalEntry()
	limitMembersEntry.SetPlaceHolder("1..50000 (default 1000)")
	limitMembersEntry.Validator = nil
	limitMembersEntry.SetText(prefs.String(preferences.KeyUIMembersMenuLimit))

	outputLabel := widget.NewLabel("Output CSV")
	outputEntry := widget.NewEntry()
	outputEntry.SetPlaceHolder("Optional")
	outputEntry.Validator = nil
	outputEntry.SetText(prefs.String(preferences.KeyUIMembersMenuOutput))

	limitMessagesEntry := custom.NewNumericalEntry()
	limitMessagesEntry.Disable()
	limitMessagesEntry.SetPlaceHolder("1..5000 (default 10)")
	limitMessagesEntry.Validator = nil

	parseFromMessagesCheck := widget.NewCheck("Parse members from messages", nil)
	parseFromMessagesCheck.OnChanged = func(b bool) {
		if b {
			limitMessagesEntry.Enable()
			limitMessagesEntry.SetText(prefs.String(preferences.KeyUIMembersMenuMsgLimit))
		} else {
			limitMessagesEntry.Disable()
			limitMessagesEntry.SetText("")
		}
	}
	parseFromMessagesCheck.SetChecked(prefs.Bool(preferences.KeyUIMembersMenuParseMsgs))
	if parseFromMessagesCheck.Checked {
		limitMessagesEntry.SetText(prefs.String(preferences.KeyUIMembersMenuMsgLimit))
	}

	parseBioCheck := widget.NewCheck("Parse bio", nil)
	parseBioCheck.SetChecked(prefs.Bool(preferences.KeyUIMembersMenuParseBio))

	addInfoCheck := widget.NewCheck("Add additional info", nil)
	addInfoCheck.SetChecked(prefs.Bool(preferences.KeyUIMembersMenuAddInfo))

	parseButton := widget.NewButton("Parse", nil)

	parseButton.OnTapped = func() {
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

		req := &client.GetMembersRequest{}

		chatKind, chat := utils.ValidateChatName(chatNameEntry.Text)
		if chatKind == utils.ChatNameEmpty {
			cl.ExtLog.Warn("bad chat name",
				zap.String("chat", chat))
			enableAll()
			return
		}
		req.ChatID = chat
		req.InviteLink = chatKind == utils.ChatNameInviteLink

		var membersLimit int
		var err error
		if limitMembersEntry.Text == "" {
			limitMembersEntry.SetText(preferences.DefaultUIMembersMenuLimit)
		}
		if membersLimit, err = utils.ValidateAndGetNumeric(
			limitMembersEntry.Text, 1, 50000,
		); err != nil {
			cl.ExtLog.Warn("bad members limit",
				zap.String("value", limitMembersEntry.Text),
				zap.Error(err),
			)
			enableAll()
			return
		}
		req.Limit = membersLimit

		if outputEntry.Text == "" {
			outputEntry.SetText(
				"chat-members-" + time.Now().Format("20060102-150405") + ".csv",
			)
		}
		req.Output = outputEntry.Text

		if parseFromMessagesCheck.Checked {
			var msgLimit int
			if limitMessagesEntry.Text == "" {
				limitMessagesEntry.SetText(preferences.DefaultUIMemberMenuMsgLimit)
			}
			if msgLimit, err = utils.ValidateAndGetNumeric(
				limitMessagesEntry.Text, 1, 5000,
			); err != nil {
				cl.ExtLog.Warn("bad messages limit",
					zap.String("value", limitMessagesEntry.Text),
					zap.Error(err),
				)
				enableAll()
				return
			}
			req.ParseFromMessages = true
			req.MessagesLimit = msgLimit
		}

		req.ParseBio = parseBioCheck.Checked
		req.AddAdditionalInfo = addInfoCheck.Checked

		if err := req.Validate(); err != nil {
			cl.ExtLog.Error("validating get chat members request failed",
				zap.Error(err),
			)
			enableAll()
			return
		}

		prefs.SetString(preferences.KeyUIMembersMenuChat, chatNameEntry.Text)
		prefs.SetString(preferences.KeyUIMembersMenuLimit, limitMembersEntry.Text)
		prefs.SetString(preferences.KeyUIMembersMenuOutput, outputEntry.Text)
		prefs.SetBool(preferences.KeyUIMembersMenuParseMsgs, parseFromMessagesCheck.Checked)
		prefs.SetString(preferences.KeyUIMembersMenuMsgLimit, limitMessagesEntry.Text)
		prefs.SetBool(preferences.KeyUIMembersMenuParseBio, parseBioCheck.Checked)
		prefs.SetBool(preferences.KeyUIMembersMenuAddInfo, addInfoCheck.Checked)

		errCh := make(chan error, 1)
		go func() {
			errCh <- cl.GetMembers(req, false)
			enableAll()
		}()
	}

	form := container.New(layout.NewFormLayout(),
		chatNameLabel, chatNameEntry,
		limitMembersLabel, limitMembersEntry,
		outputLabel, outputEntry,
	)

	msgRow := container.New(layout.NewFormLayout(),
		parseFromMessagesCheck, limitMessagesEntry,
	)
	options := container.NewHBox(parseBioCheck, addInfoCheck)
	actions := container.NewCenter(container.New(
		layout.NewGridWrapLayout(func() fyne.Size {
			sz := parseButton.MinSize()
			return fyne.Size{Width: sz.Width + 25.0, Height: sz.Height}
		}()), parseButton,
	))

	return container.NewVBox(
		header,
		widget.NewSeparator(),
		form,
		msgRow,
		options,
		actions,
	)
}

// searchMessagesMenu search messages from username in given chat. It is the part of mainScreen.
//
//	Services: *client.Client, fyne.App
func searchMessagesMenu(r *Router) fyne.CanvasObject {
	var (
		cl *client.Client
		a  fyne.App
	)
	_ = r.GetServiceAs(&cl)
	_ = r.GetServiceAs(&a)
	prefs := a.Preferences()

	header := widget.NewLabelWithStyle("Search messages",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true},
	)

	chatNameEntry := widget.NewEntry()
	chatNameEntry.SetPlaceHolder("@chat or t.me/username or id")
	chatNameEntry.Validator = nil
	chatNameEntry.SetText(prefs.String(preferences.KeyUIMsgSearcherMenuChat))

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("@username")
	usernameEntry.Validator = nil
	usernameEntry.SetText(prefs.String(preferences.KeyUIMsgSearcherMenuUsername))

	outputEntry := widget.NewEntry()
	outputEntry.SetPlaceHolder("Optional")
	outputEntry.Validator = nil
	outputEntry.SetText(prefs.String(preferences.KeyUIMsgSearcherMenuOutput))

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("Channel or group"), chatNameEntry,
		widget.NewLabel("Username"), usernameEntry,
		widget.NewLabel("Output CSV"), outputEntry,
	)
	t := time.Now()
	fromDateEntry, toDateEntry := widget.NewDateEntry(), widget.NewDateEntry()
	if fd, err := time.Parse(
		prefs.String(preferences.KeyUIMsgSearcherMenuFromDate),
		"01/02/2006",
	); err == nil {
		fromDateEntry.SetDate(&fd)
	} else {
		fromDateEntry.SetDate(&t)
	}
	if fd, err := time.Parse(
		prefs.String(preferences.KeyUIMsgSearcherMenuToDate),
		"01/02/2006",
	); err == nil {
		toDateEntry.SetDate(&fd)
	} else {
		toDateEntry.SetDate(&t)
	}
	fromDateEntry.SetPlaceHolder("MM/DD/YYYY")
	toDateEntry.SetPlaceHolder("MM/DD/YYYY")
	fromDateEntry.Validator = nil
	fromDateEntry.OnChanged = func(d *time.Time) {
		cl.ExtLog.Debug("call from date changed", zap.Any("date", d))
		utils.ValidateTime(d)
		cl.ExtLog.Debug("from date changed", zap.Any("date", t))
	}
	toDateEntry.Validator = nil
	toDateEntry.OnChanged = func(d *time.Time) {
		cl.ExtLog.Debug("call to date changed", zap.Any("date", d))
		utils.ValidateTime(d)
		cl.ExtLog.Debug("to date changed", zap.Any("date", t))
	}
	dateRow := container.NewCenter(
		container.NewHBox(
			container.New(layout.NewGridWrapLayout(
				fyne.NewSize(180, fromDateEntry.MinSize().Height)),
				fromDateEntry,
			),
			container.New(layout.NewGridWrapLayout(
				fyne.NewSize(180, toDateEntry.MinSize().Height)),
				toDateEntry,
			),
		),
	)

	searchButton := widget.NewButton("Search", nil)
	actions := container.NewCenter(container.New(
		layout.NewGridWrapLayout(func() fyne.Size {
			sz := searchButton.MinSize()
			return fyne.Size{Width: sz.Width + 27.0, Height: sz.Height}
		}()), searchButton,
	))

	searchButton.OnTapped = func() {
		func() {
			fyne.Do(func() {
				chatNameEntry.Disable()
				usernameEntry.Disable()
				outputEntry.Disable()
				fromDateEntry.Disable()
				toDateEntry.Disable()
				searchButton.Disable()
			})
		}()
		enableAll := func() {
			fyne.Do(func() {
				chatNameEntry.Enable()
				usernameEntry.Enable()
				outputEntry.Enable()
				fromDateEntry.Enable()
				toDateEntry.Enable()
				searchButton.Enable()
			})
		}

		req := &client.SearchMessagesRequest{}

		chatKind, chat := utils.ValidateChatName(chatNameEntry.Text)
		if chatKind == utils.ChatNameEmpty {
			cl.ExtLog.Warn("bad chat name",
				zap.String("chat", chat))
			enableAll()
			return
		}
		req.ChatID = chat
		req.InviteLink = chatKind == utils.ChatNameInviteLink

		if !utils.ValidateUsername(usernameEntry.Text) {
			cl.ExtLog.Warn("bad username",
				zap.String("username", usernameEntry.Text))
			enableAll()
			return
		}
		req.Username = usernameEntry.Text

		if outputEntry.Text == "" {
			outputEntry.SetText(
				"chat-members-" + time.Now().Format("20060102-150405") + ".csv",
			)
		}
		req.Output = outputEntry.Text

		if !utils.ValidateTime(fromDateEntry.Date) {
			cl.ExtLog.Warn("bad from date",
				zap.Any("date", fromDateEntry.Date))
			enableAll()
			return
		}
		if !utils.ValidateTime(toDateEntry.Date) {
			cl.ExtLog.Warn("bad to date",
				zap.Any("date", toDateEntry.Date))
			enableAll()
			return
		}

		req.FromDate = fromDateEntry.Date.Format("2006/01/02")
		req.ToDate = toDateEntry.Date.Format("2006/01/02")

		if err := req.Validate(); err != nil {
			cl.ExtLog.Error("validating search messages request failed",
				zap.Error(err),
			)
			enableAll()
			return
		}

		prefs.SetString(preferences.KeyUIMsgSearcherMenuChat, chatNameEntry.Text)
		prefs.SetString(preferences.KeyUIMsgSearcherMenuUsername, usernameEntry.Text)
		prefs.SetString(preferences.KeyUIMsgSearcherMenuOutput, outputEntry.Text)
		prefs.SetString(preferences.KeyUIMsgSearcherMenuFromDate, req.FromDate)
		prefs.SetString(preferences.KeyUIMsgSearcherMenuToDate, req.ToDate)

		errCh := make(chan error, 1)
		go func() {
			errCh <- cl.SearchMessages(req, false)
			enableAll()
		}()
	}

	return container.NewVBox(
		header,
		widget.NewSeparator(),
		form,
		dateRow,
		actions,
	)
}

// TODO:
func printDialogsMenu(r *Router) fyne.CanvasObject {
	printButton := widget.NewButton("Print dialogs", nil)
	printButton.OnTapped = func() {

	}
	return container.NewVBox()
}

// TODO:
func loadingScreen() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Loading..."),
		widget.NewProgressBarInfinite(),
	)
}

// mainScreen is the main application screen, that shows after login.
//
//	Services: *client.Client, fyne.Window, fyne.App(not used here, but need)
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

	logGrid := custom.NewLogGrid(widget.TextGridStyleDefault)
	cl.SetUserLogger(logGrid.Pushback)
	// logGrid.Scroll.Hide()

	menu := container.NewHBox(
		layout.NewSpacer(),

		widget.NewButton("Members", func() {
			setContent(membersMenu(r))
			// logGrid.Scroll.Show()
		}),
		widget.NewButton("Chat Stats", func() {
			setContent(chatStatsMenu(r))
			// logGrid.Scroll.Show()
		}),
		widget.NewButton("Search Messages", func() {
			setContent(searchMessagesMenu(r))
			// logGrid.Scroll.Show()
		}),
		widget.NewButton("TODO", func() {
			setContent(widget.NewLabel("todo"))
			// logGrid.Scroll.Show()
		}),

		layout.NewSpacer(),
	)

	cl.CheckConnection()

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
