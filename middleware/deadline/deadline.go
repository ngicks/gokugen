package deadline

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/common"
)

type DeadlineExeededErr struct {
	scheduled time.Time
	executed  time.Time
}

func (e DeadlineExeededErr) Error() string {
	return fmt.Sprintf(
		"deadline exeeded: scheduled at=%s, but executed at=%s",
		e.scheduled.Format(time.RFC3339Nano),
		e.executed.Format(time.RFC3339Nano),
	)
}
func (e DeadlineExeededErr) ScheduledTime() time.Time {
	return e.scheduled
}

func (e DeadlineExeededErr) ExecutedTime() time.Time {
	return e.executed
}

type DeadlineMiddleware struct {
	deadline   time.Duration
	getNow     common.GetNower
	shouldSkip func(ctx gokugen.SchedulerContext) bool
}

func New(deadline time.Duration, shouldSkip func(ctx gokugen.SchedulerContext) bool) *DeadlineMiddleware {
	return &DeadlineMiddleware{
		deadline:   deadline,
		shouldSkip: shouldSkip,
	}
}

func (mw *DeadlineMiddleware) Middleware(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		if mw.shouldSkip(ctx) {
			return handler(ctx)
		}
		return handler(
			gokugen.WrapContext(
				ctx,
				gokugen.WithWorkFnWrapper(wrapper(mw.deadline, mw.getNow)),
			),
		)
	}
}

func wrapper(
	deadline time.Duration,
	getNow common.GetNower,
) func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
	return func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
		return func(taskCtx context.Context, scheduled time.Time) (any, error) {
			now := getNow.GetNow()
			if now.Sub(scheduled) > deadline {
				return nil, DeadlineExeededErr{scheduled: scheduled, executed: now}
			}
			return workFn(taskCtx, scheduled)
		}
	}
}
