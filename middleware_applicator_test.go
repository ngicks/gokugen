package gokugen_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	mock_gokugen "github.com/ngicks/gokugen/__mock"
	"github.com/ngicks/gokugen/middleware/observe"
	"github.com/stretchr/testify/require"
)

func TestScheduler(t *testing.T) {
	t.Run("Use order is mw application order", func(t *testing.T) {
		_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

		schduler := gokugen.NewMiddlewareApplicator(mockSched)

		orderMu := sync.Mutex{}
		order := make([]string, 0)
		hook := func(label1, label2 string) (ctxObserver func(ctx gokugen.SchedulerContext), workFnObserver func(ret any, err error)) {
			ctxObserver = func(ctx gokugen.SchedulerContext) {
				orderMu.Lock()
				defer orderMu.Unlock()
				order = append(order, label1)
			}
			workFnObserver = func(ret any, err error) {
				orderMu.Lock()
				defer orderMu.Unlock()
				order = append(order, label2)
			}
			return
		}
		mw1 := observe.New(hook("ctx1", "work1"))
		mw2 := observe.New(hook("ctx2", "work2"))
		mw3 := observe.New(hook("ctx3", "work3"))
		mw4 := observe.New(hook("ctx4", "work4"))

		schduler.Use(mw1.Middleware)
		schduler.Use(mw2.Middleware)
		schduler.Use([]gokugen.MiddlewareFunc{mw3.Middleware, mw4.Middleware}...)

		schduler.Schedule(gokugen.NewPlainContext(
			time.Now(),
			func(taskCtx context.Context, scheduled time.Time) (any, error) { return nil, nil },
			nil,
		))

		orderMu.Lock()
		require.Equal(t, []string{"ctx1", "ctx2", "ctx3", "ctx4"}, order)
		orderMu.Unlock()

		getTrappedTask().Do(context.TODO(), func() {})

		orderMu.Lock()
		// observe action takes place *after* inner work is called.
		// Then call order is mw-applicaton order.
		require.Equal(t, []string{"ctx1", "ctx2", "ctx3", "ctx4", "work1", "work2", "work3", "work4"}, order)
		orderMu.Unlock()
	})
}
