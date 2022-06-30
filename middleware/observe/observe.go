package observe

import (
	"context"
	"time"

	"github.com/ngicks/gokugen"
)

type ObserveMiddleware struct {
	ctxObserver    func(ctx gokugen.SchedulerContext)
	workFnObserver func(ret any, err error)
}

func New(ctxObserver func(ctx gokugen.SchedulerContext), workFnObserver func(ret any, err error)) *ObserveMiddleware {
	if ctxObserver == nil {
		ctxObserver = func(ctx gokugen.SchedulerContext) {}
	}
	if workFnObserver == nil {
		workFnObserver = func(ret any, err error) {}
	}

	return &ObserveMiddleware{
		ctxObserver:    ctxObserver,
		workFnObserver: workFnObserver,
	}
}

func (mw *ObserveMiddleware) Middleware(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		mw.ctxObserver(ctx)
		return handler(
			gokugen.WrapContext(
				ctx,
				gokugen.WithWorkFnWrapper(
					func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
						return func(taskCtx context.Context, scheduled time.Time) (ret any, err error) {
							ret, err = workFn(taskCtx, scheduled)
							mw.workFnObserver(ret, err)
							return
						}
					},
				),
			),
		)
	}
}
