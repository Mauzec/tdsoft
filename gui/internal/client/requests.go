package client

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type OutHandler func(string, *PyMsg)
type ErrHandler func(*PyMsg)

func ComposeOnOut(base, extra map[string]OutHandler) OutHandler {
	return func(t string, env *PyMsg) {
		if env == nil {
			return
		}
		if hdl, ok := extra[env.Code]; ok {
			hdl(t, env)
			return
		}
		if hdl, ok := base[env.Code]; ok {
			hdl(t, env)
			return
		}
		panic(fmt.Sprintf("unhandled output from python: %v", env))
	}
}
func ComposeOnErr(base, extra map[string]ErrHandler) ErrHandler {
	return func(env *PyMsg) {
		if env == nil {
			return
		}
		if hdl, ok := extra[env.Code]; ok {
			hdl(env)
			return
		}
		if hdl, ok := base[env.Code]; ok {
			hdl(env)
			return
		}
		panic(fmt.Sprintf("unhandled error from python: %v", env))
	}
}

// All request fields are required.

type Request interface {
	Validate() error
}

type GetMembersRequest struct {
	// ChatID is either a username(t.me/chat, chat, @chat),
	// a chatID (not peerID), or invite link.
	//
	// TODO: invite link not supported yet.
	ChatID string `validate:"required"`

	// InviteLink says if chat is an invite link.
	InviteLink bool `validate:"-"`

	// Limit is the maximum number of members to return.
	// The maximum is 50,000.
	Limit int `validate:"min=1,max=50000"`

	// Output is the path to the CSV file where
	// results will be saved.
	Output string `validate:"min=1,filepath"`

	// ParseFromMessages parses users/bots from messages if true.
	// Default is false.
	ParseFromMessages bool `validate:"-"`

	// MessagesLimit is the number of messages to parse
	// if ParseFromMessages is true. The maximum is 5000.
	// Validate if ParseFromMessages is true.
	MessagesLimit int `validate:"omitempty,min=1,max=5000"`

	// TODO: not implemented yet.
	ExcludeBots bool `validate:"-"`

	// ParseBio parses users' bio.
	// This may slow down the process.
	ParseBio bool `validate:"-"`

	// AddAdditionalInfo adds additional information about users,
	// such as bio, premium, scam flag, etc.
	AddAdditionalInfo bool `validate:"-"`

	// AutoJoin automatically joins the chat if true.
	//
	// TODO: not implemented yet
	AutoJoin bool `validate:"-"`
}

func (req *GetMembersRequest) Validate() error {
	if req.ParseFromMessages && req.MessagesLimit == 0 {
		return errors.New("get zero messages limit, but parse from messages is true")
	}
	return validator.New().Struct(req)
}

type GetChatStatsRequest struct {
	// ChatID is either a username(t.me/chat, chat, @chat),
	// a chatID (not peerID), or invite link.
	//
	// TODO: invite link not supported yet.
	ChatID string `validate:"required"`

	// InviteLink says if chat is an invite link.
	InviteLink bool `validate:"-"`

	// MessagesLimit is the number of messages to parse
	// No max value, 0 means all messages
	MessagesLimit int `validate:"min=0"`

	// Output is the path to the CSV file where
	// results will be saved.
	Output string `validate:"min=1,filepath"`
}

func (req *GetChatStatsRequest) Validate() error {
	return validator.New().Struct(req)
}

type SearchMessagesRequest struct {
	// ChatID is either a username(t.me/chat, chat, @chat),
	// a chatID (not peerID), or invite link.
	//
	// TODO: invite link not supported yet.
	ChatID string `validate:"required"`

	// InviteLink says if chat is an invite link.
	InviteLink bool `validate:"-"`

	// Username is a username(t.me/user, user, @user)
	// Required
	Username string `validate:"required"`

	// Output is the path to the CSV file where
	// results will be saved.
	Output string `validate:"min=1,filepath"`

	// FromDate is the start date in MM/DD/YYYY format.
	// Required
	FromDate string `validate:"required"`

	// ToDate is the end date in MM/DD/YYYY format.
	// Required
	ToDate string `validate:"required"`
}

func (req *SearchMessagesRequest) Validate() error {
	return validator.New().Struct(req)
}

type PrintDialogsRequest struct {
	// Limit is the maximum number of dialogs to receive.
	// No max value
	Limit int `validate:"min=1"`

	// InviteLink says if chat is an invite link.
	InviteLink bool `validate:"-"`

	// Output is the path to the CSV file where
	// results will be saved.
	Output string `validate:"min=1,filepath"`
}

// GetMembers get members of a group/channel if possible
func (cl *Client) GetMembers(req *GetMembersRequest, validate bool) error {
	if err := cl.ensureUserLogF(); err != nil {
		return err
	}
	if validate {
		if err := req.Validate(); err != nil {
			cl.ExtLog.Error(
				"validating get members request failed",
				zap.Error(err),
			)
			return err
		}
	}
	cl.ExtLog.Info("get members", zap.Any("request", req))

	args := []string{cl.cfg.ScriptsPath + "/get_members.py"}
	args = append(args, "../"+cl.cfg.Session, req.ChatID)
	args = append(args, "--limit", strconv.Itoa(req.Limit))
	args = append(args, "--output", req.Output)
	if req.ParseFromMessages {
		args = append(args, "--parse-from-messages")
		args = append(args, "--messages-limit", strconv.Itoa(req.MessagesLimit))

	}
	if req.ParseBio {
		args = append(args, "--parse-bio")
	}
	if req.AddAdditionalInfo {
		args = append(args, "--add-additional-info")
	}

	extraOut := map[string]OutHandler{
		"MEMBERS_FETCHED": func(t string, pm *PyMsg) {
			cl.ExtLog.Info("fetched members",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(1,
				fmt.Sprintf("fetched %v members", pm.Details["total"]),
			)
		},
		"MEMBERS_FROM_MESSAGES_FETCHED": func(t string, pm *PyMsg) {
			cl.ExtLog.Info("fetched members from messages",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(1,
				fmt.Sprintf("fetched %v members from messages",
					pm.Details["total"]))
		},
	}
	extraErr := map[string]ErrHandler{
		"MEMBERS_LIMIT_TOO_HIGH": func(pm *PyMsg) {
			cl.ExtLog.Error("limit too high",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(3, fmt.Sprintf("limit too high, got %v, max is %v",
				pm.Details["limit"], pm.Details["max"]))
		},
		"MESSAGE_LIMIT_TOO_HIGH": func(pm *PyMsg) {
			cl.ExtLog.Error("messages limit too high",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(3, fmt.Sprintf("messages limit too high, got %v, max is %v",
				pm.Details["limit"], pm.Details["max"]))
		},
		"INVALID_CHAT_NAME": func(pm *PyMsg) {
			cl.ExtLog.Error("invalid chat name",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(3, fmt.Sprintf("invalid chat name: %s",
				pm.Details["name"]))
		},
		"INVITE_LINK_NOT_SUPPORTED": func(pm *PyMsg) {
			cl.ExtLog.Error("invite link not supported yet")
			_ = cl.UserLog(3, "invite link not supported yet")
		},
	}
	onOut := ComposeOnOut(cl.defaultPyOutHandlers, extraOut)
	onErr := ComposeOnErr(cl.defaultPyErrHandlers, extraErr)

	if err := runPyWithStreaming(cl.cfg.VenvPath, args, onOut, onErr); err != nil {
		return err
	}
	return nil
}

func (cl *Client) GetChatStats(req *GetChatStatsRequest, validate bool) error {
	if cl.UserLogF == nil {
		cl.ExtLog.Error("no user log function to set")
		return errors.New("no user log function set")
	}
	if validate {
		if err := req.Validate(); err != nil {
			cl.ExtLog.Error("validating get chat stats request failed",
				zap.Error(err),
			)
			return err
		}
	}
	cl.ExtLog.Info("get chat stats", zap.Any("request", req))

	args := []string{cl.cfg.ScriptsPath + "/get_chat_statistic.py"}
	args = append(args, "../"+cl.cfg.Session, req.ChatID)
	args = append(args, "--messages-limit", strconv.Itoa(req.MessagesLimit))
	args = append(args, "--output", req.Output)
	if req.InviteLink {
		args = append(args, "--invite-link")
	}

	extraOut := map[string]OutHandler(nil)
	extraErr := map[string]ErrHandler{
		"INVALID_CHAT_NAME": func(pm *PyMsg) {
			cl.ExtLog.Error("invalid chat name",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(3, fmt.Sprintf("invalid chat name: %s",
				pm.Details["name"]))
		},
		"INVITE_LINK_NOT_SUPPORTED": func(pm *PyMsg) {
			cl.ExtLog.Error("invite link not supported yet")
			_ = cl.UserLog(3, "invite link not supported yet")
		},
	}
	onOut := ComposeOnOut(cl.defaultPyOutHandlers, extraOut)
	onErr := ComposeOnErr(cl.defaultPyErrHandlers, extraErr)

	if err := runPyWithStreaming(cl.cfg.VenvPath, args, onOut, onErr); err != nil {
		return err
	}
	return nil
}

func (cl *Client) SearchMessages(req *SearchMessagesRequest, validate bool) error {
	if cl.UserLogF == nil {
		cl.ExtLog.Error("no user log function to set")
		return errors.New("no user log function set")
	}
	if validate {
		if err := req.Validate(); err != nil {
			cl.ExtLog.Error(
				"validating search messages request failed",
				zap.Error(err),
			)
			return err
		}
	}
	cl.ExtLog.Info("searching messages", zap.Any("request", req))

	args := []string{cl.cfg.ScriptsPath + "/search_messages.py"}
	args = append(args, "../"+cl.cfg.Session, req.ChatID, req.Username)

	if req.Output != "" {
		args = append(args, "--output", req.Output)
	}
	args = append(args, "--from-date", req.FromDate)
	args = append(args, "--to-date", req.ToDate)

	extraOut := map[string]OutHandler{
		"MESSAGES_FETCHED": func(s string, pm *PyMsg) {
			cl.ExtLog.Info("fetched messages",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(1,
				fmt.Sprintf("fetched %v messages", pm.Details["total"]),
			)
		},
	}
	extraErr := map[string]ErrHandler{
		"FROM_DATE_REQUIRED": func(pm *PyMsg) {
			cl.ExtLog.Error("got no from date")
			_ = cl.UserLog(3, "start date is required")
		},
		"TO_DATE_REQUIRED": func(pm *PyMsg) {
			cl.ExtLog.Error("got no to date")
			_ = cl.UserLog(3, "end date is required")
		},
		"FROM_DATE_INVALID": func(pm *PyMsg) {
			cl.ExtLog.Error("from_date invalid", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "invalid from date format, use MM/DD/YYYY")
		},
		"TO_DATE_INVALID": func(pm *PyMsg) {
			cl.ExtLog.Error("to_date invalid", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "invalid to date format, use MM/DD/YYYY")
		},
		"INVALID_CHAT_NAME": func(pm *PyMsg) {
			cl.ExtLog.Error("invalid chat name",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(3, fmt.Sprintf("invalid chat name: %s",
				pm.Details["name"]))
		},
		"INVITE_LINK_NOT_SUPPORTED": func(pm *PyMsg) {
			cl.ExtLog.Error("invite link not supported yet")
			_ = cl.UserLog(3, "invite link not supported yet")
		},
		"INVALID_USERNAME": func(pm *PyMsg) {
			cl.ExtLog.Error("invalid username",
				zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "invalid username")
		},
	}
	onOut := ComposeOnOut(cl.defaultPyOutHandlers, extraOut)
	onErr := ComposeOnErr(cl.defaultPyErrHandlers, extraErr)

	if err := runPyWithStreaming(cl.cfg.VenvPath, args, onOut, onErr); err != nil {
		return err
	}

	return nil
}

func (cl *Client) PrintDialogs(req *PrintDialogsRequest, validate bool) error {
	if cl.UserLogF == nil {
		cl.ExtLog.Error("no user log function to set")
		return errors.New("no user log function set")
	}
	if validate {
		if err := validator.New().Struct(req); err != nil {
			cl.ExtLog.Error(
				"validating print dialogs request failed",
				zap.Error(err),
			)
			return err
		}
	}

	args := []string{cl.cfg.ScriptsPath + "/print_dialogs.py"}
	args = append(args, "../"+cl.cfg.Session)

	if req.Limit > 0 {
		args = append(args, "--limit", strconv.Itoa(req.Limit))
	}
	if req.Output != "" {
		args = append(args, "--output", req.Output)
	}

	if err := runPyWithStreaming(cl.cfg.VenvPath, args,
		ComposeOnOut(cl.defaultPyOutHandlers, nil),
		ComposeOnErr(cl.defaultPyErrHandlers, nil)); err != nil {
		return err
	}

	return nil
}
