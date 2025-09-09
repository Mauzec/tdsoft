package client

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

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

	cmd := exec.Command("../../.venv/bin/python3", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get members: %w: %s", err, out)
	}
	return nil
}
