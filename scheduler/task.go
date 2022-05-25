package scheduler

import (
	"sync/atomic"
	"time"
)

// Task is simple set of data of
// scheduledTime and work and calling state.
//
// work will be called with cancel channel and scheduled time.
// cancelCh is closed when caller wants to quit the task immediately.
// It is adviced for a passed work to detect cancellation by the channel,
// if the work needs long time to complete.
type Task struct {
	scheduledTime time.Time
	work          func(cancelCh <-chan struct{}, scheduled time.Time)
	done          uint32
	cancelled     uint32
}

// NewTask creates a new Task instance.
// scheduledTime is scheduled time when work should be invoked.
// work is work of Task, this will be only called once.
func NewTask(scheduledTime time.Time, work func(cancelCh <-chan struct{}, scheduled time.Time)) *Task {
	return &Task{
		scheduledTime: scheduledTime,
		work:          work,
	}
}

func (t *Task) Do(cancelCh <-chan struct{}) {
	if t.work != nil && !t.IsCancelled() && atomic.CompareAndSwapUint32(&t.done, 0, 1) {
		t.work(cancelCh, t.scheduledTime)
	}
}

func (t *Task) GetScheduledTime() time.Time {
	return t.scheduledTime
}

func (t *Task) Cancel() (alreadyCancelled bool) {
	return !atomic.CompareAndSwapUint32(&t.cancelled, 0, 1)
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

func (t *TaskController) Cancel() (alreadyCancelled bool) {
	return t.t.Cancel()
}

func (t *TaskController) IsCancelled() bool {
	return t.t.IsCancelled()
}

func (t *TaskController) IsDone() bool {
	return t.t.IsDone()
}
