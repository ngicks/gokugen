package observe_test

import (
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	mock_gokugen "github.com/ngicks/gokugen/__mock"
	"github.com/ngicks/gokugen/middleware/observe"
	"github.com/stretchr/testify/assert"
)

func sapmleMw(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		return handler(
			gokugen.WrapContext(
				ctx,
				gokugen.WithWorkFnWrapper(func(_ gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
					return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) {
						ret, _ := workFn(ctxCancelCh, taskCancelCh, scheduled)
						return ret.(string) + "bar", nil
					}
				}),
				gokugen.WithWorkId(":"),
			),
		)
	}
}

func TestObserve(t *testing.T) {
	_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

	ma := gokugen.NewMiddlewareApplicator(mockSched)

	inputCtx := gokugen.NewPlainContext(
		time.Now(),
		func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) {
			return "foo", nil
		},
		nil,
	)

	ma.Use(
		observe.New(
			func(ctx gokugen.SchedulerContext) {
				assert.Equal(t, ctx.(*gokugen.PlainContext), inputCtx)
			},
			func(ret any, err error) {
				assert.Equal(t, ret.(string), "foo")
			},
		).Middleware,
		sapmleMw,
		observe.New(
			func(ctx gokugen.SchedulerContext) {
				workId, _ := gokugen.GetWorkId(ctx)
				assert.Equal(t, workId, ":")
			},
			func(ret any, err error) {
				assert.Equal(t, ret.(string), "foobar")
			},
		).Middleware,
	)

	ma.Schedule(inputCtx)
	getTrappedTask().Do(make(<-chan struct{}))
}
