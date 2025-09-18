package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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

type Request interface {
	Validate() error
}

type GetMembersRequest struct {
	// ChatID is either a username(t.me/chat, chat, @chat),
	// a chatID (not peerID), or invite link. Required
	//
	// TODO: invite link not supported yet.
	ChatID string

	// Limit is the maximum number of members to return.
	// The default is 1000, and the maximum is 50,000.
	Limit int

	// Output is the path to the CSV file where
	// results will be saved. The default is
	// "./get-members-<timestamp>.csv".
	Output string

	// ParseFromMessages parses users/bots from messages if true.
	// Default is false.
	ParseFromMessages bool

	// MessagesLimit is the number of messages to parse
	// if ParseFromMessages is true. The default is 10,
	// and the maximum is 5000.
	MessagesLimit int

	// TODO: not implemented yet.
	ExcludeBots bool

	// ParseBio parses users' bio.
	// This may slow down the process. Default is false.
	ParseBio bool

	// AddAdditionalInfo adds additional information about users,
	// such as bio, premium, scam flag, etc.
	// Default is false.
	AddAdditionalInfo bool

	// AutoJoin automatically joins the chat if true.
	//
	// TODO: not implemented yet
	AutoJoin bool
}

func (req *GetMembersRequest) Validate() error {
	req.ChatID = strings.TrimSpace(req.ChatID)
	if req.ChatID == "" {
		return errors.New("ChatID is required")
	}
	if req.Limit < 0 || req.Limit > 50000 {
		return errors.New("Limit must be between 1 and 50,000")
	}
	if req.ParseFromMessages && (req.MessagesLimit < 0 || req.MessagesLimit > 5000) {
		return errors.New("MessagesLimit must be between 1 and 5000")
	}
	req.Output = strings.TrimSpace(req.Output)
	return nil
}

type GetChatStatsRequest struct {
	// ChatID is either a username(t.me/chat, chat, @chat),
	// a chatID (not peerID), or invite link. Required
	//
	// TODO: invite link not supported yet.
	ChatID string

	// MessagesLimit is the number of messages to parse
	// The default is 0 (all messages), no max value
	MessagesLimit int

	// Output is the path to the CSV file where
	// results will be saved. The default is
	// "./get-chat-statistics-<timestamp>.csv".
	Output string
}

func (req *GetChatStatsRequest) Validate() error {
	req.ChatID = strings.TrimSpace(req.ChatID)
	if req.ChatID == "" {
		return errors.New("ChatID is required")
	}
	if req.MessagesLimit < 0 {
		return errors.New("MessagesLimit must be greater or equal to 0")
	}
	req.Output = strings.TrimSpace(req.Output)
	return nil
}

type SearchMessagesRequest struct {
	// ChatID is either a username(t.me/chat, chat, @chat),
	// a chatID (not peerID), or invite link. Required
	//
	// TODO: invite link not supported yet.
	ChatID string

	// Username is a username(t.me/user, user, @user)
	// Required
	Username string

	// Output is the path to the CSV file where
	// results will be saved. The default is
	// "./search-messages-<username>-<timestamp>.csv".
	Output string

	// FromDate is the start date in MM/DD/YYYY format.
	FromDate string

	// ToDate is the end date in MM/DD/YYYY format.
	ToDate string
}

func (req *SearchMessagesRequest) Validate() error {
	req.ChatID = strings.TrimSpace(req.ChatID)
	if req.ChatID == "" {
		return errors.New("ChatID is required")
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return errors.New("Username is required")
	}
	if req.FromDate == "" {
		return errors.New("FromDate is required")
	}
	if req.ToDate == "" {
		return errors.New("ToDate is required")
	}
	req.Output = strings.TrimSpace(req.Output)
	return nil
}

// GetMembers get members of a group/channel if possible
func (cl *Client) GetMembers(req *GetMembersRequest, validate bool) error {
	if cl.UserLogF == nil {
		cl.ExtLog.Error("no user log function to set")
		return errors.New("no user log function set")
	}
	if validate {
		if err := req.Validate(); err != nil {
			cl.ExtLog.Warn("validating get members request failed", zap.Error(err))
			return err
		}
	}

	args := []string{cl.cfg.ScriptsPath + "/get_members.py"}
	args = append(args, req.ChatID)

	if req.Limit > 0 {
		args = append(args, "--limit", strconv.Itoa(req.Limit))
	}
	if req.Output != "" {
		args = append(args, "--output", req.Output)
	}
	if req.ParseFromMessages {
		args = append(args, "--parse-from-messages")
		if req.MessagesLimit > 0 {
			args = append(args, "--messages-limit", strconv.Itoa(req.MessagesLimit))
		}
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
		_ = cl.UserLog(3, "something went wrong")
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
			cl.ExtLog.Warn("validating get chat stats request failed", zap.Error(err))
			return err
		}
	}

	args := []string{cl.cfg.ScriptsPath + "/get_chat_statistic.py"}
	args = append(args, req.ChatID)

	if req.MessagesLimit > 0 {
		args = append(args, "--history-limit", strconv.Itoa(req.MessagesLimit))
	}
	if req.Output != "" {
		args = append(args, "--output", req.Output)
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
		_ = cl.UserLog(3, "something went wrong")
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
			cl.ExtLog.Warn("validating search messages request failed", zap.Error(err))
			return err
		}
	}

	args := []string{cl.cfg.ScriptsPath + "/search_messages.py"}
	args = append(args, req.ChatID)
	args = append(args, req.Username)

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
		_ = cl.UserLog(3, "something went wrong")
		return err
	}

	return nil
}
