package log

//go:generate mockgen -source log.go -destination __mock/log.go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ngicks/gokugen"
)

type Logger interface {
	Info(v any, logValues ...string)
	Error(e error, logValues ...string)
}

type LogMiddleware struct {
	logger           Logger
	timeFormat       string
	shouldLogParam   bool
	additionalValues []any
}

type Option = func(mw *LogMiddleware) *LogMiddleware

func SetTimeFormat(timeFormat string) (Option, error) {
	_, err := time.Parse(timeFormat, time.Now().Format(timeFormat))
	if err != nil {
		return nil, err
	}

	return func(mw *LogMiddleware) *LogMiddleware {
		mw.timeFormat = timeFormat
		return mw
	}, nil
}

func LogParam() Option {
	return func(mw *LogMiddleware) *LogMiddleware {
		mw.shouldLogParam = true
		return mw
	}
}

func LogAdditionalValues(keys ...any) Option {
	return func(mw *LogMiddleware) *LogMiddleware {
		mw.additionalValues = keys
		return mw
	}
}

func New(logger Logger, options ...Option) *LogMiddleware {
	mw := &LogMiddleware{
		logger:     logger,
		timeFormat: time.RFC3339Nano,
	}
	for _, opt := range options {
		mw = opt(mw)
	}
	if mw.additionalValues == nil {
		mw.additionalValues = make([]any, 0)
	}
	return mw
}

func (mw *LogMiddleware) Middleware(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		return handler(
			gokugen.WrapContext(
				ctx,
				gokugen.WithWorkFnWrapper(wrapper(mw.logger, mw.buildLogValueSet)),
			),
		)
	}
}

type logValueSetBuilder = func(ctx gokugen.SchedulerContext) (logValues []string)

func wrapper(logger Logger, logVal logValueSetBuilder) func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
	return func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
		return func(taskCtx context.Context, scheduled time.Time) (any, error) {
			values := logVal(self)
			logger.Info(nil, append(values, "timing", "before_work")...)
			ret, err := workFn(taskCtx, scheduled)
			values = append(values, "timing", "after_work")
			if err != nil {
				logger.Error(err, values...)
			} else {
				logger.Info(ret, values...)
			}
			return ret, err
		}
	}
}

func (mw *LogMiddleware) buildLogValueSet(ctx gokugen.SchedulerContext) (logValues []string) {
	logValues = append(logValues, "scheduled_at", ctx.ScheduledTime().Format(time.RFC3339Nano))

	taskId, _ := gokugen.GetTaskId(ctx)
	if taskId != "" {
		logValues = append(logValues, "task_id", taskId)
	}
	workId, _ := gokugen.GetWorkId(ctx)
	if workId != "" {
		logValues = append(logValues, "work_id", workId)
	}

	if mw.shouldLogParam {
		param, err := gokugen.GetParam(ctx)
		if err == nil {
			marshaled, err := json.Marshal(param)
			if err == nil {
				logValues = append(logValues, "param", string(marshaled))
			} else {
				logValues = append(logValues, "param", "marshalling error")
			}
		} else if !errors.Is(err, gokugen.ErrValueNotFound) {
			logValues = append(logValues, "param", "loading error")
		}
	}

	for _, key := range mw.additionalValues {
		val, err := ctx.Value(key)
		keyLabel := fmt.Sprintf("%v", key)
		if err != nil && !errors.Is(err, gokugen.ErrValueNotFound) {
			logValues = append(logValues, keyLabel, "loading error")
		} else {
			marshaled, err := json.Marshal(val)
			if err == nil {
				logValues = append(logValues, keyLabel, string(marshaled))
			} else {
				logValues = append(logValues, keyLabel, "marshalling error")
			}
		}
	}

	return
}
