package scheduler

import (
	"context"
	"sync/atomic"
	"time"

	atomicparam "github.com/ngicks/type-param-common/sync-param/atomic-param"
)

type WorkFn = func(ctx context.Context, scheduled time.Time)

// Task is simple set of data of
// scheduledTime and work and calling state.
//
// work will be called with ctxCancelCh, taskCancelCh and scheduled time.
// ctxCancelCh is closed when the larger context is being torn down, needing all tasks to be aborted.
// taskCancelCh is closed when the task is cancelled by calling Close. Further computing is not advised.
// A passed work should make use of these channels if it would take long time.
type Task struct {
	mu            sync.Mutex
	scheduledTime time.Time
	work          WorkFn
	isDone        uint32
	isCancelled   uint32
	cancelCtx     atomicparam.Value[*context.CancelFunc]
}

// NewTask creates a new Task instance.
// scheduledTime is scheduled time when work should be invoked.
// work is work of Task, this will be only called once.
func NewTask(scheduledTime time.Time, work WorkFn) *Task {
	cancelCtx := atomicparam.NewValue[*context.CancelFunc]()
	cancelCtx.Store(nil)
	return &Task{
		scheduledTime: scheduledTime,
		work:          work,
		cancelCtx:     cancelCtx,
	}
}

func (t *Task) Do(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if t.IsCancelled() {
		return
	}
	if atomic.CompareAndSwapUint32(&t.isDone, 0, 1) && t.work != nil {
		t.cancelCtx.Store(&cancel)
		// forget cancel immediately in case where Task is held long time after it is done.
		defer t.cancelCtx.Store(nil)
		// in case of race condition.
		// Cancel might be called right between cancelFunc storage and above IsCancelled call.
		if t.IsCancelled() {
			return
		}
		select {
		case <-ctx.Done():
			// Fast path: ctx is already cancelled.
			return
		default:
		}
		t.work(innerCtx, t.scheduledTime)
	}
}

func (t *Task) GetScheduledTime() time.Time {
	return t.scheduledTime
}

func (t *Task) Cancel() (cancelled bool) {
	cancelled = atomic.CompareAndSwapUint32(&t.isCancelled, 0, 1)
	if cancel := *t.cancelCtx.Load(); cancel != nil {
		cancel()
	}
	return
}

func (t *Task) CancelWithReason(err error) (cancelled bool) {
	return t.Cancel()
}

func (t *Task) IsCancelled() bool {
	return atomic.LoadUint32(&t.isCancelled) == 1
}

func (t *Task) IsDone() bool {
	return atomic.LoadUint32(&t.isDone) == 1
}

// TaskController is a small wrapper around Task.
// Simply it removes Do method from Task and
// expose other methods as it delegates them to inner Task.
type TaskController struct {
	t *Task
}

func NewTaskController(t *Task) *TaskController {
	return &TaskController{
		t: t,
	}
}

func (t *TaskController) GetScheduledTime() time.Time {
	return t.t.GetScheduledTime()
}

func (t *TaskController) Cancel() (cancelled bool) {
	return t.t.Cancel()
}

func (t *TaskController) CancelWithReason(err error) (cancelled bool) {
	return t.Cancel()
}

func (t *TaskController) IsCancelled() bool {
	return t.t.IsCancelled()
}

func (t *TaskController) IsDone() bool {
	return t.t.IsDone()
}
