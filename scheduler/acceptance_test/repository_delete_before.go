package acceptancetest

import (
	"errors"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/helper"
	"github.com/stretchr/testify/require"
)

var deleteBeforeParams []scheduler.TaskParam

func init() {
	for i := 0; i < 5; i++ {
		// 0. scheduled
		// 1. dispatched
		// 2. done
		// 3. done with error
		// 4. cancelled

		// Tasks being dispatched indicates they've reached their scheduled time.
		// But implementations must not place the assumption.
		deleteBeforeParams = append(
			deleteBeforeParams,
			randParam(oneYearBefore),
		)
	}
}

func TestRepository_DeleteBefore(t *testing.T, repo scheduler.TaskRepository) {
	require := require.New(t)
	// fail fast: If caller breaks invariants
	// where repo must implements scheduler.BeforeDeleter,
	// then panic first.
	del := repo.(scheduler.BeforeDeleter)

	tasks := addTasks(t, repo, deleteBeforeParams)

	defer func() {
		// clean up.
		for _, task := range tasks {
			_, _ = repo.Cancel(task.Id)
		}
		del.DeleteBefore(oneYearBefore.Add(time.Second), false)
	}()

	beforeOperation := TruncatedNow()

	// 0 = scheduled. Not yet dispatched.
	// 1 = dispatched
	require.NoError(repo.MarkAsDispatched(tasks[1].Id))
	// 2 = done
	require.NoError(repo.MarkAsDispatched(tasks[2].Id))
	require.NoError(repo.MarkAsDone(tasks[2].Id, nil))
	// 3 = done with error
	require.NoError(repo.MarkAsDispatched(tasks[3].Id))
	require.NoError(repo.MarkAsDone(tasks[3].Id, errors.New("faked error")))
	// 4 = cancelled
	require.NoError(helper.ExtractError(repo.Cancel(tasks[4].Id)))

	deleted, err := del.DeleteBefore(oneYearBefore, false)
	require.NoError(err)
	require.Nil(deleted.Cancelled)
	require.Nil(deleted.Done)
	// make sure things are not removed.
	assertTasksExist(t, repo, tasks, true)

	deleted, err = del.DeleteBefore(oneYearBefore, true)
	require.NoError(err)
	require.NotNil(deleted.Cancelled)
	require.Len(deleted.Cancelled, 0)
	require.NotNil(deleted.Done)
	require.Len(deleted.Done, 0)
	assertTasksExist(t, repo, tasks, true)

	// might be close to done_at / cancelled_at.
	deleted, err = del.DeleteBefore(beforeOperation, true)
	require.NoError(err)
	require.NotNil(deleted.Cancelled)
	require.Len(deleted.Cancelled, 0)
	require.NotNil(deleted.Done)
	require.Len(deleted.Done, 0)
	assertTasksExist(t, repo, tasks, true)

	deleted, err = del.DeleteBefore(TruncatedNow().Add(time.Second), true)
	require.NoError(err)
	require.Len(deleted.Cancelled, 1, "unexpected = %+v", deleted)
	require.Len(deleted.Done, 2, "unexpected = %+v", deleted)
	assertTasksExist(t, repo, tasks[:2], true)
	assertTasksExist(t, repo, tasks[2:], false)
}
