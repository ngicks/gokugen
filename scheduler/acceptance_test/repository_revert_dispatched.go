package acceptancetest

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

var params []scheduler.TaskParam

func init() {
	for i := 0; i < 5; i++ {
		// 0. scheduled
		// 1. dispatched
		// 2. done
		// 3. done with error
		// 4. cancelled

		// Tasks being dispatched indicates they've reached their scheduled time.
		// But implementations must not place the assumption.
		params = append(params, randParam(time.UnixMilli(rand.Int63())))
	}
}

func TestRepository_RevertDispatched(t *testing.T, repo scheduler.TaskRepository) {
	rev := repo.(scheduler.DispatchedReverter)

	tasks := make([]scheduler.Task, len(params))
	for idx, p := range params {
		task, err := repo.AddTask(p)
		if err != nil {
			t.Fatalf("AddTask must not return error. err = %+v", err)
		}
		tasks[idx] = task
	}

	checkErr := func(f func() error) {
		t.Helper()
		if err := f(); err != nil {
			t.Fatalf("An unexpected error. err = %+v", err)
		}
	}
	// 0 = scheduled. Not yet dispatched.
	// 1 = dispatched
	checkErr(func() error { return repo.MarkAsDispatched(tasks[1].Id) })
	// 2 = done
	checkErr(func() error { return repo.MarkAsDispatched(tasks[2].Id) })
	checkErr(func() error { return repo.MarkAsDone(tasks[2].Id, nil) })
	// 3 = done with error
	checkErr(func() error { return repo.MarkAsDispatched(tasks[3].Id) })
	checkErr(func() error { return repo.MarkAsDone(tasks[3].Id, errors.New("faked error")) })
	// 4 = cancelled
	checkErr(func() error { _, err := repo.Cancel(tasks[4].Id); return err })

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

		if mustReverted && fetched.DispatchedAt != nil {
			t.Fatalf("The task is expected to be reverted to scheduled. task = %+v", fetched)
		}
		if !mustReverted && fetched.DispatchedAt == nil {
			t.Fatalf("The task is expected not to be reverted to scheduled. task = %+v", fetched)
		}
	}

	checkReverted(tasks[0], false)
	checkReverted(tasks[1], true)
	for _, task := range tasks[2:] {
		checkReverted(task, false)
	}

	for _, task := range tasks {
		_, _ = repo.Cancel(task.Id)
	}
}
