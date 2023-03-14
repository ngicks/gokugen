package acceptancetest

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/helper"
	"github.com/stretchr/testify/require"
)

var revertDispatchedParams []scheduler.TaskParam

func init() {
	for i := 0; i < 5; i++ {
		// 0. scheduled
		// 1. dispatched
		// 2. done
		// 3. done with error
		// 4. cancelled

		// Tasks being dispatched indicates they've reached their scheduled time.
		// But implementations must not place the assumption.
		revertDispatchedParams = append(
			revertDispatchedParams,
			randParam(time.UnixMilli(rand.Int63n(253402300799999))),
		)
	}
}

func TestRepository_RevertDispatched(t *testing.T, repo scheduler.TaskRepository) {
	require := require.New(t)
	// fail fast: If caller breaks invariants
	// where repo must implements scheduler.DispatchedReverter,
	// then panic first.
	rev := repo.(scheduler.DispatchedReverter)

	tasks := addTasks(t, repo, revertDispatchedParams)
	defer func() {
		// clean up.
		for _, task := range tasks {
			_, _ = repo.Cancel(task.Id)
		}
	}()

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

	err := rev.RevertDispatched()
	if err != nil {
		t.Fatalf("RevertDispatched must not return error. err = %+v", err)
	}

	checkReverted := func(task scheduler.Task, mustReverted bool) {
		t.Helper()
		fetched, err := repo.GetById(task.Id)
		if err != nil {
			t.Fatalf("GetById returned an error. err = %+v", err)
		}

		before := task.DispatchedAt != nil
		after := fetched.DispatchedAt != nil
		// mustReverted indicates the task must be affected by the operation.
		if mustReverted && before == after && fetched.DispatchedAt != nil {
			t.Fatalf("The task is expected to be reverted to scheduled. task = %+v", fetched)
		}
		if !mustReverted && before != after && fetched.DispatchedAt == nil {
			t.Fatalf("The task is expected not to be reverted to scheduled. task = %+v", fetched)
		}
	}

	checkReverted(tasks[0], false)
	checkReverted(tasks[1], true)
	for _, task := range tasks[2:] {
		checkReverted(task, false)
	}

}
