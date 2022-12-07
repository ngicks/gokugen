package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/gommon"
	"github.com/ngicks/gommon/pkg/atomicstate"
)

// CancellerLoop requests TaskDispenser to remove cancelled element at a given interval.
// This is intended to be only one running instance per a TaskDispenser.
type CancellerLoop struct {
	*atomicstate.WorkingStateChecker
	workingState *atomicstate.WorkingStateSetter
	taskTimer    TaskDispenser
	getNow       gommon.GetNower
	interval     time.Duration
}

// NewCancellerLoop creates a CancellerLoop.
//
// panic: when one or more of arguments is nil or zero-value.
func NewCancellerLoop(taskTimer TaskDispenser, getNow gommon.GetNower, interval time.Duration) *CancellerLoop {
	if taskTimer == nil || getNow == nil || interval <= 0 {
		panic(fmt.Errorf(
			"err arg invalid: one or more of arguments is nil or zero-value. "+
				"taskTimer is nil=[%t], getNow is nil=[%t], interval is zero=[%t]",
			taskTimer == nil,
			getNow == nil,
			interval == 0,
		))
	}
	return &CancellerLoop{
		taskTimer: taskTimer,
		getNow:    getNow,
		interval:  interval,
	}
}

// Start starts a loop that requests TaskTimer to remove cancelled tasks at at given interval.
// Cancelling of Start is controlled by ctx.
//
// If ctx is nil, Start immediately returns ErrInvalidArg.
// If loop is already running in some goroutine, Start immediately returns ErrAlreadyStarted.
func (l *CancellerLoop) Start(ctx context.Context) error {
	if ctx == nil {
		panic(fmt.Errorf(
			"err invalid arg: one or more of arguments is nil. ctx is nil=[%t]",
			ctx == nil,
		))
	}

	if !l.workingState.SetWorking() {
		return ErrAlreadyStarted
	}
	defer l.workingState.SetWorking(false)

	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			removeCancelled(l.taskTimer, l.getNow)

		}
	}
	return nil
}

func removeCancelled(taskDispenser TaskDispenser, getNow gommon.GetNower) (removed bool) {
	p, err := taskDispenser.Peek()
	if err != nil {
		return false
	}

	if p != nil && p.ScheduledAt().Sub(getNow.GetNow()) > time.Second {
		// Racy Push may add min element in between previous Peek and this RemoveCancelled.
		// But it is ok because each taskTimer method is thread safe.
		// Benchmark shows RemoveCancelled takes only a few micro secs.
		taskDispenser.RemoveCancelled()
		return true
	}
	return false
}
