package acceptancetest

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gokugen/scheduler/util"
	"github.com/ngicks/type-param-common/set"
	"github.com/ngicks/type-param-common/slice"
)

func TestRepository_GetNext_on_empty_repo(t *testing.T, repo scheduler.TaskRepository) {
	_, err := repo.GetNext()
	AssertErrEmpty(t, "", err, true)
}

func TestRepository_AddTask(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	var ids slice.Stack[string]

	for i := 0; i < 100; i++ {
		param := randParam(now.AddDate(0, 0, i+5))
		timeCreatedAt := TruncatedNow()
		task, err := repo.AddTask(param)
		if err != nil {
			t.Fatalf("AddTask must not return error: %+v", err)
		}

		var didError bool

		if !IsTaskBasedOnParam(task, param) {
			didError = true
			t.Errorf(
				"task does not inherits properties of param, expected = %+v, actual = %+v",
				param, task,
			)
		}

		if task.Id == "" {
			didError = true
			t.Error("task.Id is empty")
		}

		if !IsTaskInitial(task, timeCreatedAt) {
			didError = true
			t.Errorf(
				"not correctly initialized = %+v, must be created around = %s",
				task,
				timeCreatedAt.Format(time.RFC3339Nano),
			)
		}

		if didError {
			t.FailNow()
		}
		ids.Push(task.Id)
	}

	idSet := set.New[string]()

	for _, v := range ids {
		idSet.Add(v)
	}

	if idSet.Len() != ids.Len() {
		t.Fatalf(
			"All ids must be unique but is not. id generated %d times, but unique ids = %d",
			ids.Len(),
			idSet.Len(),
		)
	}
}

func TestRepository_GetById(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	t.Run("set and fetch", func(t *testing.T) {
		task, err := repo.AddTask(randParam(now.AddDate(0, 1, 0)))
		if err != nil {
			t.Fatalf("AddTask must not return error: %+v", err)
		}
		got, err := repo.GetById(task.Id)
		if err != nil {
			t.Fatalf("GetById must not return error: %+v", err)
		}
		if !task.Equal(got) {
			t.Fatalf("must be equal: expected = %+v, actual = %+v", task, got)
		}
	})
	t.Run("trying to fetch nonexistent id", func(t *testing.T) {
		_, err := repo.GetById(scheduler.NeverExistentId)
		AssertErrIdNotFound(t, scheduler.NeverExistentId, err, true)
	})
}

func TestRepository_Update_only_non_zero_param_fields(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	task, err := repo.AddTask(randParam(now.AddDate(0, 2, 0)))
	if err != nil {
		t.Fatalf("AddTask must not return error: %+v", err)
	}

	old := task
	for _, param := range possibleParams {
		updateOk, err := repo.Update(task.Id, param.Param)
		if err != nil {
			t.Fatalf("Update must not return error: %+v", err)
		}
		if !updateOk {
			t.Fatalf("updated must be true, but returned false")
		}
		updated, _ := repo.GetById(task.Id)
		if old.Equal(updated) {
			t.Fatalf("must not be equal")
		}

		oldOtherThanUpdated := param.Filler(old)
		updatedOtherThanUpdated := param.Filler(updated)
		if !oldOtherThanUpdated.Equal(updatedOtherThanUpdated) {
			t.Fatalf(
				"update updated zero value, expected = %+v, actual = %+v",
				task,
				updated,
			)
		}

		old = updated
	}

	updateOk, err := repo.Update(task.Id, possibleParams[0].Param)
	if err != nil {
		t.Fatalf("Update must not return error: %+v", err)
	}
	if updateOk {
		t.Fatalf("updated must be false, but returned true.")
	}
}

func TestRepository_Update_error_on_non_updatable(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	addFarFutureTask := addFarFutureTask(repo, now)
	testErrOnUpdateNonUpdatableTask(t, repo, addFarFutureTask, func(id string) error {
		_, err := repo.Update(id, scheduler.TaskParam{WorkId: "yay-yay"})
		return err
	})
}

func TestRepository_Update_param_only_is_always_allowed(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	addFarFutureTask := addFarFutureTask(repo, now)
	{
		task := addFarFutureTask(t)
		err := repo.MarkAsDispatched(task.Id)
		if err != nil {
			t.Errorf("MarkAsDispatched must not return error: %+v", err)
		}

		updated, err := repo.Update(
			task.Id,
			scheduler.TaskParam{Meta: map[string]string{"foo": "bar"}},
		)
		if !updated {
			t.Errorf("must be updated. err = %+v", err)
		}
	}
	{
		task := addFarFutureTask(t)
		_, err := repo.Cancel(task.Id)
		if err != nil {
			t.Errorf("Cancel must not return error: %+v", err)
		}

		updated, err := repo.Update(
			task.Id,
			scheduler.TaskParam{Meta: map[string]string{"baz": "qux"}},
		)
		if !updated {
			t.Errorf("must be updated. err = %+v", err)
		}
	}
	{
		task := addFarFutureTask(t)
		err := repo.MarkAsDispatched(task.Id)
		if err != nil {
			t.Fatalf("MarkAsDispatched must not return error: %+v", err)
		}
		err = repo.MarkAsDone(task.Id, nil)
		if err != nil {
			t.Fatalf("MarkAsDone must not return error: %+v", err)
		}

		updated, err := repo.Update(
			task.Id,
			scheduler.TaskParam{Meta: map[string]string{"quux": "corge"}},
		)
		if !updated {
			t.Errorf("must be updated. err = %+v", err)
		}
	}
}

func TestRepository_Update_error_on_nonexistent(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	_, err := repo.Update(scheduler.NeverExistentId, scheduler.TaskParam{})
	AssertErrIdNotFound(t, scheduler.NeverExistentId, err, true)
}

func TestRepository_Cancel(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	addFarFutureTask := addFarFutureTask(repo, now)

	task := addFarFutureTask(t)
	cancelled, err := repo.Cancel(task.Id)
	if err != nil {
		t.Fatalf("Cancel must not return error = %+v", err)
	}
	if !cancelled {
		t.Fatalf("cancelled must be true but false.")
	}

	cancelled, err = repo.Cancel(task.Id)
	if err != nil {
		t.Fatalf("Cancel must not return error = %+v", err)
	}
	if cancelled {
		t.Fatalf("cancelling twice must return false cancelled but true.")
	}

	now = TruncatedNow()
	got, err := repo.GetById(task.Id)
	if err != nil {
		t.Fatalf("GetById must not return error, id = %s, err = %+v", task.Id, err)
	}

	if got.CancelledAt == nil || !IsTimeNearNow(*got.CancelledAt, now) {
		t.Fatalf(
			"CancelledAt is incorrect, expected after %s, but is %v",
			now, got.CancelledAt,
		)
	}
}

func TestRepository_Cancel_error_on_non_updatable(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	testErrOnUpdateNonUpdatableTask(
		t,
		repo,
		addFarFutureTask(repo, now),
		func(id string) error {
			_, err := repo.Cancel(id)
			return err
		},
		skipAlreadyCancelled,
	)
}

func TestRepository_Cancel_error_on_nonexistent(t *testing.T, repo scheduler.TaskRepository) {
	_, err := repo.Cancel(scheduler.NeverExistentId)
	AssertErrIdNotFound(t, scheduler.NeverExistentId, err, true)
}

func TestRepository_MarkAsDispatched_error_on_non_updatable(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	testErrOnUpdateNonUpdatableTask(t, repo, addFarFutureTask(repo, now), repo.MarkAsDispatched)
}

func TestRepository_MarkAsDispatched_error_on_nonexistent(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	err := repo.MarkAsDispatched(scheduler.NeverExistentId)
	AssertErrIdNotFound(t, scheduler.NeverExistentId, err, true)
}

func TestRepository_MarkAsDone_error_on_not_dispatched(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	task := addFarFutureTask(repo, now)(t)
	err := repo.MarkAsDone(task.Id, nil)
	AssertErrNotDispatched(t, task.Id, err, true)
}

func TestRepository_MarkAsDone_with_error(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	task := addFarFutureTask(repo, now)(t)

	err := repo.MarkAsDispatched(task.Id)
	if err != nil {
		t.Fatalf("Marking-as-dispatched failed with initial task, %+v", err)
	}

	errorLabel := "mock error"

	err = repo.MarkAsDone(task.Id, errors.New(errorLabel))
	if err != nil {
		t.Fatalf("Marking-as-done failed with dispatched task, %+v", err)
	}

	got, err := repo.GetById(task.Id)
	if err != nil {
		t.Fatalf("getting id failed, %+v", err)
	}

	if !strings.Contains(got.Err, errorLabel) {
		t.Fatalf(
			"task.Err must not be empty and contain error message = %s, but is %s",
			errorLabel, got.Err,
		)
	}
}

func TestRepository_MarkAsDone_error_on_non_updatable(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	addFarFutureTask := addFarFutureTask(repo, now)

	testErrOnUpdateNonUpdatableTask(
		t,
		repo,
		addFarFutureTask,
		func(id string) error {
			return repo.MarkAsDone(id, nil)
		},
		skipAlreadyDispatched,
	)
	testErrOnUpdateNonUpdatableTask(
		t,
		repo,
		addFarFutureTask,
		func(id string) error {
			return repo.MarkAsDone(id, errors.New("mocked error"))
		},
		skipAlreadyDispatched,
	)
}

func TestRepository_MarkAsDone_error_on_nonexistent(t *testing.T, repo scheduler.TaskRepository, now time.Time) {
	err := repo.MarkAsDone(scheduler.NeverExistentId, nil)
	AssertErrIdNotFound(t, scheduler.NeverExistentId, err, true)
}

func TestRepository_normal_usecase(t *testing.T, repo scheduler.TaskRepository) {
	repo.StartTimer()
	defer repo.StopTimer()

	for _, mockedWorkResult := range []error{nil, errors.New("mocked failed work error")} {
		now := TruncatedNow()
		old, err := repo.AddTask(randParam(now.Add(time.Millisecond)))
		if err != nil {
			t.Fatalf("must not return error: %+v", err)
		}

		timerChan := repo.TimerChannel()
		if timerChan == nil {
			t.Fatalf("returned timer channel is nil")
		}

		<-timerChan

		task, err := repo.GetNext()
		if err != nil {
			t.Fatalf("must not return error: %+v", err)
		}
		if !task.Equal(old) {
			t.Fatalf("GetNext returned the wrong task. expected = %+v, actual = %+v",
				old,
				task,
			)
		}
		// A consumer calls GetById to allow last second update.
		task, err = repo.GetById(task.Id)
		if err != nil {
			t.Fatalf("must not return error: %+v", err)
		}
		if !task.Equal(old) {
			t.Fatalf("Pop returned the wrong task. expected = %+v, actual = %+v",
				old,
				task,
			)
		}

		err = repo.MarkAsDispatched(task.Id)
		if err != nil {
			t.Fatalf("must not return error: %+v", err)
		}
		peeked, _ := repo.GetById(task.Id)
		if peeked.DispatchedAt == nil {
			t.Fatal("incorrect update, expected non nil DispatchedAt but is nil")
		}

		err = repo.MarkAsDone(task.Id, mockedWorkResult)
		if err != nil {
			t.Fatalf("must not return error: %+v", err)
		}

		peeked, _ = repo.GetById(task.Id)
		if peeked.DoneAt == nil {
			t.Fatal("incorrect update, expected non nil DoneAt but is nil")
		}
		if mockedWorkResult != nil && peeked.Err == "" {
			t.Fatal("incorrect update, expected non empty Err but is empty")
		}

		_, _ = repo.Cancel(task.Id)
	}
}

func TestRepository_timer_update_AddTask(t *testing.T, repo scheduler.TaskRepository) {
	repo.StartTimer()
	defer repo.StopTimer()

	now := util.DropMicros(TruncatedNow())
	task, _ := repo.AddTask(randParam(now.Add(500 * time.Millisecond)))
	if peeked, _ := repo.GetNext(); peeked.Id != task.Id {
		t.Fatalf("GetNext did not change its min element even if "+
			"AddTask is called with min ScheduledAt. expected = %s, actual = %s",
			task.Id,
			peeked.Id,
		)
	}

	<-repo.TimerChannel()
	then := TruncatedNow()

	if sub := then.Sub(now); sub < 500*time.Millisecond {
		t.Fatalf(
			"Timer is expected to emit after 500 milli secs but passed duration is %s",
			sub.String(),
		)
	}

	_, _ = repo.Cancel(task.Id)

	now = TruncatedNow()
	task1, _ := repo.AddTask(randParam(now.Add(time.Second)))
	task2, _ := repo.AddTask(randParam(now.Add(500 * time.Millisecond)))

	<-repo.TimerChannel()
	then = TruncatedNow()

	if sub := then.Sub(now); sub < 500*time.Millisecond {
		t.Fatalf(
			"Timer is expected to emit after 500 milli secs but passed duration is %s",
			sub.String(),
		)
	}

	_, _ = repo.Cancel(task1.Id)
	_, _ = repo.Cancel(task2.Id)

}

func TestRepository_timer_update_Update(t *testing.T, repo scheduler.TaskRepository) {
	testUpdateTimer(
		t,
		repo,
		250*time.Millisecond, 250*time.Millisecond,
		func(id1, id2 string, now time.Time) error {
			var err error
			_, err = repo.Update(
				id1,
				scheduler.TaskParam{ScheduledAt: now.Add(250 * time.Millisecond)},
			)
			if err != nil {
				return err
			}
			_, err = repo.Update(
				id2,
				scheduler.TaskParam{ScheduledAt: now.Add(500 * time.Millisecond)},
			)
			if err != nil {
				return err
			}
			return nil
		},
	)
}

func TestRepository_timer_update_Cancel(t *testing.T, repo scheduler.TaskRepository) {
	testUpdateTimer(
		t,
		repo,
		250*time.Millisecond, 500*time.Millisecond,
		func(id1, id2 string, _ time.Time) error {
			_, err := repo.Cancel(id1)
			return err
		},
	)
}

func TestRepository_timer_update_MarkAsDispatched(t *testing.T, repo scheduler.TaskRepository) {
	testUpdateTimer(
		t,
		repo,
		250*time.Millisecond, 500*time.Millisecond,
		func(id1, id2 string, _ time.Time) error {
			return repo.MarkAsDispatched(id1)
		},
	)
}

func TestRepository_timer(t *testing.T, repo scheduler.TaskRepository) {
	repo.StopTimer()

	now := TruncatedNow()
	task, _ := repo.AddTask(randParam(now.Add(500 * time.Millisecond)))

	select {
	case <-repo.TimerChannel():
		t.Fatal("timer must not emit at stopped state.")
	case <-time.After(500 * time.Millisecond):
	}

	repo.StartTimer()
	now = TruncatedNow()
	<-repo.TimerChannel()
	then := TruncatedNow()

	if sub := then.Sub(now); sub > time.Millisecond {
		t.Fatalf(
			"time between attaching channel and receiving must almost instance but is %s",
			sub.String(),
		)
	}

	_, _ = repo.Cancel(task.Id)
}
