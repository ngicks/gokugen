package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/stretchr/testify/require"
)

func TestRepository_tasks_can_be_marked_as_dispatched(
	t *testing.T,
	repo def.Repository,
	debug bool,
) {
	t.Helper()

	eachState := CreateEachState(t, repo)

	testRepository_mark_as_dispatched(t, repo, eachState, debug)
	testRepository_mark_as_dispatched_error(t, repo, eachState, debug)
}

func testRepository_mark_as_dispatched(
	t *testing.T,
	repo def.Repository,
	eachState EachStateTask,
	debug bool,
) {
	t.Helper()
	require := require.New(t)

	require.Zero(eachState.Scheduled.DispatchedAt)

	timeBeforeDispatching := time.Now().Truncate(time.Millisecond)
	err := repo.MarkAsDispatched(context.Background(), eachState.Scheduled.Id)
	require.NoError(err)

	refetched, err := repo.GetById(context.Background(), eachState.Scheduled.Id)
	require.NoError(err)

	require.NotZero(refetched.DispatchedAt)
	require.GreaterOrEqual(
		refetched.DispatchedAt.Value().Compare(timeBeforeDispatching), 0,
		"DispatchedAt must be time value fetched at the time it is cancelled."+
			" DispatchedAt = %s, time fetched before cancellation = %s",
		refetched.DispatchedAt.Value(), timeBeforeDispatching,
	)

	assertTimeNormalized(t, refetched.DispatchedAt.Value())
}

func testRepository_mark_as_dispatched_error(
	t *testing.T,
	repo def.Repository,
	eachState EachStateTask,
	debug bool,
) {

	var err error
	err = repo.MarkAsDispatched(context.Background(), def.NeverExistentId)
	assertIsIdNotFound(t, err)

	err = repo.MarkAsDispatched(context.Background(), eachState.Cancelled.Id)
	assertIsAlreadyCancelled(t, err)

	err = repo.MarkAsDispatched(context.Background(), eachState.Dispatched.Id)
	assertIsAlreadyDispatched(t, err)

	err = repo.MarkAsDispatched(context.Background(), eachState.Done.Id)
	assertIsAlreadyDone(t, err)

	err = repo.MarkAsDispatched(context.Background(), eachState.Err.Id)
	assertIsAlreadyDone(t, err)
}
