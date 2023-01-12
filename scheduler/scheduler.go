package scheduler

import (
	"context"

	"github.com/ngicks/gommon/pkg/atomicstate"
)

type Scheduler struct {
	*atomicstate.RunningStateChecker
	runningState *atomicstate.RunningStateSetter

	dispatcher Dispatcher
	repo       TaskRepository
	hooks      *hookWrapper

	loop loop
}

func New(
	dispatcher Dispatcher,
	repo TaskRepository,
	hooks LoopHooks,
) *Scheduler {
	checker, setter := atomicstate.NewRunningState()

	wrappedHook := newHookWrapper(hooks)
	s := &Scheduler{
		RunningStateChecker: checker,
		runningState:        setter,

		dispatcher: dispatcher,
		repo:       repo,
		hooks:      wrappedHook,

		loop: newLoop(dispatcher, repo, wrappedHook),
	}

	return s
}

func (s *Scheduler) Run(ctx context.Context, startTimer, stopTimerOnClose bool) error {
	if s.RunningStateChecker.IsRunning() {
		return ErrAlreadyRunning
	}
	s.runningState.SetRunning()
	defer s.runningState.SetRunning(false)

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

func (s *Scheduler) AddOnTaskDone(fn *OnTaskDone) {
	s.hooks.addOnTaskDone(fn)
}

func (s *Scheduler) RemoveOnTaskDone(fn *OnTaskDone) {
	s.hooks.removeOnTaskDone(fn)
}
