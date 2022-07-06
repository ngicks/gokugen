package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ngicks/gommon"
	"github.com/ngicks/gommon/state"
	gopherpool "github.com/ngicks/gopher-pool"
)

// Scheduler is an in-memory scheduler backed by min heap.
type Scheduler struct {
	*state.WorkingStateChecker
	workingInner *state.WorkingStateSetter
	*state.EndedStateChecker
	endedInner *state.EndedStateSetter

	wg sync.WaitGroup

	taskTimer     *TaskTimer
	cancellerLoop *CancellerLoop
	dispatchLoop  *DispatchLoop
	workerPool    *gopherpool.WorkerPool
	workCh        chan gopherpool.WorkFn
}

func NewScheduler(initialWorkerNum, queueMax uint) *Scheduler {
	return newScheduler(initialWorkerNum, queueMax, gommon.GetNowImpl{})
}

func newScheduler(initialWorkerNum, queueMax uint, getNow gommon.GetNower) *Scheduler {

	workingStateChecker, workingStateInner := state.NewWorkingState()
	endedStateChecker, endedStateInner := state.NewEndedState()

	workCh := make(chan gopherpool.WorkFn)
	taskTimer := NewTaskTimer(queueMax, getNow, gommon.NewTimerImpl())

	s := &Scheduler{
		WorkingStateChecker: workingStateChecker,
		workingInner:        workingStateInner,
		EndedStateChecker:   endedStateChecker,
		endedInner:          endedStateInner,

		taskTimer:     taskTimer,
		cancellerLoop: NewCancellerLoop(taskTimer, getNow, time.Minute),
		dispatchLoop:  NewDispatchLoop(taskTimer, getNow),
		workerPool: gopherpool.NewWorkerPool(
			gopherpool.SetDefaultWorkerConstructor(
				workCh,
				nil,
				nil,
			),
		),
		workCh: workCh,
	}
	s.AddWorker(uint32(initialWorkerNum))
	return s
}

func (s *Scheduler) Schedule(task *Task) (*TaskController, error) {
	if s.IsEnded() {
		return nil, ErrAlreadyEnded
	}

	err := s.dispatchLoop.PushTask(task)
	if err != nil {
		return nil, err
	}
	return &TaskController{t: task}, nil
}

type LoopError struct {
	cancellerLoopErr error
	dispatchLoopErr  error
}

func (e LoopError) Error() string {
	return fmt.Sprintf(
		"cancellerLoopErr: %s, dispatchLoopErr: %s",
		e.cancellerLoopErr,
		e.dispatchLoopErr,
	)
}

// Start starts needed loops.
// Start creates one goroutine for periodical-removal-of-cancelled-task.
// Start blocks until ctx is cancelled, and other loops to return.
func (s *Scheduler) Start(ctx context.Context) error {
	if s.IsEnded() {
		return ErrAlreadyEnded
	}
	if !s.workingInner.SetWorking() {
		return ErrAlreadyStarted
	}
	defer s.workingInner.SetWorking(false)

	err := new(LoopError)
	s.wg.Add(1)
	s.taskTimer.Start()
	go func() {
		err.cancellerLoopErr = s.cancellerLoop.Start(ctx)
		s.wg.Done()
	}()
	err.dispatchLoopErr = s.dispatchLoop.Start(ctx, s.workCh)
	s.wg.Wait()
	s.taskTimer.Stop()

	if err.cancellerLoopErr == nil && err.dispatchLoopErr == nil {
		return nil
	}
	return err
}

func (s *Scheduler) AddWorker(delta uint32) (workerNum int) {
	return int(s.workerPool.Add(delta))
}

func (s *Scheduler) RemoveWorker(delta uint32) (aliveWorkerNum int, sleepingWorkerNum int) {
	return s.workerPool.Remove(delta)
}

func (s *Scheduler) ActiveWorkerNum() int64 {
	return s.workerPool.ActiveWorkerNum()
}

// End remove all workers and let this scheduler to step into ended-state where no new Start is allowed.
// End also cancel tasks if they are working on in any work.
// Calling this method *before* cancelling of ctx passed to Start will cause blocking forever.
func (s *Scheduler) End() {
	s.endedInner.SetEnded()
	// wait for the Start loop to be done.
	s.wg.Wait()
	s.workerPool.Kill()
	s.workerPool.Wait()
}
