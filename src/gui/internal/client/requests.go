package client

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// TODO: add specific messages about errors/info that can be handled in UI

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
		fmt.Printf("[LOG] %s: %s, %+v\n", env.Code, env.Message, env.Details)
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
		fmt.Printf("[ERROR] %s: %s, %+v\n", env.Code, env.Message, env.Details)
	}
}

type Request interface {
	Validate() error
}

type GetMembersRequest struct {
	// ChatID is either a username(t.me/user, user, @user),
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
	//
	// TODO: фdd a space to the path with the option.
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

	if req.ParseFromMessages && req.MessagesLimit < 0 || req.MessagesLimit > 5000 {
		return errors.New("MessagesLimit must be between 1 and 5000")

	}
	return nil
}

// GetMembers get members of a group/channel if possible
func (cl *Client) GetMembers(req *GetMembersRequest) error {
	if cl.userLogF == nil {
		return errors.New("no user log function set")
	}
	if err := req.Validate(); err != nil {
		return err
	}

	args := []string{"../get_members.py"}
	args = append(args, req.ChatID)

	if req.Limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Output != "" {
		args = append(args, "--output", req.Output)
	}
	if req.ParseFromMessages {
		args = append(args, "--parse-from-messages")
		if req.MessagesLimit > 0 {
			args = append(args, "--messages-limit", fmt.Sprintf("%d", req.MessagesLimit))
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
			cl.extLog.Info("fetched members",
				zap.Any("details", pm.Details))
			cl.UserLog(1,
				fmt.Sprintf("fetched %v members", pm.Details["total"]))

		},
		"MEMBERS_FROM_MESSAGES_FETCHED": func(t string, pm *PyMsg) {
			cl.extLog.Info("fetched members from messages",
				zap.Any("details", pm.Details))
			cl.UserLog(1,
				fmt.Sprintf("fetched %v members from messages",
					pm.Details["total"]))

		},
		"ALL_DONE": func(t string, pm *PyMsg) {
			cl.extLog.Info("all done", zap.Any("details", pm.Details))
			cl.UserLog(1, "all done")

		},
	}
	extraErr := map[string]ErrHandler{
		"MEMBERS_LIMIT_TOO_HIGH": func(pm *PyMsg) {
			cl.extLog.Error("limit too high",
				zap.Any("details", pm.Details))
			cl.UserLog(3, fmt.Sprintf("limit too high, got %v, max is %v",
				pm.Details["limit"], pm.Details["max"]))
		},
		"MESSAGE_LIMIT_TOO_HIGH": func(pm *PyMsg) {
			cl.extLog.Error("messages limit too high",
				zap.Any("details", pm.Details))
			cl.UserLog(3, fmt.Sprintf("messages limit too high, got %v, max is %v",
				pm.Details["limit"], pm.Details["max"]))
		},
		"INVALID_CHAT_NAME": func(pm *PyMsg) {
			cl.extLog.Error("invalid chat name",
				zap.Any("details", pm.Details))
			cl.UserLog(3, fmt.Sprintf("invalid chat name: %s",
				pm.Details["name"]))
		},
		"INVITE_LINK_NOT_SUPPORTED": func(pm *PyMsg) {
			cl.extLog.Error("invite link not supported yet")
			cl.UserLog(3, "invite link not supported yet")
		},
	}
	onOut := ComposeOnOut(cl.defaultPyOutHandlers, extraOut)
	onErr := ComposeOnErr(cl.defaultPyErrHandlers, extraErr)

	if err := runPyWithStreaming(args, onOut, onErr); err != nil {
		return err
	}
	return nil
}
