package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ngicks/gommon/pkg/atomicstate"
	"github.com/ngicks/gommon/pkg/common"
)

type Scheduler struct {
	*atomicstate.WorkingStateChecker
	workingState *atomicstate.WorkingStateSetter

	wg            sync.WaitGroup
	taskDispenser TaskDispenser

	dispatcherLoop *DispatcherLoop
	cancellerLoop  *CancellerLoop
}

type Option func(s *Scheduler) *Scheduler

func SetGetNow(getNow common.GetNower) Option {
	return func(s *Scheduler) *Scheduler {
		s.dispatcherLoop.getNow = getNow
		s.cancellerLoop.getNow = getNow
		return s
	}
}

func SetInterval(interval time.Duration) Option {
	return func(s *Scheduler) *Scheduler {
		s.cancellerLoop.interval = interval
		return s
	}
}

func New(taskDispenser TaskDispenser, options ...Option) *Scheduler {
	checker, setter := atomicstate.NewWorkingState()

	s := &Scheduler{
		WorkingStateChecker: checker,
		workingState:        setter,
		taskDispenser:       taskDispenser,
		dispatcherLoop:      NewDispatchLoop(taskDispenser, common.GetNowImpl{}),
		cancellerLoop:       NewCancellerLoop(taskDispenser, common.GetNowImpl{}, time.Hour),
	}

	for _, opt := range options {
		s = opt(s)
	}
	return s
}

func (s *Scheduler) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: one or more args are invalid. ctx == nil:[%t]",
			ErrInvalidArg,
			ctx == nil,
		)
	}

	if !s.workingState.SetWorking() {
		return ErrAlreadyStarted
	}
	defer s.workingState.SetWorking(false)

	err := new(LoopError)

	s.wg.Add(1)
	go func() {
		s.taskDispenser.MarkStart()
		s.wg.Done()
	}()

	s.wg.Add(1)
	go func() {
		err.cancellerLoopErr = s.cancellerLoop.Start(ctx)
		s.wg.Done()
	}()

	err.dispatchLoopErr = s.dispatcherLoop.Start(ctx)

	s.wg.Wait()

	s.taskDispenser.Stop()

	if err.IsEmpty() {
		return nil
	}
	return err
}
