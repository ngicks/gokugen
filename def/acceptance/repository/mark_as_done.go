package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/stretchr/testify/require"
)

func TestRepository_dispatched_tasks_can_be_marked_as_done(
	t *testing.T,
	repo def.Repository,
	debug bool,
) {
	t.Helper()

	eachState := CreateEachState(t, repo)
	testRepository_mark_as_done(t, repo, eachState, nil, debug)
	testRepository_mark_as_done_error(t, repo, eachState, nil, debug)

	eachState = CreateEachState(t, repo)
	testRepository_mark_as_done(t, repo, eachState, fakeErr, debug)
	testRepository_mark_as_done_error(t, repo, eachState, fakeErr, debug)
}

func testRepository_mark_as_done(
	t *testing.T, repo def.Repository,
	eachState EachStateTask,
	taskErr error,
	debug bool,
) {
	t.Helper()
	require := require.New(t)

	require.Zero(eachState.Dispatched.DoneAt)

	timeBeforeMarkingAsDone := time.Now().Truncate(time.Millisecond)
	err := repo.MarkAsDone(context.Background(), eachState.Dispatched.Id, taskErr)
	require.NoError(err)

	refetched, err := repo.GetById(context.Background(), eachState.Dispatched.Id)
	require.NoError(err)

	if taskErr != nil {
		require.Equal(taskErr.Error(), refetched.Err)
	}

	require.NotZero(refetched.DoneAt)
	require.GreaterOrEqual(
		refetched.DoneAt.Value().Compare(timeBeforeMarkingAsDone), 0,
		"DoneAt must be time value fetched at the time it is cancelled."+
			" DoneAt = %s, time fetched before cancellation = %s",
		refetched.DoneAt.Value(), timeBeforeMarkingAsDone,
	)

	assertTimeNormalized(t, refetched.DoneAt.Value())
}

func testRepository_mark_as_done_error(
	t *testing.T,
	repo def.Repository,
	eachState EachStateTask,
	taskErr error,
	debug bool,
) {
	var err error
	err = repo.MarkAsDone(context.Background(), def.NeverExistentId, taskErr)
	assertIsIdNotFound(t, err)

	err = repo.MarkAsDone(context.Background(), eachState.Scheduled.Id, taskErr)
	assertIsNotDispatched(t, err)

	err = repo.MarkAsDone(context.Background(), eachState.Cancelled.Id, taskErr)
	assertIsAlreadyCancelled(t, err)

	err = repo.MarkAsDone(context.Background(), eachState.Done.Id, taskErr)
	assertIsAlreadyDone(t, err)

	err = repo.MarkAsDone(context.Background(), eachState.Err.Id, taskErr)
	assertIsAlreadyDone(t, err)
}
