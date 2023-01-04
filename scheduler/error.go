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

type RepositoryErrorKind string

const (
	AlreadyCancelled  RepositoryErrorKind = "already_cancelled"
	AlreadyDone       RepositoryErrorKind = "already_done"
	AlreadyDispatched RepositoryErrorKind = "already_dispatched"
	Empty             RepositoryErrorKind = "empty"
	NotDispatched     RepositoryErrorKind = "not_dispatched"
	IdNotFound        RepositoryErrorKind = "id_not_found"
)

type RepositoryError struct {
	Id   string
	Kind RepositoryErrorKind
	Raw  error
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf(
		"error: kind = %s, id = %s, raw error = %+v",
		e.Kind,
		e.Id,
		e.Raw,
	)
}

func IsRepositoryErr(err error, kind RepositoryErrorKind) bool {
	repoErr, ok := err.(*RepositoryError)
	if !ok {
		return false
	}
	return repoErr.Kind == kind
}

func IsAlreadyCancelled(err error) bool {
	return IsRepositoryErr(err, AlreadyCancelled)
}
func IsAlreadyDone(err error) bool {
	return IsRepositoryErr(err, AlreadyDone)
}
func IsAlreadyDispatched(err error) bool {
	return IsRepositoryErr(err, AlreadyDispatched)
}
func IsEmpty(err error) bool {
	return IsRepositoryErr(err, Empty)
}
func IsNotDispatched(err error) bool {
	return IsRepositoryErr(err, NotDispatched)
}
func IsIdNotFound(err error) bool {
	return IsRepositoryErr(err, IdNotFound)
}
