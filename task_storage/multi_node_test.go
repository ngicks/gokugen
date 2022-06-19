package taskstorage_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	taskstorage "github.com/ngicks/gokugen/task_storage"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

func prepareMulti(freeParam bool) (
	ts *taskstorage.MultiNodeTaskStorage,
	repo *taskstorage.InMemoryRepo,
	registry *syncparam.Map[string, gokugen.WorkFnWParam],
	sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	doAllTasks func(),
	getTaskResults func() []error,
) {
	_, ts, repo, registry = buildTaskStorage()
	sched, doAllTasks, getTaskResults = prepare(ts, freeParam)
	return
}

func TestMultiNode(t *testing.T) {
	prep := func(paramLoad bool) func() (
		repo *taskstorage.InMemoryRepo,
		registry *syncparam.Map[string, gokugen.WorkFnWParam],
		sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
		doAllTasks func(),
		getTaskResults func() []error,
	) {
		return func() (
			repo *taskstorage.InMemoryRepo,
			registry *syncparam.Map[string, gokugen.WorkFnWParam],
			sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
			doAllTasks func(),
			getTaskResults func() []error,
		) {
			_, repo, registry, sched, doAllTasks, getTaskResults = prepareMulti(paramLoad)
			return
		}
	}

	t.Run("no param load", func(t *testing.T) {
		storageTestSet(t, prep(false))
	})

	t.Run("param load", func(t *testing.T) {
		storageTestSet(t, prep(true))
	})

	t.Run("error if already marked as working", func(t *testing.T) {
		_, repo, registry, sched, doAllTasks, getTaskResults := prepareMulti(false)

		var count int64
		registry.Store("foobar", func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
			atomic.AddInt64(&count, 1)
			return nil
		})

		sched(gokugen.WithWorkId(gokugen.WithParam(gokugen.NewPlainContext(time.Now(), nil, nil), nil), "foobar"))

		storedTasks, _ := repo.GetAll()
		task := storedTasks[0]

		err := repo.Update(task.Id, taskstorage.UpdateDiff{
			UpdateKey: taskstorage.UpdateKey{
				State: true,
			},
			Diff: taskstorage.TaskInfo{
				State: taskstorage.Working,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		doAllTasks()

		results := getTaskResults()

		if !errors.Is(results[0], taskstorage.ErrOtherNodeWorkingOnTheTask) {
			t.Fatalf("wrong error type: %v", results[0])
		}
		if atomic.LoadInt64(&count) != 0 {
			t.Fatalf("internal work is called")
		}
	})
}
