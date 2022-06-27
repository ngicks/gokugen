package gokugen

//go:generate mockgen -source scheduler.go -destination __mock/scheduler.go

import (
	"errors"
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

type WorkFn = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error)
type WorkFnWParam = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) (any, error)

var (
	ErrValueNotFound = errors.New("value not found")
)

type PlainContext struct {
	scheduledTime time.Time
	workFn        WorkFn
	values        map[any]any
}

func NewPlainContext(scheduledTime time.Time, workFn WorkFn, values map[any]any) SchedulerContext {
	return &PlainContext{
		scheduledTime: scheduledTime,
		workFn:        workFn,
		values:        values,
	}
}

func (ctx *PlainContext) ScheduledTime() time.Time {
	return ctx.scheduledTime
}
func (ctx *PlainContext) Work() WorkFn {
	return ctx.workFn
}
func (ctx *PlainContext) Value(key any) (any, error) {
	if ctx.values == nil {
		return nil, nil
	}
	return ctx.values[key], nil
}

type SchedulerContext interface {
	ScheduledTime() time.Time
	Work() WorkFn
	Value(key any) (any, error)
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
