package scheduler

import (
	"context"
	"errors"

	"github.com/ngicks/gommon/pkg/atomicstate"
)

type Scheduler struct {
	*atomicstate.WorkingStateChecker
	workingState *atomicstate.WorkingStateSetter

	dispatcher Dispatcher
	repo       TaskRepository
	hooks      LoopHooks

	loop loop
}

func New(
	dispatcher Dispatcher,
	repo TaskRepository,
	hooks LoopHooks,
) *Scheduler {
	checker, setter := atomicstate.NewWorkingState()

	s := &Scheduler{
		WorkingStateChecker: checker,
		workingState:        setter,

		dispatcher: dispatcher,
		repo:       repo,
		hooks:      hooks,

		loop: newLoop(dispatcher, repo, hooks),
	}

	return s
}

var (
	ErrAlreadyRunning = errors.New("already running")
)

func (s *Scheduler) Run(ctx context.Context, startTimer, stopTimerOnClose bool) error {
	if s.WorkingStateChecker.IsWorking() {
		return ErrAlreadyEnded
	}
	s.workingState.SetWorking()
	defer s.workingState.SetWorking(false)

	return s.loop.Run(ctx, startTimer, stopTimerOnClose)
}

func (s *Scheduler) AddTask(param TaskParam) (Task, error) {
	return s.repo.AddTask(param)
}

func (s *Scheduler) Cancel(id string) error {
	return s.loop.Cancel(id)
}

func (s *Scheduler) Update(id string, param TaskParam) error {
	return s.loop.Update(id, param)
}
