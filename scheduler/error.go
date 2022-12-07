package scheduler

import (
	"errors"
	"fmt"
)

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrAlreadyEnded   = errors.New("already ended")
	ErrInvalidArg     = errors.New("invalid argument")
)

type LoopError struct {
	cancellerLoopErr error
	dispatchLoopErr  error
}

func (e LoopError) IsEmpty() bool {
	return e.cancellerLoopErr == nil && e.dispatchLoopErr == nil
}

func (e LoopError) Error() string {
	return fmt.Sprintf(
		"cancellerLoopErr: %s, dispatchLoopErr: %s",
		e.cancellerLoopErr,
		e.dispatchLoopErr,
	)
}
