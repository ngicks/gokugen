package scheduler

import (
	"context"

	"github.com/ngicks/type-param-common/set"
)

type Scheduler struct {
	workRegistry WorkRegistry
	repo         TaskRepository
	hooks        *hookWrapper

	meta *set.Set[string]
	loop loop
}

func New(
	workRegistry WorkRegistry,
	dispatcher Dispatcher,
	repo TaskRepository,
	hooks LoopHooks,
) *Scheduler {

	wrappedHook := newHookWrapper(hooks)
	s := &Scheduler{
		workRegistry: workRegistry,
		repo:         repo,
		hooks:        wrappedHook,

		meta: set.New[string](),
		loop: newLoop(dispatcher, repo, wrappedHook),
	}

	return s
}

func (s *Scheduler) Run(ctx context.Context, startTimer, stopTimerOnClose bool) error {
	return s.loop.Run(ctx, startTimer, stopTimerOnClose)
}

func (s *Scheduler) AddTask(param TaskParam) (Task, error) {
	if _, ok := s.workRegistry.Load(param.WorkId); !ok {
		return Task{}, &ErrWorkIdNotFound{param}
	}
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

func (s *Scheduler) RegisterMetaKey(key string) (registered bool) {
	if s.meta.Has(key) {
		return false
	}
	s.meta.Add(key)
	return true
}
