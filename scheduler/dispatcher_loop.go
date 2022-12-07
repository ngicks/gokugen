package scheduler

import (
	"context"
	"fmt"

	"github.com/ngicks/gommon"
)

// DispatcherLoop waits for a TaskTimer to emit the timer signal,
// and then sends scheduled tasks to worker channel.
//
// Multiple calls of Start is allowed. But performance benefits are questionable.
type DispatcherLoop struct {
	taskDispenser TaskDispenser
	dispatcher    Dispatcher
	getNow        gommon.GetNower
}

// NewDispatchLoop creates DispatchLoop.
//
// panic: when one or more of arguments is nil.
func NewDispatchLoop(taskDispenser TaskDispenser, getNow gommon.GetNower) *DispatcherLoop {
	if taskDispenser == nil || getNow == nil {
		panic(fmt.Errorf(
			"err invalid arg: one or more of arguments is nil. taskTimer is nil=[%t], getNow is nil=[%t]",
			taskDispenser == nil,
			getNow == nil,
		))
	}
	return &DispatcherLoop{
		taskDispenser: taskDispenser,
		getNow:        getNow,
	}
}

// Start starts a dispatcher loop.
// Start does not have responsibility of starting the TaskTimer.
// A caller must ensure that the taskTimer is started.
// Calling multiple Start in different goroutines is allowed, but performance benefits are questionable.
// Cancelling ctx will end this Start loop, with returning nil.
//
// If one or more of arguments are nil, Start immediately returns ErrInvalidArg.
//
// panic:
//   - if any of args is nil.
func (l *DispatcherLoop) Start(ctx context.Context) error {
	if ctx == nil {
		panic(fmt.Errorf(
			"err invalid arg: one or more of arguments is nil. ctx is nil=[%t]",
			ctx == nil,
		))
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-l.taskDispenser.Timer():
			var err error
			next, err := l.taskDispenser.TaskBefore(l.getNow.GetNow())
			if err != nil {
				continue
			}
			if next == nil {
				continue
			}
			err = l.dispatcher.Dispatch(ctx, next)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
