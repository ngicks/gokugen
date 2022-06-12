package taskstorage_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	taskstorage "github.com/ngicks/gokugen/task_storage"
)

func storageTestSet(
	t *testing.T,
	prepare func() (
		repo *taskstorage.InMemoryRepo,
		registry *gokugen.WorkRegistry,
		sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
		doAllTasks func(),
		getTaskResults func() []error,
	),
) {
	t.Run("basic usage", func(t *testing.T) {
		repo, registry, sched, doAllTasks, _ := prepare()

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
		repo, registry, sched, _, _ := prepare()

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
		repo, registry, sched, doAllTasks, _ := prepare()

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

type testMode int

const (
	singleNodeMode testMode = iota
	multiNodeMode
)

type syncer interface {
	Sync(
		schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	) (rescheduled map[string]gokugen.Task, schedulingErr map[string]error, err error)
}

func testSync(t *testing.T, mode testMode) {
	var ts syncer
	var repo *taskstorage.InMemoryRepo
	var registry *gokugen.WorkRegistry
	var sched func(ctx gokugen.SchedulerContext) (gokugen.Task, error)
	var doAllTasks func()

	switch mode {
	case singleNodeMode:
		ts, repo, registry, sched, doAllTasks, _ = prepareSingle(true)
	case multiNodeMode:
		ts, repo, registry, sched, doAllTasks, _ = prepareMulti(true)
	}

	rescheduled, schedulingErr, err := ts.Sync(sched)
	if len(rescheduled) != 0 || len(schedulingErr) != 0 || err != nil {
		t.Fatalf(
			"len of rescheduled = %d, len of schedulingErr = %d, err = %v",
			len(rescheduled),
			len(schedulingErr),
			err,
		)
	}

	var called int64
	registry.Store("foobar", func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
		atomic.AddInt64(&called, 1)
		return nil
	})
	registry.Store("external", func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
		atomic.AddInt64(&called, 1)
		return nil
	})

	sched(taskstorage.WithWorkIdAndParam(gokugen.NewPlainContext(time.Now(), nil, nil), "foobar", nil))
	sched(taskstorage.WithWorkIdAndParam(gokugen.NewPlainContext(time.Now(), nil, nil), "foobar", nil))
	sched(taskstorage.WithWorkIdAndParam(gokugen.NewPlainContext(time.Now(), nil, nil), "foobar", nil))

	task1, task2, task3 := func() (taskstorage.TaskInfo, taskstorage.TaskInfo, taskstorage.TaskInfo) {
		tasks, err := repo.GetAll()
		if err != nil {
			t.Fatalf("should not be error: %v", err)
		}
		return tasks[0], tasks[1], tasks[2]
	}()

	// task1 is known, no one changed it.

	// task2 is known but changed externally.
	repo.Update(task2.Id, taskstorage.UpdateDiff{
		UpdateKey: taskstorage.UpdateKey{
			State: true,
		},
		Diff: taskstorage.TaskInfo{
			State: taskstorage.Working,
		},
	})

	// task3 will be changed later.

	// unknown and must be rescheduled.
	repo.Insert(taskstorage.TaskInfo{
		WorkId:        "external",
		ScheduledTime: time.Now(),
		State:         taskstorage.Initialized,
	})

	// unknown and must **NOT** be rescheduled.
	repo.Insert(taskstorage.TaskInfo{
		WorkId:        "external",
		ScheduledTime: time.Now(),
		State:         taskstorage.Working,
	})

	// unknown work id
	repo.Insert(taskstorage.TaskInfo{
		WorkId:        "baz?",
		ScheduledTime: time.Now(),
		State:         taskstorage.Initialized,
	})

	rescheduled, schedErr, err := ts.Sync(sched)
	if err != nil {
		t.Fatalf("must not be err: %v", err)
	}

	if len(rescheduled) != 1 {
		t.Fatalf("rescheduled must be 1: %v", rescheduled)
	}
	if len(schedErr) != 1 {
		t.Fatalf("schedErr must be 1: %d", len(schedErr))
	}

	repo.Update(task3.Id, taskstorage.UpdateDiff{
		UpdateKey: taskstorage.UpdateKey{
			State: true,
		},
		Diff: taskstorage.TaskInfo{
			State: taskstorage.Working,
		},
	})

	rescheduled, schedErr, err = ts.Sync(sched)
	if err != nil {
		t.Fatalf("must not be err: %v", err)
	}

	if len(rescheduled) != 0 {
		t.Fatalf("rescheduled must be 0: %d", len(rescheduled))
	}
	if len(schedErr) != 0 {
		t.Fatalf("schedErr must be 1: %d", len(schedErr))
	}

	doAllTasks()

	currentCallCount := atomic.LoadInt64(&called)
	var violated bool
	switch mode {
	case singleNodeMode:
		// In single node mode, it ignores Working state. So call count is 4.
		violated = currentCallCount != 4
	case multiNodeMode:
		// In multi node mode, it respects Working state. So call count is 2.
		violated = currentCallCount != 2
	}
	if violated {
		t.Fatalf("call count is %d", currentCallCount)
	}

	info, err := repo.GetById(task1.Id)
	if err != nil {
		t.Fatalf("must not be err: %v", err)
	}
	if info.State != taskstorage.Done {
		t.Fatalf("work is not done correctly: %s", info.State)
	}
}

func TestSingleNodeSync(t *testing.T) {
	t.Run("Sync: single node", func(t *testing.T) {
		testSync(t, singleNodeMode)
	})

	t.Run("Sync: multi node", func(t *testing.T) {
		testSync(t, multiNodeMode)
	})
}
