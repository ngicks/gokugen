package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/gokugen/common"
)

// CancellerLoop requests TaskTimer to remove cancelled element at a given interval.
// This is intended to be only one running instance per a TaskTimer.
type CancellerLoop struct {
	workingState
	taskTimer *TaskTimer
	getNow    common.GetNower
	interval  time.Duration
}

// NewCancellerLoop creates a CancellerLoop.
//
// panic: when one or more of arguments is nil or zero-vakue.
func NewCancellerLoop(taskTimer *TaskTimer, getNow common.GetNower, interval time.Duration) *CancellerLoop {
	if taskTimer == nil || getNow == nil || interval <= 0 {
		panic(fmt.Errorf(
			"%w: one or more of aruguments is nil or zero-value. taskTimer is nil=[%t], getNow is nil=[%t], interval is zero=[%t]",
			ErrInvalidArg,
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
		return ErrInvalidArg
	}

	if !l.setWorking() {
		return ErrAlreadyStarted
	}
	defer l.setWorking(false)

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

func removeCancelled(taskTimer *TaskTimer, getNow common.GetNower) (removed bool) {
	p := taskTimer.Peek()
	if p != nil && p.scheduledTime.Sub(getNow.GetNow()) > time.Second {
		// Racy Push may add min element in between previous Peek and this RemoveCancelled.
		// But it is ok because each taskTimer method is thread safe.
		// Benchmark shows RemoveCancelled takes only a few micro secs.
		taskTimer.RemoveCancelled(0, 10_000)
		return true
	}
	return false
}
