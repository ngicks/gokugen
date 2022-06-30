package recover_test

import (
	"context"
	"testing"
	"time"

	"github.com/ngicks/gokugen"

	mockgokugen "github.com/ngicks/gokugen/__mock"
	"github.com/ngicks/gokugen/middleware/observe"
	recovermw "github.com/ngicks/gokugen/middleware/recover"
)

func TestRecoverMw(t *testing.T) {
	_, mockSched, getTrappedTask := mockgokugen.BuildMockScheduler(t)

	ma := gokugen.NewMiddlewareApplicator(mockSched)

	ma.Schedule(
		gokugen.NewPlainContext(
			time.Now(),
			func(taskCtx context.Context, scheduled time.Time) (any, error) {
				panic("mock pacnic")
			},
			nil,
		),
	)
	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatalf("must not be recovered")
			}
		}()
		getTrappedTask().Do(context.TODO(), func() {})
	}()

	observer := observe.New(nil, func(ret any, err error) {
		if _, ok := err.(*recovermw.RecoveredError); !ok {
			t.Fatalf("unknown erro type: %T, %v", err, err)
		}
	})
	ma.Use(recovermw.New().Middleware, observer.Middleware)

	ma.Schedule(
		gokugen.NewPlainContext(
			time.Now(),
			func(taskCtx context.Context, scheduled time.Time) (any, error) {
				panic("mock pacnic")
			},
			nil,
		),
	)
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("must be recovered")
			}
		}()
		getTrappedTask().Do(context.TODO(), func() {})
	}()
}
