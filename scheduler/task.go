package scheduler

import (
	"sync/atomic"
	"time"
)

type WorkFn = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time)

// Task is simple set of data of
// scheduledTime and work and calling state.
//
// work will be called with ctxCancelCh, taskCancelCh and scheduled time.
// ctxCancelCh is closed when the larger context is being torn down, needing all tasks to be aborted.
// taskCancelCh is closed when the task is cancelled by calling Close. Further computing is not advised.
// A passed work should make use of these channels if it would take long time.
type Task struct {
	scheduledTime time.Time
	work          WorkFn
	cancelCh      chan struct{}
	done          uint32
	cancelled     uint32
}

// NewTask creates a new Task instance.
// scheduledTime is scheduled time when work should be invoked.
// work is work of Task, this will be only called once.
func NewTask(scheduledTime time.Time, work WorkFn) *Task {
	return &Task{
		scheduledTime: scheduledTime,
		work:          work,
		cancelCh:      make(chan struct{}),
	}
}

func (t *Task) Do(ctxCancelCh <-chan struct{}) {
	if t.work != nil && !t.IsCancelled() && atomic.CompareAndSwapUint32(&t.done, 0, 1) {
		select {
		case <-ctxCancelCh:
			// Fast path: ctx is already cancelled.
			return
		default:
		}
		t.work(ctxCancelCh, t.cancelCh, t.scheduledTime)
	}
}

func (t *Task) GetScheduledTime() time.Time {
	return t.scheduledTime
}

func (t *Task) Cancel() (cancelled bool) {
	cancelled = atomic.CompareAndSwapUint32(&t.cancelled, 0, 1)
	if cancelled {
		close(t.cancelCh)
	}
	return
}

func (t *Task) CancelWithReason(err error) (cancelled bool) {
	return t.Cancel()
}

func (t *Task) IsCancelled() bool {
	return atomic.LoadUint32(&t.cancelled) == 1
}

func (t *Task) IsDone() bool {
	return atomic.LoadUint32(&t.done) == 1
}

// TaskController is a small wrapper around Task.
// Simply it removes Do method from Task.
// For other methods, it delegates to inner Task.
type TaskController struct {
	t *Task
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
