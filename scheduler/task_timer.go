package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/ngicks/gokugen/common"
)

// TaskTimer is a wrapper around the task min-heap and a timer channel.
// It manages timer to be always reset to a min task.
type TaskTimer struct {
	workingState
	mu     sync.Mutex
	q      TaskQueue
	getNow common.GetNow
	timer  common.ITimer
}

// NewTaskTimer creates TaskTimer.
// queueMax is max for tasks. Passing zero sets it unlimited.
//
// panic: If getNow or timerImpl is nil.
func NewTaskTimer(queueMax uint, getNow common.GetNow, timerImpl common.ITimer) *TaskTimer {
	if getNow == nil || timerImpl == nil {
		panic(
			fmt.Errorf(
				"%w: one or more of aruguments is nil. getNow is nil=[%t], timerImpl is nil=[%t]",
				ErrInvalidArg,
				getNow == nil,
				timerImpl == nil,
			),
		)
	}
	return &TaskTimer{
		q:      NewUnsafeQueue(queueMax),
		getNow: getNow,
		timer:  timerImpl,
	}
}

// Start starts TaskTimer,
// setting timer to min task if any.
func (f *TaskTimer) Start() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if min := f.q.Peek(); min != nil {
		f.timer.Reset(min.scheduledTime, f.getNow.GetNow())
	}
}

func (f *TaskTimer) Stop() {
	f.setWorking(false)
	f.timer.Stop()
}

// GetTimer returns timer channel
// that emits when a scheduled time of a min task is past.
func (f *TaskTimer) GetTimer() <-chan time.Time {
	return f.timer.GetChan()
}

// Push pushes *Task into underlying heap.
// After successful push, timer is updated to new min element
// unless one of shouldResetTimer is false.
func (f *TaskTimer) Push(task *Task, shouldResetTimer ...bool) error {
	shouldReset := true
	for _, reset := range shouldResetTimer {
		if !reset {
			shouldReset = false
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	var prevMin *Task
	if shouldReset {
		prevMin = f.q.Peek()
	}

	err := f.q.Push(task)
	if err != nil {
		return err
	}

	if shouldReset {
		if newMin := f.q.Peek(); prevMin == nil || newMin.scheduledTime.Before(prevMin.scheduledTime) {
			f.timer.Reset(newMin.scheduledTime, f.getNow.GetNow())
		}
	}
	return nil
}

func excludeCancelled(ele *Task) bool {
	if ele == nil {
		return false
	}
	return ele.IsCancelled()
}

// RemoveCancelled removes elements from underlying heap if they are cancelled.
// Heap is scanned in given range [start,end).
// Wider range is, longer it will hold lock. So range size must be chosen wisely.
func (f *TaskTimer) RemoveCancelled(start, end int) (removed []*Task) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.q.Exclude(excludeCancelled, start, end)
}

func (f *TaskTimer) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.q.Len()
}

// GetScheduledTask gets tasks whose scheduled time is earlier than given time t.
// Note that returned slice might be zero-length, because min task might be removed in racy `RemoveCancelled` call.
func (f *TaskTimer) GetScheduledTask(t time.Time) []*Task {
	f.mu.Lock()
	defer f.mu.Unlock()
	tasks := make([]*Task, 0)
	for {
		p := f.q.Pop(func(next *Task) bool {
			if next == nil {
				return false
			}
			return next.scheduledTime.Before(t) || next.scheduledTime.Equal(t)
		})
		if p == nil {
			break
		}
		tasks = append(tasks, p)
	}

	if p := f.q.Peek(); p != nil {
		f.timer.Reset(p.scheduledTime, f.getNow.GetNow())
	}
	return tasks
}

// Peek return min element without modifying underlying heap.
func (f *TaskTimer) Peek() *Task {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.q.Peek()
}
