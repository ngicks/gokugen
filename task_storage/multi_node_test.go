package taskstorage_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/impl/repository"
	taskstorage "github.com/ngicks/gokugen/task_storage"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

func prepareMulti(freeParam bool) (
	ts *taskstorage.MultiNodeTaskStorage,
	repo *repository.InMemoryRepo,
	registry *syncparam.Map[string, gokugen.WorkFnWParam],
	sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	doAllTasks func(),
	getTaskResults func() []resultSet,
) {
	_, ts, repo, registry = buildTaskStorage()
	sched, doAllTasks, getTaskResults = prepare(ts, freeParam)
	return
}

func TestMultiNode(t *testing.T) {
	prep := func(paramLoad bool) func() (
		repo *repository.InMemoryRepo,
		registry *syncparam.Map[string, gokugen.WorkFnWParam],
		sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
		doAllTasks func(),
		getTaskResults func() []resultSet,
	) {
		return func() (
			repo *repository.InMemoryRepo,
			registry *syncparam.Map[string, gokugen.WorkFnWParam],
			sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
			doAllTasks func(),
			getTaskResults func() []resultSet,
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
		registry.Store("foobar", func(taskCtx context.Context, scheduled time.Time, param any) (any, error) {
			atomic.AddInt64(&count, 1)
			return nil, nil
		})

		sched(
			gokugen.BuildContext(
				time.Now(),
				nil,
				nil,
				gokugen.WithParam(nil),
				gokugen.WithWorkId("foobar"),
			),
		)

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

		if !errors.Is(results[0].err, taskstorage.ErrOtherNodeWorkingOnTheTask) {
			t.Fatalf("wrong error type: %v", results[0])
		}
		if atomic.LoadInt64(&count) != 0 {
			t.Fatalf("internal work is called")
		}
	})
}
