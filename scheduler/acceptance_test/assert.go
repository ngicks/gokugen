package acceptancetest

import (
	"errors"
	"testing"

	"github.com/ngicks/gokugen/scheduler"
)

func assertRepositoryErr(t *testing.T, id string, err error, fatal bool, kind scheduler.RepositoryErrorKind) (didErr bool) {
	t.Helper()

	var errorf func(format string, args ...any)
	if fatal {
		errorf = t.Fatalf
	} else {
		errorf = t.Errorf
	}

	if err == nil {
		errorf("err must not be nil: %s", id)
		return true
	}
	orgErr := err

	var repoErr *scheduler.RepositoryError
	var ok bool
	for {
		repoErr, ok = err.(*scheduler.RepositoryError)
		if ok {
			break
		}
		err = errors.Unwrap(err)
		if err == nil {
			errorf("wrong error type. must be a wrapped or unwrapped RepositoryError, but is %T", orgErr)
			return true
		}
	}
	if repoErr.Kind != kind {
		errorf("wrong error kind. must be %s, but is %s", kind, repoErr.Kind)
		return true
	}
	return false
}

func AssertErrAlreadyCancelled(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.AlreadyCancelled)
}

func AssertErrAlreadyDone(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.AlreadyDone)
}

func AssertErrAlreadyDispatched(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.AlreadyDispatched)
}

func AssertErrEmpty(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.Empty)
}

func AssertErrNotDispatched(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.NotDispatched)
}

func AssertErrIdNotFound(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.IdNotFound)
}
