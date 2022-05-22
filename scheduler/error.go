package scheduler

import "errors"

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrAlreadyEnded   = errors.New("already ended")
	ErrInvalidArg     = errors.New("invalid argument")
)
