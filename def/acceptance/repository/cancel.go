package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/stretchr/testify/require"
)

func TestRepository_tasks_can_be_cancelled(t *testing.T, repo def.Repository, debug bool) {
	t.Helper()

	eachState := CreateEachState(t, repo)

	testRepository_cancel(t, repo, eachState, debug)
	testRepository_cancel_error(t, repo, eachState, debug)
}

func testRepository_cancel(t *testing.T, repo def.Repository, eachState EachStateTask, debug bool) {
	t.Helper()
	require := require.New(t)

	require.Zero(eachState.Scheduled.CancelledAt)

	timeBeforeCancellation := time.Now().Truncate(time.Millisecond)
	err := repo.Cancel(context.Background(), eachState.Scheduled.Id)
	require.NoError(err)

	refetched, err := repo.GetById(context.Background(), eachState.Scheduled.Id)
	require.NoError(err)

	require.NotZero(refetched.CancelledAt)
	require.GreaterOrEqual(
		refetched.CancelledAt.Value().Compare(timeBeforeCancellation), 0,
		"CancelledAt must be time value fetched at the time it is cancelled."+
			" CancelledAt = %s, time fetched before cancellation = %s",
		refetched.CancelledAt.Value(), timeBeforeCancellation,
	)

	assertTimeNormalized(t, refetched.CancelledAt.Value())
}

func testRepository_cancel_error(
	t *testing.T,
	repo def.Repository,
	eachState EachStateTask,
	debug bool,
) {

	var err error
	err = repo.Cancel(context.Background(), def.NeverExistentId)
	assertIsIdNotFound(t, err)

	err = repo.Cancel(context.Background(), eachState.Cancelled.Id)
	assertIsAlreadyCancelled(t, err)

	err = repo.Cancel(context.Background(), eachState.Dispatched.Id)
	assertIsAlreadyDispatched(t, err)

	err = repo.Cancel(context.Background(), eachState.Done.Id)
	assertIsAlreadyDone(t, err)

	err = repo.Cancel(context.Background(), eachState.Err.Id)
	assertIsAlreadyDone(t, err)
}
