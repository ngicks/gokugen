package gokugen_test

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/scheduler"
)

func mockMw(hook func(ctx gokugen.SchedulerContext) gokugen.SchedulerContext) gokugen.MiddlewareFunc {
	return func(shf gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
		return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
			newCtx := hook(ctx)
			if newCtx != nil {
				return shf(newCtx)
			}
			return shf(ctx)
		}
	}
}

func TestScheduler(t *testing.T) {
	t.Run("Use order is mw application order", func(t *testing.T) {
		innerScheduler := scheduler.NewScheduler(1, 0)
		schduler := gokugen.NewScheduler(innerScheduler)

		orderMu := sync.Mutex{}
		order := make([]string, 0)
		hook := func(label string) func(ctx gokugen.SchedulerContext) gokugen.SchedulerContext {
			return func(ctx gokugen.SchedulerContext) gokugen.SchedulerContext {
				orderMu.Lock()
				defer orderMu.Unlock()
				order = append(order, label)
				return nil
			}
		}
		mw1 := mockMw(hook("1"))
		mw2 := mockMw(hook("2"))
		mw3 := mockMw(hook("3"))
		mw4 := mockMw(hook("4"))

		schduler.Use(mw1)
		schduler.Use(mw2)
		schduler.Use([]gokugen.MiddlewareFunc{mw3, mw4}...)

		schduler.Schedule(gokugen.NewPlainContext(
			time.Now(),
			func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) { return nil, nil },
			nil,
		))

		orderMu.Lock()
		if !reflect.DeepEqual(order, []string{"1", "2", "3", "4"}) {
			t.Fatalf("incorrect mw application order. %v", order)

		}
		orderMu.Unlock()

	})

}
