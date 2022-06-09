package taskstorage_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	taskstorage "github.com/ngicks/gokugen/task_storage"
)

var _ gokugen.Task = &fakeTask{}

type fakeTask struct {
	ctx gokugen.SchedulerContext
}

func (t *fakeTask) Cancel() (cancelled bool) {
	return true
}
func (t *fakeTask) GetScheduledTime() time.Time {
	return t.ctx.ScheduledTime()
}
func (t *fakeTask) IsCancelled() (cancelled bool) {
	return
}
func (t *fakeTask) IsDone() (done bool) {
	return
}

func buildTaskStorage() (
	singleNode *taskstorage.SingleNodeTaskStorage,
	multiNode *taskstorage.MultiNodeTaskStorage,
	repo *taskstorage.InMemoryRepo,
	registry *gokugen.WorkRegistry,
) {
	repo = taskstorage.NewInMemoryRepo()
	registry = gokugen.NewWorkRegistry()
	singleNode = taskstorage.NewSingleNodeTaskStorage(
		repo,
		func(ti taskstorage.TaskInfo) bool { return true },
		registry,
	)
	multiNode = taskstorage.NewMultiNodeTaskStorage(
		repo,
		func(ti taskstorage.TaskInfo) bool { return true },
		registry,
	)
	return
}

func prepare(
	ts interface {
		Middleware(freeParam bool) []gokugen.MiddlewareFunc
	},
	freeParam bool,
) (
	sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	doAllTasks func(),
) {
	mws := ts.Middleware(freeParam)

	workMu := sync.Mutex{}
	works := make([]taskstorage.WorkFn, 0)

	doAllTasks = func() {
		workMu.Lock()
		defer workMu.Unlock()
		for _, v := range works {
			v(make(<-chan struct{}), make(<-chan struct{}), time.Now())
		}
	}
	sched = func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		workMu.Lock()
		works = append(works, ctx.Work())
		workMu.Unlock()
		return &fakeTask{
			ctx: ctx,
		}, nil
	}
	for i := len(mws) - 1; i >= 0; i-- {
		sched = mws[i](sched)
	}
	return
}

func prepareSingle(freeParam bool) (
	ts *taskstorage.SingleNodeTaskStorage,
	repo *taskstorage.InMemoryRepo,
	registry *gokugen.WorkRegistry,
	sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	doAllTasks func(),
) {
	singleNode, _, repo, registry := buildTaskStorage()
	sched, doAllTasks = prepare(singleNode, freeParam)
	return
}

func storageTestSet(
	t *testing.T,
	prepare func() (
		repo *taskstorage.InMemoryRepo,
		registry *gokugen.WorkRegistry,
		sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
		doAllTasks func(),
	),
) {
	t.Run("basic usage", func(t *testing.T) {
		repo, registry, sched, doAllTasks := prepare()

		registry.Store("foobar", func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
			return nil
		})
		now := time.Now()
		task, err := sched(taskstorage.WithWorkIdAndParam(gokugen.NewPlainContext(now, nil, nil), "foobar", nil))
		if err != nil {
			t.Fatalf("must not be non nil error: %v", err)
		}
		if task.GetScheduledTime() != now {
			t.Fatalf(
				"scheduled time is modified: now=%s, stored in task=%s",
				now.Format(time.RFC3339Nano),
				task.GetScheduledTime().Format(time.RFC3339Nano),
			)
		}
		stored, err := repo.GetAll()
		if err != nil {
			t.Fatalf("must not be non nil error: %v", err)
		}
		if len(stored) == 0 {
			t.Fatalf("stored task must not be zero")
		}
		storedTask := stored[0]
		taskId := storedTask.Id
		if storedTask.WorkId != "foobar" {
			t.Fatalf("unmatched work id: %s", storedTask.WorkId)
		}
		if storedTask.ScheduledTime != now {
			t.Fatalf("unmatched scheduled time: %s", storedTask.ScheduledTime.Format(time.RFC3339Nano))
		}

		doAllTasks()

		storedInfoLater, err := repo.GetById(taskId)
		if err != nil {
			t.Fatalf("must not be non nil error: %v", err)
		}

		if storedInfoLater.State != taskstorage.Done {
			t.Fatalf("incorrect state: %s", storedInfoLater.State)
		}
	})

	t.Run("cancel marks data as cancelled inside repository", func(t *testing.T) {
		repo, registry, sched, _ := prepare()

		registry.Store("foobar", func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
			return nil
		})
		now := time.Now()
		task, _ := sched(taskstorage.WithWorkIdAndParam(gokugen.NewPlainContext(now, nil, nil), "foobar", nil))
		task.Cancel()

		stored, _ := repo.GetAll()
		storedTask := stored[0]

		if storedTask.State != taskstorage.Cancelled {
			t.Fatalf("wrong state: must be cancelled, but is %s", storedTask.State)
		}
	})

	t.Run("failed marks data as failed inside repository", func(t *testing.T) {
		repo, registry, sched, doAllTasks := prepare()

		registry.Store("foobar", func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
			return errors.New("mock error")
		})
		now := time.Now()
		sched(taskstorage.WithWorkIdAndParam(gokugen.NewPlainContext(now, nil, nil), "foobar", nil))

		doAllTasks()

		stored, _ := repo.GetAll()
		storedTask := stored[0]

		if storedTask.State != taskstorage.Failed {
			t.Fatalf("wrong state: must be cancelled, but is %s", storedTask.State)
		}
	})
}

func TestSingleNode(t *testing.T) {
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
			_, repo, registry, sched, doAllTasks = prepareSingle(paramLoad)
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
