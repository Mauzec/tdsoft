package utils

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	apperrors "github.com/mauzec/tdsoft/gui/internal/errors"
)

type ChatNameKind int

const (
	ChatNameEmpty ChatNameKind = iota
	ChatNameUsername
	ChatNameInviteLink
	ChatNameChatID
)

var (
	usernameRe = regexp.MustCompile("^@?([A-Za-z0-9_]{4,32})$")

	DefaultStructValidator = validator.New()
)

func ValidateUsername(username string) bool {
	username = strings.TrimSpace(username)
	if username == "" {
		return false
	}
	return usernameRe.MatchString(username)
}

// ValidateChatName determines whether the given string is a TG-link(invite or direct), or username, or chat Id.
// It can't determine if username, chat_id, or invite link is valid or not, just the format.
// There are only 'light' checks. Returns the [ChatNameKind] and a cleaned version of it,
// or ChatNameEmpty, "" if not valid.
func ValidateChatName(chat string) (ChatNameKind, string) {
	chat = strings.TrimSpace(chat)
	if chat == "" {
		return ChatNameEmpty, ""
	}

	// is chat id
	if chat[0] == '0' {
		return ChatNameEmpty, ""
	}
	firstSkip := false
	if chat[0] == '-' {
		chat = chat[1:]
		firstSkip = true
	}
	if chat == "" {
		return ChatNameEmpty, ""
	}
	if n, err := strconv.Atoi(chat); err == nil && n > 0 {
		return ChatNameChatID, chat
	}
	if firstSkip {
		return ChatNameEmpty, ""
	}

	// is username
	if ValidateUsername(chat) {
		return ChatNameUsername, chat
	}
	if chat[0] == '@' {
		return ChatNameEmpty, ""
	}

	// is link
	if !strings.Contains(chat, "://") {
		chat = "https://" + chat
	}
	url, err := url.Parse(chat)
	if err != nil {
		return ChatNameEmpty, ""
	}
	host := strings.ToLower(url.Hostname())
	if host == "" {
		return ChatNameEmpty, ""
	}
	host = strings.TrimPrefix(host, "www.")
	if host != "t.me" && host != "telegram.me" {
		return ChatNameEmpty, ""
	}
	path := strings.ToLower(strings.TrimPrefix(url.Path, "/"))
	if path == "" {
		return ChatNameEmpty, ""
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
		chat = chat[:len(chat)-1]
	}
	slashes := strings.Count(path, "/")
	if path[0] == '+' {
		token := path[1:]
		if token == "" || slashes > 0 {
			return ChatNameEmpty, ""
		}
		return ChatNameInviteLink, chat
	}
	if strings.HasPrefix(path, "joinchat/") {
		token := path[9:]
		if token == "" || slashes > 1 {
			return ChatNameEmpty, ""
		}
		return ChatNameInviteLink, chat
	}
	if slashes > 0 {
		return ChatNameEmpty, ""
	}
	if path[0] != '@' && ValidateUsername(path) {
		return ChatNameUsername, path
	}
	return ChatNameEmpty, ""
}

func ValidateAndGetNumeric(s string, minimum, maximum int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return -1, apperrors.ErrNumericConversionFailed
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1, apperrors.ErrOnlyDigitsAllowed
		}
	}
	if n, err := strconv.Atoi(s); err != nil {
		return -1, apperrors.ErrNumericConversionFailed
	} else if n < minimum || n > maximum {
		return -1, apperrors.ErrValidationFailed
	} else {
		return n, nil
	}
}

func ValidateTime(d *time.Time) bool {
	if d == nil {
		return false
	}
	if d.Year() <= 2000 {
		return false
	}
	if d.Month() > 12 || d.Month() < 1 || d.Day() < 1 || d.Day() > 31 {
		return false
	}
	return true
}
