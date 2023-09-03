package scheduler

import (
	"context"
	"errors"
	"sync"

	"github.com/ngicks/eventqueue"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/mockable"
)

type VolatileTask interface {
	def.Observer
	GetNext(ctx context.Context) (def.Task, error)
	Pop(ctx context.Context) (def.Task, error)
}

type kindTask struct {
	task   def.Task
	isRepo bool
}

type taskResult struct {
	beforeDispatch kindTask
	err            error
}

type Scheduler struct {
	volatileTask VolatileTask
	repo         def.ObservableRepository
	dispatcher   def.Dispatcher

	stepMu     sync.Mutex
	lastTask   *kindTask
	getNextErr error

	taskResultCh <-chan taskResult
	eventQueue   *eventqueue.EventQueue[taskResult]

	clock mockable.Clock
}

func NewScheduler(
	repo def.ObservableRepository,
	volatileTask VolatileTask,
	dispatcher def.Dispatcher,
) *Scheduler {
	if repo == nil && volatileTask == nil {
		panic("Either or both of repo and volatileTask must be non nil")
	}
	sink := eventqueue.NewChannelSink[taskResult](0)

	if repo == nil {
		repo = new(nopRepository)
	}
	if volatileTask == nil {
		volatileTask = new(nopVolatileTask)
	}

	return &Scheduler{
		volatileTask: volatileTask,
		repo:         repo,
		dispatcher:   dispatcher,

		taskResultCh: sink.Outlet(),
		eventQueue:   eventqueue.New[taskResult](sink),
		clock:        mockable.NewClockReal(),
	}
}

func (s *Scheduler) RunQueue(ctx context.Context) (remaining int, err error) {
	return s.eventQueue.Run(ctx)
}

// Step lets s proceed to the next step. It blocks until s reaches a state or an error occurs.
func (s *Scheduler) Step(ctx context.Context) StepState {
	s.stepMu.Lock()
	defer s.stepMu.Unlock()

	if s.getNextErr != nil || s.repo.LastTimerUpdateError() != nil {
		s.repo.StopTimer()
		s.repo.StartTimer(ctx)
		if err := s.repo.LastTimerUpdateError(); err != nil {
			return StateTimerUpdateError(err)
		}
	}
	if s.getNextErr != nil || s.volatileTask.LastTimerUpdateError() != nil {
		s.volatileTask.StopTimer()
		s.volatileTask.StartTimer(ctx)
		if err := s.volatileTask.LastTimerUpdateError(); err != nil {
			return StateTimerUpdateError(err)
		}
	}

	s.getNextErr = nil

	if s.lastTask != nil {
		next := *s.lastTask
		s.lastTask = nil
		return s.dispatchTask(ctx, next)
	}

	select {
	case <-ctx.Done():
		return StateAwaitingNext(ctx.Err())
	case res := <-s.taskResultCh:
		var err error
		if res.beforeDispatch.isRepo && !errors.Is(res.err, context.Canceled) {
			// In case dispatcher is cancelled.
			err = s.repo.MarkAsDone(ctx, res.beforeDispatch.task.Id, res.err)
		}
		return StateTaskDone(res.beforeDispatch.task.Id, res.err, err)
	case <-s.repo.TimerChannel():
		next, err := s.repo.GetNext(ctx)
		s.setGetNextResult(next, err, true)
		return StateNextTask(next, err)
	case <-s.volatileTask.TimerChannel():
		next, err := s.volatileTask.GetNext(ctx)
		s.setGetNextResult(next, err, false)
		return StateNextTask(next, err)
	}
}

func (s *Scheduler) setGetNextResult(task def.Task, err error, isRepo bool) {
	if err == nil {
		s.lastTask = &kindTask{
			task:   task,
			isRepo: isRepo,
		}
		s.getNextErr = nil
	} else {
		s.lastTask = nil
		s.getNextErr = err
	}
}

func (s *Scheduler) dispatchTask(ctx context.Context, next kindTask) StepState {
	errCh, dispatchErr := s.dispatcher.Dispatch(
		ctx,
		func() func(context.Context) (def.Task, error) {
			if next.isRepo {
				return s.repoFetcher(next.task.Id)
			} else {
				return s.volatileFetcher(next.task.Id)
			}
		}(),
	)
	if dispatchErr != nil {
		return StateDispatchErr(next.task.Id, dispatchErr)
	}

	s.eventQueue.Reserve(func() taskResult {
		err := <-errCh
		return taskResult{
			beforeDispatch: next,
			err:            err,
		}
	})

	return StateDispatched(next.task.Id)
}

func (s *Scheduler) repoFetcher(id string) func(ctx context.Context) (def.Task, error) {
	return func(ctx context.Context) (def.Task, error) {
		err := s.repo.MarkAsDispatched(ctx, id)
		if err != nil {
			return def.Task{}, err
		}
		task, err := s.repo.GetById(ctx, id)
		if err != nil {
			return def.Task{}, err
		}
		return task, nil
	}
}

func (s *Scheduler) volatileFetcher(id string) func(ctx context.Context) (def.Task, error) {
	return func(ctx context.Context) (def.Task, error) {
		return s.volatileTask.Pop(ctx)
	}
}
