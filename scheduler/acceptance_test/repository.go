package acceptancetest

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gokugen/scheduler/util"
	"github.com/ngicks/type-param-common/set"
	"github.com/ngicks/type-param-common/slice"
)

func randParam(scheduledAt time.Time) scheduler.TaskParam {
	return scheduler.TaskParam{
		ScheduledAt: scheduledAt,
		WorkId:      "foo",
		Param:       RandByte(),
		Priority:    int(rand.Int31()),
	}
}

// TestRepository is an exported acceptance test.
// Implementations may call this test with their own implementation in their test file.
func TestRepository(t *testing.T, repo scheduler.TaskRepository) {
	now := TruncatedNow()
	ids := slice.Deque[string]{}

	addFarFutureTask := addFarFutureTask(repo, now)

	t.Run("AddTask", func(t *testing.T) {
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
				t.Errorf("task does not inherits properties of param, expected = %+v, actual = %+v", param, task)
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
			ids.PushBack(task.Id)
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
	})

	t.Run("GetById", func(t *testing.T) {
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
			_, err := repo.GetById(scheduler.NeverExistenceId)
			AssertErrIdNotFound(t, scheduler.NeverExistenceId, err, true)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("update only non-zero param fields", func(t *testing.T) {
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

		})

		t.Run("trying to update id that is already unable-to-update state", func(t *testing.T) {
			testErrOnUpdateNonUpdatableTask(t, repo, addFarFutureTask, func(id string) error {
				_, err := repo.Update(id, scheduler.TaskParam{})
				return err
			})
		})

		t.Run("trying to update nonexistent id", func(t *testing.T) {
			_, err := repo.Update(scheduler.NeverExistenceId, scheduler.TaskParam{})
			AssertErrIdNotFound(t, scheduler.NeverExistenceId, err, true)
		})
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Run("cancel", func(t *testing.T) {
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

			now := TruncatedNow()
			got, err := repo.GetById(task.Id)
			if err != nil {
				t.Fatalf("GetById must not return error, id = %s, err = %+v", task.Id, err)
			}

			if got.CancelledAt == nil || !IsTimeNearNow(*got.CancelledAt, now) {
				t.Fatalf("CancelledAt is incorrect, expected after %s, but is %v", now, got.CancelledAt)
			}
		})

		t.Run("trying to cancel id that is already unable-to-update state", func(t *testing.T) {
			testErrOnUpdateNonUpdatableTask(
				t,
				repo,
				addFarFutureTask,
				func(id string) error {
					_, err := repo.Cancel(id)
					return err
				},
				skipAlreadyCancelled,
			)
		})

		t.Run("trying to cancel nonexistent id", func(t *testing.T) {
			_, err := repo.Cancel(scheduler.NeverExistenceId)
			AssertErrIdNotFound(t, scheduler.NeverExistenceId, err, true)
		})
	})

	t.Run("MarkAsDispatched", func(t *testing.T) {
		t.Run("trying to mark-as-dispatched id that is already unable-to-update state", func(t *testing.T) {
			testErrOnUpdateNonUpdatableTask(t, repo, addFarFutureTask, repo.MarkAsDispatched)
		})

		t.Run("trying to mark-as-dispatched nonexistent id", func(t *testing.T) {
			err := repo.MarkAsDispatched(scheduler.NeverExistenceId)
			AssertErrIdNotFound(t, scheduler.NeverExistenceId, err, true)
		})
	})

	t.Run("MarkAsDone", func(t *testing.T) {
		t.Run("only eligible state for MarkAsDone is marked-as-dispatched state", func(t *testing.T) {
			task := addFarFutureTask(t)
			err := repo.MarkAsDone(task.Id, nil)
			AssertErrNotDispatched(t, task.Id, err, true)
		})

		t.Run("MarkAsDone with non-nil error will set error string", func(t *testing.T) {
			task := addFarFutureTask(t)
			repo.MarkAsDispatched(task.Id)

			errorLabel := "mock error"
			repo.MarkAsDone(task.Id, errors.New(errorLabel))

			got, _ := repo.GetById(task.Id)

			if !strings.Contains(got.Err, errorLabel) {
				t.Fatalf(
					"task.Err must not be empty and contain error message = %s, but is %s",
					errorLabel, got.Err,
				)
			}
		})

		t.Run("trying to mark-as-done id that is already unable-to-update state", func(t *testing.T) {
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
		})

		t.Run("trying to mark-as-done nonexistent id", func(t *testing.T) {
			err := repo.MarkAsDone(scheduler.NeverExistenceId, nil)
			AssertErrIdNotFound(t, scheduler.NeverExistenceId, err, true)
		})
	})

	t.Run(
		"normal usecase, sequence of AddTask, Pop, MarkAsDispatched, and MarkAsFailed or MarkAsDone",
		func(t *testing.T) {
			repo.StartTimer()
			defer repo.StopTimer()

			for _, mockedWorkResult := range []error{nil, errors.New("mocked failed work error")} {
				now := TruncatedNow()
				old, err := repo.AddTask(randParam(now.Add(time.Millisecond)))
				if err != nil {
					t.Fatalf("must not return error: %+v", err)
				}
				<-repo.TimerChannel()
				task, err := repo.GetNext()
				if err != nil {
					t.Fatalf("must not return error: %+v", err)
				}
				if !task.Equal(old) {
					t.Fatalf("Pop returned the wrong task. expected = %+v, actual = %+v",
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

				repo.Cancel(task.Id)
			}
		},
	)

	t.Run("timer is updated if min element is affected", func(t *testing.T) {
		t.Run(
			"AddTask",
			func(t *testing.T) {
				repo.StartTimer()
				defer repo.StopTimer()

				now := util.DropMicros(TruncatedNow())
				task, _ := repo.AddTask(randParam(now.Add(500 * time.Millisecond)))
				if peeked, _ := repo.GetNext(); peeked.Id != task.Id {
					t.Fatalf("Peek did not change its min element"+
						" even if AddTask is called with min ScheduledAt. expected = %s, actual = %s",
						task.Id,
						peeked.Id,
					)
				}

				<-repo.TimerChannel()
				then := TruncatedNow()

				if sub := then.Sub(now); sub < 500*time.Millisecond {
					t.Fatalf("Timer is expected to emit after 500 milli secs but passed duration is %s", sub.String())
				}

				repo.Cancel(task.Id)

				now = TruncatedNow()
				task1, _ := repo.AddTask(randParam(now.Add(time.Second)))
				task2, _ := repo.AddTask(randParam(now.Add(500 * time.Millisecond)))

				<-repo.TimerChannel()
				then = TruncatedNow()

				if sub := then.Sub(now); sub < 500*time.Millisecond {
					t.Fatalf("Timer is expected to emit after 500 milli secs but passed duration is %s", sub.String())
				}

				repo.Cancel(task1.Id)
				repo.Cancel(task2.Id)
			},
		)

		t.Run(
			"Update",
			func(t *testing.T) {
				testUpdateTimer(
					t,
					repo,
					250*time.Millisecond, 250*time.Millisecond,
					func(id1, id2 string, now time.Time) error {
						var err error
						_, err = repo.Update(id1, scheduler.TaskParam{ScheduledAt: now.Add(500 * time.Millisecond)})
						if err != nil {
							return err
						}
						_, err = repo.Update(id2, scheduler.TaskParam{ScheduledAt: now.Add(750 * time.Millisecond)})
						if err != nil {
							return err
						}
						return nil
					},
				)
			},
		)

		t.Run("Cancel", func(t *testing.T) {
			testUpdateTimer(
				t,
				repo,
				250*time.Millisecond, 500*time.Millisecond,
				func(id1, id2 string, _ time.Time) error {
					_, err := repo.Cancel(id1)
					return err
				})
		})

		t.Run(
			"MarkAsDispatched",
			func(t *testing.T) {
				testUpdateTimer(
					t,
					repo,
					250*time.Millisecond, 500*time.Millisecond,
					func(id1, id2 string, _ time.Time) error {
						return repo.MarkAsDispatched(id1)
					},
				)
			},
		)
	})

	t.Run("Start and Stop timer", func(t *testing.T) {
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

		repo.Cancel(task.Id)
	})
}

func addFarFutureTask(repo scheduler.TaskRepository, now time.Time) func(t *testing.T) scheduler.Task {
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
			didError = true
		}
	}

	if didError {
		t.FailNow()
	}
}

// testUpdateTimer tests repository actions changes timer.
// It schedules 2 tasks, scheduled to be off in d1 and d2 from now.
// updateAction must update states so that the min element is scheduled after 500 milli secs from now.
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

	defer repo.Cancel(task1.Id)
	defer repo.Cancel(task2.Id)

	err := updateAction(task1.Id, task2.Id, now)
	if err != nil {
		t.Fatalf("updateAction must not return err = %+v", err)
	}

	<-repo.TimerChannel()
	then := TruncatedNow()

	if sub := then.Sub(now); sub < 500*time.Millisecond {
		t.Fatalf("Timer is expected to emit after 500 milli secs but passed duration is %s", sub.String())
	}
}
