package utils

import (
	"strconv"
	"strings"

	apperrors "github.com/mauzec/tdsoft/gui/internal/errors"
)

func PreValidateChatName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return apperrors.ErrFieldRequired
	}
	return nil
}

func ValidateAndGetNumeric(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1, apperrors.ErrOnlyDigitsAllowed
		}
	}
	if n, err := strconv.Atoi(s); err != nil {
		return -1, apperrors.ErrNumericConversionFailed
	} else {
		return n, nil
	}

}
