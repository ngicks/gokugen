package log

import (
	"time"

	"github.com/ngicks/gokugen"
)

//go:generate mockgen -source log.go -destination __mock/log.go

type Logger interface {
	Info(taskId, workId string, v any)
	Error(taskId, workId string, e error)
}

type LogMiddleware struct {
	logger Logger
}

func New(logger Logger) *LogMiddleware {
	return &LogMiddleware{
		logger: logger,
	}
}

func (mw *LogMiddleware) Middleware(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		return handler(gokugen.WithWorkFnWrapper(
			ctx,
			func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
				return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) {
					taskId, _ := gokugen.GetTaskId(self)
					workId, _ := gokugen.GetWorkId(self)
					ret, err := workFn(ctxCancelCh, taskCancelCh, scheduled)
					if err != nil {
						mw.logger.Error(taskId, workId, err)
					} else {
						mw.logger.Info(taskId, workId, ret)
					}
					return ret, err
				}
			},
		))
	}
}
