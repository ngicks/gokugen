package cron

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ngicks/gokugen"
)

var (
	ErrOnceTask          = errors.New("task returned same schedule time")
	ErrNonexistentWorkId = errors.New("nonexistent work id")
	ErrStillWorking      = errors.New("task is still working")
	ErrAlreadyScheduled  = errors.New("task is already scheduled")
)

type WorkFnWParam = gokugen.WorkFnWParam
type WorkRegistry interface {
	Load(key string) (value WorkFnWParam, ok bool)
}

type Scheduler interface {
	Schedule(ctx gokugen.SchedulerContext) (gokugen.Task, error)
}

type RowLike interface {
	NextScheduler
	GetCommand() []string
}

// CronLikeRescheduler schedules command of RowLike according to its configuration.
type CronLikeRescheduler struct {
	mu               sync.Mutex
	err              error
	scheduler        Scheduler
	row              RowLike
	state            *ScheduleState
	scheduled        bool
	isWorking        int64
	ongoingTask      gokugen.Task
	shouldReschedule func(workErr error, callCount int) bool
	workRegistry     WorkRegistry
}

func NewCronLikeRescheduler(
	rowLike RowLike,
	whence time.Time,
	shouldReschedule func(workErr error, callCount int) bool,
	scheduler Scheduler,
	workRegistry WorkRegistry,
) *CronLikeRescheduler {
	return &CronLikeRescheduler{
		row:              rowLike,
		state:            NewScheduleState(rowLike, whence),
		scheduler:        scheduler,
		shouldReschedule: shouldReschedule,
		workRegistry:     workRegistry,
	}
}

// Schedule starts scheduling.
// If shouldReschedule is non nil and if it returns true,
// Rowlike would be rescheduled to its next time.
//
// - ErrStillWorking is returned if task c created is still being worked on.
//   - Schedule right after Cancel may cause this state. No overlapping schedule is not allowed.
// - ErrAlreadyScheduled is returned if second or more call is without preceding Cancel.
// - ErrOnceTask is returned if RowLike is once task.
//   - c is returned if command of RowLike is invalid.
// - ErrNonexistentWorkId is returned when command does not exist in WorkRegistry.
// - Scheduler's error is returned when Schedule returns error.
//
// ErrOnceTask, ErrOnceTask and Scheduler's error are sticky.
// Once Schedule returned it, Schedule always return that error.
//
// All error may or may not be wrapped. User should use errors.Is().
func (c *CronLikeRescheduler) Schedule() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}

	if atomic.LoadInt64(&c.isWorking) == 1 {
		// Schedule right after Cancel may lead to this state.
		// Overlapping scheduling is not allowed.
		return ErrStillWorking
	}

	if c.ongoingTask != nil || c.scheduled {
		return ErrAlreadyScheduled
	}
	c.scheduled = true

	c.mu.Unlock()
	// defer-ing in case of runtime panic.
	defer c.mu.Lock()
	// schedule takes lock by itself.
	err := c.schedule()
	return err
}

func (c *CronLikeRescheduler) schedule() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if errors.Is(c.err, ErrOnceTask) {
		return nil
	} else if c.err != nil {
		return c.err
	}

	same, callCount, next := c.state.Next()
	if same {
		c.err = ErrOnceTask
	}

	command := c.row.GetCommand()
	if command == nil || len(command) == 0 {
		c.err = ErrMalformed
		return c.err
	}

	workRaw, ok := c.workRegistry.Load(command[0])
	if !ok {
		c.err = fmt.Errorf("%w: workId = %s", ErrNonexistentWorkId, command[0])
		return c.err
	}

	task, err := c.scheduler.Schedule(
		gokugen.BuildContext(
			next,
			nil,
			nil,
			gokugen.WithWorkId(command[0]),
			gokugen.WithParam(command[1:]),
			gokugen.WithWorkFn(
				func(taskCtx context.Context, scheduled time.Time) (any, error) {
					atomic.StoreInt64(&c.isWorking, 1)
					defer atomic.StoreInt64(&c.isWorking, 0)

					ret, err := workRaw(taskCtx, scheduled, command[1:])
					if c.shouldReschedule != nil && c.shouldReschedule(err, callCount) {
						c.schedule()
					}
					return ret, err
				},
			),
		),
	)
	if err != nil {
		c.err = err
		return c.err
	}
	if task != nil {
		c.ongoingTask = task
	}
	return nil
}

// Cancel stops scheduling of c.
// If a task is being worked on at the time Cancel is called,
// this also cancels task itself.
func (c *CronLikeRescheduler) Cancel() (cancelled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.scheduled = false
	if c.ongoingTask != nil {
		cancelled := c.ongoingTask.Cancel()
		c.ongoingTask = nil
		return cancelled
	}
	return
}
