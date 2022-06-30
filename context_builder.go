package gokugen

import "time"

type Option func(ctx SchedulerContext) SchedulerContext

// BuildContext builds a new SchedulerContext.
// workFn, valMap can be nil.
// Add options to set additional values to the ctx.
func BuildContext(scheduledTime time.Time, workFn WorkFn, valMap map[any]any, options ...Option) SchedulerContext {
	var ctx SchedulerContext = NewPlainContext(scheduledTime, workFn, valMap)
	return WrapContext(ctx, options...)
}

// WrapContext wrapps parent with options.
func WrapContext(parent SchedulerContext, options ...Option) (ctx SchedulerContext) {
	ctx = parent
	for _, opt := range options {
		ctx = opt(ctx)
	}
	return ctx
}

func WithParam(param any) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WrapWithParam(ctx, param)
	}
}
func WithParamLoader(loader func() (any, error)) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WrapWithParamLoader(ctx, loader)
	}
}
func WithWorkFn(workFn WorkFn) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WrapWithWorkFn(ctx, workFn)
	}
}
func WithWorkFnWrapper(wrapper WorkFnWrapper) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WrapWithWorkFnWrapper(ctx, wrapper)
	}
}
func WithTaskId(taskId string) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WrapWithTaskId(ctx, taskId)
	}
}
func WithWorkId(workId string) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WrapWithWorkId(ctx, workId)
	}
}
