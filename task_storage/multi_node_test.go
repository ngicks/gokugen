package taskstorage_test

import (
	"testing"

	"github.com/ngicks/gokugen"
	taskstorage "github.com/ngicks/gokugen/task_storage"
)

func prepareMulti(freeParam bool) (
	ts *taskstorage.MultiNodeTaskStorage,
	repo *taskstorage.InMemoryRepo,
	registry *gokugen.WorkRegistry,
	sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	doAllTasks func(),
) {
	_, multiNode, repo, registry := buildTaskStorage()
	sched, doAllTasks = prepare(multiNode, freeParam)
	return
}

func TestMultiNode(t *testing.T) {
	prep := func(paramLoad bool) func() (
		repo *taskstorage.InMemoryRepo,
		registry *gokugen.WorkRegistry,
		sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
		doAllTasks func(),
	) {
		return func() (
			repo *taskstorage.InMemoryRepo,
			registry *gokugen.WorkRegistry,
			sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
			doAllTasks func(),
		) {
			_, repo, registry, sched, doAllTasks = prepareMulti(paramLoad)
			return
		}
	}

	t.Run("no param load", func(t *testing.T) {
		storageTestSet(t, prep(false))
	})

	t.Run("param load", func(t *testing.T) {
		storageTestSet(t, prep(true))
	})
}
