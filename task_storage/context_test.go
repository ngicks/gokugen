package taskstorage

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
)

func TestContext(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		taskId := "t"
		workId := "w"
		param := "foobarbaz"
		now := time.Now()
		work := func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
			return nil
		}

		ctx := newTaskStorageCtx(gokugen.NewPlainContext(now, work, make(map[any]any)), workId, param)
		setTaskId(ctx, taskId)

		if v, err := GetWorkId(ctx); err != nil || v != workId {
			t.Fatalf("param is incorrect: %v", v)
		}
		if v, err := GetTaskId(ctx); err != nil || v != taskId {
			t.Fatalf("param is incorrect: %v", v)
		}
		if v, err := GetParam(ctx); err != nil || v != param {
			t.Fatalf("param is incorrect: %v", v)
		}

		var workCalled, postProcesseCalled int64
		workWithParam := func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
			atomic.AddInt64(&workCalled, 1)
			return nil
		}
		postProcesse := func(err error) {
			atomic.AddInt64(&postProcesseCalled, 1)
		}

		ctxParamLoadable := newtaskStorageParamLoadableCtx(ctx, func() (any, error) {
			return param, nil
		}, workWithParam, postProcesse)

		if v, err := GetWorkId(ctxParamLoadable); err != nil || v != workId {
			t.Fatalf("param is incorrect: %v", v)
		}
		if v, err := GetTaskId(ctxParamLoadable); err != nil || v != taskId {
			t.Fatalf("param is incorrect: %v", v)
		}
		if v, err := GetParam(ctxParamLoadable); err != nil || v != param {
			t.Fatalf("param is incorrect: %v", v)
		}

		{
			var ctx gokugen.SchedulerContext = ctxParamLoadable

			if ctx.ScheduledTime() != now {
				t.Fatalf("ScheduledTime does not match")
			}
			ctx.Work()(make(<-chan struct{}), make(<-chan struct{}), now)

			if atomic.LoadInt64(&workCalled) != 1 || atomic.LoadInt64(&postProcesseCalled) != 1 {
				t.Fatalf(
					"call count is incorrect workCalled=%d, postProcesseCalled=%d",
					atomic.LoadInt64(&workCalled),
					atomic.LoadInt64(&postProcesseCalled),
				)
			}
		}
	})

	t.Run("paramLoadable ctx is not holding ref to param", func(t *testing.T) {
		var paramLoadableCtx gokugen.SchedulerContext
		var called int64
		{
			param := "foobarbaz"
			paramUsed := &param
			runtime.SetFinalizer(paramUsed, func(*string) {
				atomic.AddInt64(&called, 1)
			})

			ctx := newTaskStorageCtx(gokugen.NewPlainContext(time.Now(), nil, make(map[any]any)), "foo", paramUsed)

			paramLoadableCtx = newtaskStorageParamLoadableCtx(
				ctx,
				func() (any, error) {
					return "barbarbar", nil
				},
				func(
					ctxCancelCh, taskCancelCh <-chan struct{},
					scheduled time.Time,
					param any) error {
					return nil
				},
				func(err error) {},
			)
		}

		if param, err := GetParam(paramLoadableCtx); err != nil || param != "barbarbar" {
			t.Fatalf("param is incorrect: %v", param)
		}

		for i := 0; i < 100; i++ {
			runtime.GC()
			if atomic.LoadInt64(&called) == 1 {
				break
			}
		}

		if atomic.LoadInt64(&called) != 1 {
			t.Fatalf("param is not dropped.")
		}
	})
}
