package taskstorage_test

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/impl/repository"
	taskstorage "github.com/ngicks/gokugen/task_storage"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

var _ gokugen.Task = &fakeTask{}

type fakeTask struct {
	ctx gokugen.SchedulerContext
}

func (t *fakeTask) Cancel() (cancelled bool) {
	return true
}

func (t *fakeTask) CancelWithReason(err error) (cancelled bool) {
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
	repo *repository.InMemoryRepo,
	registry *syncparam.Map[string, gokugen.WorkFnWParam],
) {
	repo = repository.NewInMemoryRepo()
	registry = new(syncparam.Map[string, gokugen.WorkFnWParam])
	singleNode = taskstorage.NewSingleNodeTaskStorage(
		repo,
		func(ti taskstorage.TaskInfo) bool { return true },
		registry,
		nil,
	)
	multiNode = taskstorage.NewMultiNodeTaskStorage(
		repo,
		func(ti taskstorage.TaskInfo) bool { return true },
		registry,
	)
	return
}

type resultSet struct {
	retVal any
	err    error
}

func prepare(
	ts interface {
		Middleware(freeParam bool) []gokugen.MiddlewareFunc
	},
	freeParam bool,
) (
	sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	doAllTasks func(),
	getTaskResults func() []resultSet,
) {
	mws := ts.Middleware(freeParam)

	workMu := sync.Mutex{}
	works := make([]taskstorage.WorkFn, 0)

	taskResults := make([]resultSet, 0)
	doAllTasks = func() {
		workMu.Lock()
		defer workMu.Unlock()
		for _, v := range works {
			ret, err := v(context.TODO(), time.Now())
			taskResults = append(taskResults, resultSet{retVal: ret, err: err})
		}
	}
	getTaskResults = func() []resultSet {
		workMu.Lock()
		defer workMu.Unlock()
		cloned := make([]resultSet, len(taskResults))
		copy(cloned, taskResults)
		return cloned
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

// parepare SingleNodeTaskStorage and other instances.
func prepareSingle(freeParam bool) (
	ts *taskstorage.SingleNodeTaskStorage,
	repo *repository.InMemoryRepo,
	registry *syncparam.Map[string, gokugen.WorkFnWParam],
	sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	doAllTasks func(),
	getTaskResults func() []resultSet,
) {
	ts, _, repo, registry = buildTaskStorage()
	sched, doAllTasks, getTaskResults = prepare(ts, freeParam)
	return
}

func TestSingleNode(t *testing.T) {
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
			_, repo, registry, sched, doAllTasks, getTaskResults = prepareSingle(paramLoad)
			return
		}
	}

	t.Run("no param load", func(t *testing.T) {
		storageTestSet(t, prep(false))
	})

	t.Run("param load", func(t *testing.T) {
		storageTestSet(t, prep(true))
	})

	t.Run("param is freed after task storage if freeParam is set to true", func(t *testing.T) {
		_, repo, registry, sched, doAllTasks, _ := prepareSingle(true)

		registry.Store("foobar", func(ctx context.Context, scheduled time.Time, param any) (any, error) {
			return nil, nil
		})

		type exampleParam struct {
			Foo string
			Bar int
		}
		paramUsedInSched := new(exampleParam)
		paramStoredInRepo := new(exampleParam)

		var called int64
		runtime.SetFinalizer(paramUsedInSched, func(*exampleParam) {
			atomic.AddInt64(&called, 1)
		})

		_, _ = sched(
			gokugen.BuildContext(
				time.Now(),
				nil,
				nil,
				gokugen.WithParam(paramUsedInSched),
				gokugen.WithWorkId("foobar"),
			),
		)
		stored, _ := repo.GetAll()
		taskId := stored[0].Id
		// see comment below.
		//
		// paramInRepo := stored[0].Param
		repo.Update(taskId, taskstorage.UpdateDiff{
			UpdateKey: taskstorage.UpdateKey{
				Param: true,
			},
			Diff: taskstorage.TaskInfo{
				Param: paramStoredInRepo,
			},
		})

		doAllTasks()

		for i := 0; i < 100; i++ {
			runtime.GC()
			if atomic.LoadInt64(&called) == 1 {
				break
			}
		}

		if atomic.LoadInt64(&called) != 1 {
			t.Fatalf("param is not dropped.")
		}
		// Comment-in these lines to see `paramUsedInSched` | `paramInRepo` is now determine to be not reachable.
		// At least, the case fails if they are kept alive.
		//
		// runtime.KeepAlive(paramUsedInSched)
		// runtime.KeepAlive(paramInRepo)
		runtime.KeepAlive(paramStoredInRepo)
	})
}
