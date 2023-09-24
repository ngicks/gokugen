package scheduler

import (
	"context"
	"errors"
	"sync"

	"github.com/ngicks/eventqueue"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/mockable"
)

type kindTask struct {
	task   def.Task
	isRepo bool
}

type taskResult struct {
	beforeDispatch kindTask
	err            error
}

type Scheduler struct {
	repo       Repository
	dispatcher def.Dispatcher

	stepMu     sync.Mutex
	lastTask   *kindTask
	getNextErr error

	taskResultCh <-chan taskResult
	eventQueue   *eventqueue.EventQueue[taskResult]

	clock mockable.Clock
}

func NewScheduler(
	repo Repository,
	dispatcher def.Dispatcher,
) *Scheduler {
	return newScheduler(repo, dispatcher)
}

func NewVolatileTask(
	volatileTask VolatileTask,
	dispatcher def.Dispatcher,
) *Scheduler {
	return newScheduler(newVolatileTaskRepo(volatileTask), dispatcher)
}

func newScheduler(repo Repository, dispatcher def.Dispatcher) *Scheduler {
	sink := eventqueue.NewChannelSink[taskResult](0)

	return &Scheduler{
		repo:       repo,
		dispatcher: dispatcher,

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

	s.getNextErr = nil

	if s.lastTask != nil {
		next := *s.lastTask
		s.lastTask = nil
		return s.dispatchTask(ctx, next, false)
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
	}
}

func (s *Scheduler) Retry(ctx context.Context, prev StepState) (state StepState, retryErr bool) {
	s.stepMu.Lock()
	defer s.stepMu.Unlock()

	err := prev.Match(StepResultHandler{
		TimerUpdateError: func(err error) error {
			s.repo.StopTimer()
			s.repo.StartTimer(ctx)
			if err := s.repo.LastTimerUpdateError(); err != nil {
				state = StateTimerUpdateError(err)
				return err
			}
			return nil
		},
		AwaitingNext: func(err error) error {
			return nil
		},
		NextTask: func(task def.Task, err error) error {
			return nil
		},
		DispatchErr: func(task def.Task, isRepo bool, _ error) error {
			fetched, err := s.repo.GetById(ctx, task.Id)
			if err != nil && !def.IsDefError(err) {
				state = StateDispatchErr(task, isRepo, err)
				return err
			}
			if err != nil {
				task = fetched
			}
			state = s.dispatchTask(ctx, kindTask{task, isRepo}, true)
			return state.Err()
		},
		Dispatched: func(id string) error {
			return nil
		},
		TaskDone: func(id string, taskErr error, updateErr error) error {
			err := s.repo.MarkAsDone(ctx, id, taskErr)
			if err != nil && !def.IsAlreadyDone(err) {
				state = StateTaskDone(id, taskErr, err)
				return err
			}
			return nil
		},
	})

	if err != nil {
		retryErr = true
	}

	return
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

func (s *Scheduler) dispatchTask(ctx context.Context, next kindTask, isRetry bool) StepState {
	errCh, dispatchErr := s.dispatcher.Dispatch(
		ctx,
		func(ctx context.Context) (def.Task, error) {
			var err error
			if !isRetry {
				err = s.repo.MarkAsDispatched(ctx, next.task.Id)
			}
			if err != nil {
				return def.Task{}, err
			}
			task, err := s.repo.GetById(ctx, next.task.Id)
			if err != nil {
				return def.Task{}, err
			}
			return task, nil
		},
	)
	if dispatchErr != nil {
		return StateDispatchErr(next.task, next.isRepo, dispatchErr)
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
