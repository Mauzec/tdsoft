package client

import "errors"

var (
	ErrNeedAuth = errors.New("need auth")

	ErrCreatorPingError = errors.New("ping to creator server failed")

	ErrPasswordNeeded = errors.New("password needed")
)
