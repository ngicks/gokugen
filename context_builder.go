package gokugen

import "time"

type Option func(ctx SchedulerContext) SchedulerContext

func BuildContext(scheduledTime time.Time, workFn WorkFn, valMap map[any]any, options ...Option) SchedulerContext {
	var ctx SchedulerContext = NewPlainContext(scheduledTime, workFn, valMap)
	return WrapContext(ctx, options...)
}

func WrapContext(parent SchedulerContext, options ...Option) (ctx SchedulerContext) {
	ctx = parent
	for _, opt := range options {
		ctx = opt(ctx)
	}
	return ctx
}

func WithParamOption(param any) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WithParam(ctx, param)
	}
}
func WithParamLoaderOption(loader func() (any, error)) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WithParamLoader(ctx, loader)
	}
}
func WithWorkFnOption(workFn WorkFn) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WithWorkFn(ctx, workFn)
	}
}
func WithWorkFnWrapperOption(wrapper WorkFnWrapper) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WithWorkFnWrapper(ctx, wrapper)
	}
}
func WithTaskIdOption(taskId string) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WithTaskId(ctx, taskId)
	}
}
func WithWorkIdOption(workId string) Option {
	return func(ctx SchedulerContext) SchedulerContext {
		return WithWorkId(ctx, workId)
	}
}
