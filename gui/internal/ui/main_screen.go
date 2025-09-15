package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mauzec/tdsoft/gui/internal/client"
	"github.com/mauzec/tdsoft/gui/internal/utils"
	"go.uber.org/zap"
)

// TODO: add red lighting of invalid entries
// TODO: add more validation to entries (chat name for ex)

const (
	ScreenMain ScreenID = "main"
)

type ParserMembersState struct {
	Chat              binding.String
	Limit             binding.String
	Output            binding.String
	ParseFromMessages binding.Bool
	MessagesLimit     binding.String
	ExcludeBots       binding.Bool
	ParseBio          binding.Bool
	AddAdditional     binding.Bool
	AutoJoin          binding.Bool
}

type ChatStatsState struct {
	Chat          binding.String
	MessagesLimit binding.String
	Output        binding.String
}

type UIMainState struct {
	ParserMembers *ParserMembersState
	ChatStats     *ChatStatsState
}

func NewUIState() *UIMainState {
	return &UIMainState{
		ParserMembers: &ParserMembersState{
			Chat:              binding.NewString(),
			Limit:             binding.NewString(),
			Output:            binding.NewString(),
			ParseFromMessages: binding.NewBool(),
			MessagesLimit:     binding.NewString(),
			ExcludeBots:       binding.NewBool(),
			ParseBio:          binding.NewBool(),
			AddAdditional:     binding.NewBool(),
			AutoJoin:          binding.NewBool(),
		},
		ChatStats: &ChatStatsState{
			Chat:          binding.NewString(),
			MessagesLimit: binding.NewString(),
			Output:        binding.NewString(),
		},
	}
}

// chatStatsMenu gets chat statistics. It is the part of mainScreen
//
//	Services: *client.Client
func chatStatsMenu(r *Router) fyne.CanvasObject {
	var (
		cl *client.Client
		st *UIMainState
	)
	_ = r.GetServiceAs(&st)
	_ = r.GetServiceAs(&cl)

	header := widget.NewLabelWithStyle("Chat statistics",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true},
	)

	chatNameLabel := widget.NewLabel("Channel or group")
	chatNameEntry := widget.NewEntryWithData(st.ChatStats.Chat)
	chatNameEntry.SetPlaceHolder("@chat or t.me/username or id")
	chatNameEntry.Validator = nil

	limitMessagesLabel := widget.NewLabel("Messages limit")
	limitMessagesEntry := widget.NewEntryWithData(st.ChatStats.MessagesLimit)
	limitMessagesEntry.SetPlaceHolder("default: 0 (all)")
	limitMessagesEntry.Validator = nil

	outputLabel := widget.NewLabel("Output CSV")
	outputEntry := widget.NewEntryWithData(st.ChatStats.Output)
	outputEntry.SetPlaceHolder("Optional, e.g. stats.csv")
	outputEntry.Validator = nil

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

		chat, _ := st.ChatStats.Chat.Get()
		if err := utils.PreValidateChatName(chat); err != nil {
			cl.ExtLog.Warn("bad chat name",
				zap.String("chat", chat), zap.Error(err),
			)
			enableAll()
			return
		}
		req.ChatID = chat

		if v, err := st.ChatStats.MessagesLimit.Get(); err != nil {
			cl.ExtLog.Warn("something wrong getting messages limit",
				zap.Any("value", limitMessagesEntry.Text),
				zap.Error(err),
			)
			enableAll()
			return
		} else if v == "" {
			req.MessagesLimit = 0
		} else if limit, err := utils.ValidateAndGetNumeric(v); err != nil {
			cl.ExtLog.Warn("bad messages limit",
				zap.Any("value", v), zap.Error(err),
			)
			enableAll()
			return
		} else {
			req.MessagesLimit = limit
		}

		out, _ := st.ChatStats.Output.Get()
		req.Output = out

		if err := req.Validate(); err != nil {
			cl.ExtLog.Warn("bad GetChatStatsRequest",
				zap.Any("req", req), zap.Error(err),
			)
			enableAll()
			return
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- cl.GetChatStats(req, false)
		}()

		go func() {
			if err := <-errCh; err != nil {
				cl.ExtLog.Error("getting chat stats failed", zap.Error(err))
				_ = cl.UserLog(3, "Getting chat statistics failed")
			}
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

// parserMembersMenu parses members. It is the part of mainScreen.
//
//	Services: *client.Client
func parserMembersMenu(r *Router) fyne.CanvasObject {
	var (
		cl *client.Client
		st *UIMainState
	)
	_ = r.GetServiceAs(&st)
	_ = r.GetServiceAs(&cl)

	header := widget.NewLabelWithStyle(
		"Parse members", fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	chatNameLabel := widget.NewLabel("Channel or group")
	chatNameEntry := widget.NewEntryWithData(st.ParserMembers.Chat)
	chatNameEntry.SetPlaceHolder("@chat")
	chatNameEntry.Validator = nil

	limitMembersLabel := widget.NewLabel("Members limit")
	limitMembersEntry := widget.NewEntryWithData(st.ParserMembers.Limit)
	limitMembersEntry.SetPlaceHolder("1..50000 (default 1000)")
	limitMembersEntry.Validator = nil

	outputLabel := widget.NewLabel("Output CSV")
	outputEntry := widget.NewEntryWithData(st.ParserMembers.Output)
	outputEntry.SetPlaceHolder("Optional")
	outputEntry.Validator = nil

	limitMessagesEntry := widget.NewEntryWithData(st.ParserMembers.MessagesLimit)
	limitMessagesEntry.Disable()
	limitMessagesEntry.SetPlaceHolder("1..5000 (default 10)")
	limitMessagesEntry.Validator = nil

	parseFromMessagesCheck := widget.NewCheckWithData(
		"Parse members from messages", st.ParserMembers.ParseFromMessages)
	parseFromMessagesCheck.OnChanged = func(b bool) {
		if b {
			limitMessagesEntry.Enable()
		} else {
			limitMessagesEntry.Disable()
		}
	}

	parseBioCheck := widget.NewCheckWithData("Parse bio", st.ParserMembers.ParseBio)
	addInfoCheck := widget.NewCheckWithData(
		"Add additional info", st.ParserMembers.AddAdditional)

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

		chat, _ := st.ParserMembers.Chat.Get()
		if err := utils.PreValidateChatName(chat); err != nil {
			cl.ExtLog.Warn("bad chat name",
				zap.String("chat", chat), zap.Error(err),
			)
			enableAll()
			return
		}
		req.ChatID = chat

		if v, err := st.ParserMembers.Limit.Get(); err != nil {
			cl.ExtLog.Warn("something wrong getting members limit",
				zap.Any("value", limitMembersEntry.Text),
				zap.Error(err),
			)
			enableAll()
			return
		} else if v == "" {
			req.Limit = 0
		} else if limit, err := utils.ValidateAndGetNumeric(v); err != nil {
			cl.ExtLog.Warn("bad members limit",
				zap.Any("value", v), zap.Error(err),
			)
			enableAll()
			return
		} else {
			req.Limit = limit
		}

		out, _ := st.ParserMembers.Output.Get()
		req.Output = out

		parseFrom, _ := st.ParserMembers.ParseFromMessages.Get()
		if parseFrom {
			req.ParseFromMessages = true

			if v, err := st.ParserMembers.MessagesLimit.Get(); err != nil {
				cl.ExtLog.Warn("something wrong getting messages limit",
					zap.Any("value", limitMessagesEntry.Text),
					zap.Error(err),
				)
				enableAll()
			} else if v == "" {
				req.MessagesLimit = 0
			} else if limit, err := utils.ValidateAndGetNumeric(v); err != nil {
				cl.ExtLog.Warn("bad messages limit",
					zap.Any("value", v), zap.Error(err),
				)
				enableAll()
				return
			} else {
				req.MessagesLimit = limit
			}
		}

		addInfo, _ := st.ParserMembers.AddAdditional.Get()
		req.AddAdditionalInfo = addInfo
		parseBio, _ := st.ParserMembers.ParseBio.Get()
		req.ParseBio = parseBio

		if err := req.Validate(); err != nil {
			cl.ExtLog.Warn("bad GetMembersRequest",
				zap.Any("req", req), zap.Error(err),
			)
			enableAll()
			return
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- cl.GetMembers(req, false)
		}()

		go func() {
			if err := <-errCh; err != nil {
				cl.ExtLog.Error("getting members failed", zap.Error(err))
				_ = cl.UserLog(3, "Getting members failed")
			}
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

// mainScreen is the main application screen, that shows after login.
//
//	Services: *client.Client, fyne.Window
func mainScreen(r *Router) fyne.CanvasObject {
	var w fyne.Window
	_ = r.GetServiceAs(&w)
	w.Resize(fyne.NewSize(800, 600))
	var cl *client.Client
	_ = r.GetServiceAs(&cl)
	r.PutService(NewUIState())

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
