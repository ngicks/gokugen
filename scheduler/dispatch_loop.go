package scheduler

import (
	"context"
	"fmt"

	"github.com/ngicks/gokugen/common"
)

// DispatchLoop waits for a Feeder to emit the timer signal,
// and then sends received tasks to worker channel.
//
// Multiple calls of Start is ok. But performance benefits are questionable.
type DispatchLoop struct {
	feeder *TaskFeeder
	getNow common.GetNow
}

// NewDispatchLoop creates DispatchLoop.
//
// panic: when one or more of arguments is nil.
func NewDispatchLoop(feeder *TaskFeeder, getNow common.GetNow) *DispatchLoop {
	if feeder == nil || getNow == nil {
		panic(fmt.Errorf(
			"%w: one or more of aruguments is nil. feeder is nil=[%t], getNow is nil=[%t]",
			ErrInvalidArg,
			feeder == nil,
			getNow == nil,
		))
	}
	return &DispatchLoop{
		feeder: feeder,
		getNow: getNow,
	}
}

func (l *DispatchLoop) PushTask(task *Task) error {
	return l.feeder.Push(task)
}

func (l *DispatchLoop) TaskLen() int {
	return l.feeder.Len()
}

// Start starts a dispatch loop.
// Start does not have reponsibility of starting the Feeder.
// A caller must ensure that the feeder is started.
// Calling multiple Start in different goroutines is allowed, but performance benefits are questionable.
// Cancelling ctx will end this Start loop, with returning nil.
//
// If one or more of arguments are nil, Start immediately returns ErrInvalidArg.
//
// panic: Closing taskCh *before* cancelling ctx may cause panic.
func (l *DispatchLoop) Start(ctx context.Context, taskCh chan<- *Task) error {
	if taskCh == nil || ctx == nil {
		return ErrInvalidArg
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-l.feeder.GetTimer():
			next := l.feeder.GetScheduledTask(l.getNow.GetNow())
			if next == nil {
				continue
			}
			for _, w := range next {
				if ctx.Err() != nil {
					// race condition causes deadlock here.
					// must not send in that case.
					l.feeder.Push(w, false)
				} else {
					taskCh <- w
				}
			}
		}
	}
	return nil
}
