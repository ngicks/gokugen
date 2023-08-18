package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/ngicks/gokugen/def"
	"github.com/stretchr/testify/require"
)

var (
	fakeErr = errors.New("fake error")
)

type eachStateTask struct {
	Scheduled, Cancelled, Dispatched, Done, Err def.Task
}

func createEachState(t *testing.T, repo def.Repository) eachStateTask {
	t.Helper()
	require := require.New(t)

	var eachState eachStateTask

	eachState.Scheduled = func() def.Task {
		task, err := repo.AddTask(context.Background(), initialParam)
		require.NoError(err)
		return task
	}()

	eachState.Cancelled = func() def.Task {
		task, err := repo.AddTask(context.Background(), initialParam)
		require.NoError(err)
		err = repo.Cancel(context.Background(), task.Id)
		require.NoError(err)
		task, err = repo.GetById(context.Background(), task.Id)
		require.NoError(err)
		return task
	}()

	eachState.Dispatched = func() def.Task {
		task, err := repo.AddTask(context.Background(), initialParam)
		require.NoError(err)
		err = repo.MarkAsDispatched(context.Background(), task.Id)
		require.NoError(err)
		task, err = repo.GetById(context.Background(), task.Id)
		require.NoError(err)
		return task
	}()

	eachState.Done = func() def.Task {
		task, err := repo.AddTask(context.Background(), initialParam)
		require.NoError(err)
		err = repo.MarkAsDispatched(context.Background(), task.Id)
		require.NoError(err)
		err = repo.MarkAsDone(context.Background(), task.Id, nil)
		require.NoError(err)
		task, err = repo.GetById(context.Background(), task.Id)
		require.NoError(err)
		return task
	}()

	eachState.Err = func() def.Task {
		task, err := repo.AddTask(context.Background(), initialParam)
		require.NoError(err)
		err = repo.MarkAsDispatched(context.Background(), task.Id)
		require.NoError(err)
		err = repo.MarkAsDone(context.Background(), task.Id, fakeErr)
		require.NoError(err)
		task, err = repo.GetById(context.Background(), task.Id)
		require.NoError(err)
		return task
	}()

	return eachState
}
