package def

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidTask       = errors.New("invalid task")
	ErrNotRegisteredMeta = errors.New("not a registered meta")
)

type RepositoryErrorKind string

const (
	AlreadyCancelled  RepositoryErrorKind = "already_cancelled"
	AlreadyDone       RepositoryErrorKind = "already_done"
	AlreadyDispatched RepositoryErrorKind = "already_dispatched"
	Exhausted         RepositoryErrorKind = "exhausted"
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
	if err == nil {
		return false
	}
	for {
		repoErr, ok := err.(*RepositoryError)
		if ok {
			return repoErr.Kind == kind
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if IsRepositoryErr(err, kind) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
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
func IsExhausted(err error) bool {
	return IsRepositoryErr(err, Exhausted)
}
func IsNotDispatched(err error) bool {
	return IsRepositoryErr(err, NotDispatched)
}
func IsIdNotFound(err error) bool {
	return IsRepositoryErr(err, IdNotFound)
}
