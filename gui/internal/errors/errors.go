package errors

import "errors"

var (
	ErrNeedAuth                  = errors.New("need auth")
	ErrExtendedLoggerNotProvided = errors.New("no extended logger provided")
	ErrCreatorPingError          = errors.New("ping to creator server failed")
	ErrCreatorWaitTimeout        = errors.New("timeout waiting for creator server")
	ErrPasswordNeeded            = errors.New("password needed")
	ErrSystemError               = errors.New("system error")
)

var (
	ErrOnlyDigitsAllowed       = errors.New("only digits are allowed")
	ErrFieldRequired           = errors.New("field is required")
	ErrNumericConversionFailed = errors.New("conversion failed")
)
