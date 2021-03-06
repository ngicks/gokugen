package scheduler

import (
	"context"
	"fmt"

	"github.com/ngicks/gommon"
	gopherpool "github.com/ngicks/gopher-pool"
)

// DispatchLoop waits for a TaskTimer to emit the timer signal,
// and then sends scheduled tasks to worker channel.
//
// Multiple calls of Start is allowed. But performance benefits are questionable.
type DispatchLoop struct {
	taskTimer *TaskTimer
	getNow    gommon.GetNower
}

// NewDispatchLoop creates DispatchLoop.
//
// panic: when one or more of arguments is nil.
func NewDispatchLoop(taskTimer *TaskTimer, getNow gommon.GetNower) *DispatchLoop {
	if taskTimer == nil || getNow == nil {
		panic(fmt.Errorf(
			"%w: one or more of aruguments is nil. taskTimer is nil=[%t], getNow is nil=[%t]",
			ErrInvalidArg,
			taskTimer == nil,
			getNow == nil,
		))
	}
	return &DispatchLoop{
		taskTimer: taskTimer,
		getNow:    getNow,
	}
}

func (l *DispatchLoop) PushTask(task *Task) error {
	return l.taskTimer.Push(task)
}

func (l *DispatchLoop) TaskLen() int {
	return l.taskTimer.Len()
}

// Start starts a dispatch loop.
// Start does not have reponsibility of starting the TaskTimer.
// A caller must ensure that the taskTimer is started.
// Calling multiple Start in different goroutines is allowed, but performance benefits are questionable.
// Cancelling ctx will end this Start loop, with returning nil.
//
// If one or more of arguments are nil, Start immediately returns ErrInvalidArg.
//
// panic: Closing workCh *before* cancelling ctx may cause panic.
func (l *DispatchLoop) Start(ctx context.Context, workCh chan<- gopherpool.WorkFn) error {
	if workCh == nil || ctx == nil {
		return ErrInvalidArg
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-l.taskTimer.GetTimer():
			next := l.taskTimer.GetScheduledTask(l.getNow.GetNow())
			if next == nil {
				continue
			}
			for _, w := range next {
				if ctx.Err() != nil {
					// race condition causes deadlock here.
					// must not send in that case.
					l.taskTimer.Push(w, false)
				} else {
					workCh <- w.Do
				}
			}
		}
	}
	return nil
}
