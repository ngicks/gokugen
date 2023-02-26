package acceptancetest

import (
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/type-param-common/slice"
)

func addFarFutureTask(
	repo scheduler.TaskRepository,
	now time.Time,
) func(t *testing.T) scheduler.Task {
	return func(t *testing.T) scheduler.Task {
		task, err := repo.AddTask(randParam(now.AddDate(0, 2, 0)))
		if err != nil {
			t.Fatalf("AddTask must not return error = %+v", err)
		}
		return task
	}
}

type updateNonUpdatableTaskOption string

var (
	skipAlreadyDispatched updateNonUpdatableTaskOption = "dispatched"
	skipAlreadyCancelled  updateNonUpdatableTaskOption = "cancelled"
	skipAlreadyDone       updateNonUpdatableTaskOption = "done"
)

func testErrOnUpdateNonUpdatableTask(
	t *testing.T,
	repo scheduler.TaskRepository,
	addFarFutureTask func(t *testing.T) scheduler.Task,
	updateAction func(id string) error,
	options ...updateNonUpdatableTaskOption,
) {
	t.Helper()
	didError := false
	// already dispatched
	if !slice.Has(options, skipAlreadyDispatched) {
		task := addFarFutureTask(t)
		err := repo.MarkAsDispatched(task.Id)
		if err != nil {
			t.Fatalf("MarkAsDispatched must not return error: %+v", err)
		}

		err = updateAction(task.Id)
		if AssertErrAlreadyDispatched(t, task.Id, err, false) {
			task, _ = repo.GetById(task.Id)
			t.Errorf("task = %+v", task)
			didError = true
		}
	}
	// already cancelled
	if !slice.Has(options, skipAlreadyCancelled) {
		task := addFarFutureTask(t)
		_, err := repo.Cancel(task.Id)
		if err != nil {
			t.Fatalf("Cancel must not return error: %+v", err)
		}

		err = updateAction(task.Id)
		if AssertErrAlreadyCancelled(t, task.Id, err, false) {
			task, _ = repo.GetById(task.Id)
			t.Errorf("task = %+v", task)
			didError = true
		}
	}
	// already done
	if !slice.Has(options, skipAlreadyDone) {
		task := addFarFutureTask(t)
		err := repo.MarkAsDispatched(task.Id)
		if err != nil {
			t.Fatalf("MarkAsDispatched must not return error: %+v", err)
		}
		err = repo.MarkAsDone(task.Id, nil)
		if err != nil {
			t.Fatalf("MarkAsDone must not return error: %+v", err)
		}

		err = updateAction(task.Id)
		if AssertErrAlreadyDone(t, task.Id, err, false) {
			task, _ = repo.GetById(task.Id)
			t.Errorf("task = %+v", task)
			didError = true
		}
	}

	if didError {
		t.FailNow()
	}
}

// testUpdateTimer tests repository actions changes timer.
// It schedules 2 tasks, scheduled to be off in d1 and d2 from now.
// updateAction must update states so that the min element is scheduled
// after 500 milli secs from now.
func testUpdateTimer(
	t *testing.T,
	repo scheduler.TaskRepository,
	d1, d2 time.Duration,
	updateAction func(id1, id2 string, now time.Time) error,
) {
	t.Helper()

	repo.StartTimer()
	defer repo.StopTimer()

	now := TruncatedNow()
	task1, _ := repo.AddTask(randParam(now.Add(d1)))
	task2, _ := repo.AddTask(randParam(now.Add(d2)))

	defer func() {
		_, _ = repo.Cancel(task1.Id)
		_, _ = repo.Cancel(task2.Id)
	}()

	err := updateAction(task1.Id, task2.Id, now)
	if err != nil {
		t.Fatalf("updateAction must not return err = %+v", err)
	}

	<-repo.TimerChannel()
	then := TruncatedNow()

	if sub := then.Sub(now); sub < 250*time.Millisecond {
		t.Fatalf(
			"Timer is expected to emit after 250 milli secs but passed duration is %s",
			sub.String(),
		)
	}
}
