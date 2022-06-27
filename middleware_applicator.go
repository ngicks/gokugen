package gokugen

//go:generate mockgen -source scheduler.go -destination __mock/scheduler.go

import (
	"sync"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

type Scheduler interface {
	Schedule(task *scheduler.Task) (*scheduler.TaskController, error)
}

var _ Task = &scheduler.TaskController{}

type Task interface {
	Cancel() (cancelled bool)
	CancelWithReason(err error) (cancelled bool)
	GetScheduledTime() time.Time
	IsCancelled() bool
	IsDone() bool
}

type ScheduleHandlerFn = func(ctx SchedulerContext) (Task, error)

type MiddlewareFunc = func(handler ScheduleHandlerFn) ScheduleHandlerFn

type MiddlewareApplicator[T Scheduler] struct {
	scheduler T
	mwMu      sync.Mutex
	mw        []MiddlewareFunc
}

func NewMiddlewareApplicator[T Scheduler](scheduler T) *MiddlewareApplicator[T] {
	return &MiddlewareApplicator[T]{
		scheduler: scheduler,
		mw:        make([]MiddlewareFunc, 0),
	}
}

func (s *MiddlewareApplicator[T]) Schedule(ctx SchedulerContext) (Task, error) {
	schedule := s.apply(func(ctx SchedulerContext) (Task, error) {
		return s.scheduler.Schedule(
			scheduler.NewTask(
				ctx.ScheduledTime(),
				func(
					ctxCancelCh, taskCancelCh <-chan struct{},
					scheduled time.Time,
				) {
					ctx.Work()(ctxCancelCh, taskCancelCh, scheduled)
				},
			),
		)
	})
	return schedule(ctx)
}

func (ma *MiddlewareApplicator[T]) Scheduler() T {
	return ma.scheduler
}

func (s *MiddlewareApplicator[T]) Use(mw ...MiddlewareFunc) {
	s.mwMu.Lock()
	defer s.mwMu.Unlock()

	s.mw = append(s.mw, mw...)
}

func (s *MiddlewareApplicator[T]) apply(schedule ScheduleHandlerFn) ScheduleHandlerFn {
	s.mwMu.Lock()
	defer s.mwMu.Unlock()

	var wrapped ScheduleHandlerFn
	wrapped = schedule
	for i := len(s.mw) - 1; i >= 0; i-- {
		handlerFunc := s.mw[i]
		if handlerFunc != nil {
			wrapped = handlerFunc(wrapped)
		}
	}
	return wrapped
}
