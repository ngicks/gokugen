package cron

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ngicks/gokugen"
)

var (
	ErrOnceTask          = errors.New("task returned same schedule time")
	ErrNonexistentWorkId = errors.New("nonexistent work id")
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
	NextSchedule(now time.Time) (time.Time, error)
	GetCommand() []string
}

type CronLikeRescheduler struct {
	mu               sync.Mutex
	err              error
	scheduler        Scheduler
	row              RowLike
	state            *ScheduleState
	scheduled        bool
	ongoingTask      gokugen.Task
	shouldReschedule func(workErr error, callCount int) bool
	workRegistry     WorkRegistry
}

func NewCronLikeRescheduler(
	taskAndSchedule RowLike,
	whence time.Time,
	shouldReschedule func(workErr error, callCount int) bool,
	scheduler Scheduler,
	workRegistry WorkRegistry,
) *CronLikeRescheduler {
	return &CronLikeRescheduler{
		row:              taskAndSchedule,
		state:            NewScheduleState(taskAndSchedule, whence),
		scheduler:        scheduler,
		shouldReschedule: shouldReschedule,
		workRegistry:     workRegistry,
	}
}

func (c *CronLikeRescheduler) Schedule() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
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

	paramSet := gokugen.WithParam(
		gokugen.WithWorkId(gokugen.NewPlainContext(next, nil, nil), command[0]),
		command[1:],
	)
	task, err := c.scheduler.Schedule(
		gokugen.WithWorkFn(
			paramSet,
			func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
				err := workRaw(ctxCancelCh, taskCancelCh, scheduled, command[1:])
				if c.shouldReschedule(err, callCount) {
					c.schedule()
				}
				return err
			},
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

func (c *CronLikeRescheduler) Cancel() (alreadyCancelled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.scheduled = false
	if c.ongoingTask != nil {
		alreadyCancelled := c.ongoingTask.Cancel()
		c.ongoingTask = nil
		return alreadyCancelled
	}
	return true
}
