package acceptancetest

import (
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
		errorf("err must be nil: %s", id)
		return true
	}
	repoErr, ok := err.(*scheduler.RepositoryError)
	if !ok {
		errorf("wrong error type. must be RepositoryError: %T", err)
		return true
	}
	if repoErr.Kind != kind {
		errorf("wrong error kind. must be %s, but is %s", kind, repoErr.Kind)
		return true
	}
	return false
}

func AssertErrIdNotFound(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.IdNotFound)
}

func AssertErrAlreadyDispatched(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.AlreadyDispatched)
}

func AssertErrAlreadyCancelled(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.AlreadyCancelled)
}

func AssertErrAlreadyDone(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.AlreadyDone)
}

func AssertErrNotDispatched(t *testing.T, id string, err error, fatal bool) (didErr bool) {
	t.Helper()
	return assertRepositoryErr(t, id, err, fatal, scheduler.NotDispatched)
}
