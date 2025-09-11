package client

import (
	"errors"
	"fmt"
	"strings"
)

// TODO: add specific messages about errors/info that can be handled in UI

type OutHandler func(string, *PyMsg)
type ErrHandler func(*PyMsg)

var DefaultOutHandlers = map[string]OutHandler{
	"FLOOD_WAIT": func(t string, pm *PyMsg) {
		fmt.Printf("[%s] flood wait: %s seconds\n",
			t, pm.Details["seconds"])
	},
	"CSV_FLUSH_ERROR": func(t string, pm *PyMsg) {
		fmt.Printf("[%s] %s\n", t, pm.Message)
	},
}
var DefaultErrHandlers = map[string]ErrHandler{
	"UNCAUGHT_ERROR": func(pm *PyMsg) {
		fmt.Printf("[ERROR] uncaught error: %s\n", pm.Details["error"])
	},
	"TASK_CANCELLED": func(pm *PyMsg) {
		fmt.Printf("[ERROR] task cancelled: %s\n", pm.Details["message"])
	},
	"RPC_ERROR": func(pm *PyMsg) {
		fmt.Printf("[ERROR] rpc error: {code: %v, message: %v}\n",
			pm.Details["code"], pm.Details["message"])
	},
	"UNEXPECTED_ERROR": func(pm *PyMsg) {
		fmt.Printf("[ERROR] unexpected error: %s\n", pm.Details["error"])
	},
}

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
			fmt.Printf("[%s] fetched %v members\n",
				t, pm.Details["total"])
		},
		"MEMBERS_FROM_MESSAGES_FETCHED": func(t string, pm *PyMsg) {
			fmt.Printf("[%s] fetched %v members from messages\n",
				t, pm.Details["total"])
		},
		"ALL_DONE": func(t string, pm *PyMsg) {
			fmt.Printf("[%s] all done, unique users total: %v\n",
				t, pm.Details["total"])
		},
	}
	extraErr := map[string]ErrHandler{
		"MEMBERS_LIMIT_TOO_HIGH": func(pm *PyMsg) {
			fmt.Printf("[ERROR] limit too high, got %v, max is %v\n",
				pm.Details["limit"], pm.Details["max"])
		},
		"MESSAGE_LIMIT_TOO_HIGH": func(pm *PyMsg) {
			fmt.Printf("[ERROR] messages limit too high, got %v, max is %v\n",
				pm.Details["limit"], pm.Details["max"])
		},
		"INVALID_CHAT_NAME": func(pm *PyMsg) {
			fmt.Printf("[ERROR] invalid chat name: %s\n",
				pm.Details["name"])
		},
		"INVITE_LINK_NOT_SUPPORTED": func(pm *PyMsg) {
			fmt.Printf("[ERROR] invite link not supported yet\n")
		},
	}
	onOut := ComposeOnOut(DefaultOutHandlers, extraOut)
	onErr := ComposeOnErr(DefaultErrHandlers, extraErr)

	if err := runPyWithStreaming(args, onOut, onErr); err != nil {
		return err
	}
	return nil
}
