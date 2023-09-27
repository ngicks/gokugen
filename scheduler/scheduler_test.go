package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/inmemory"
	"github.com/ngicks/mockable"
	"github.com/stretchr/testify/assert"
)

var (
	mockErr = fmt.Errorf("foo")
)

func TestScheduler(t *testing.T) {
	assert := assert.New(t)

	type testCase struct {
		repo          mockRepository
		expectedState State
		handler       StepResultHandler
		action        func(ctx context.Context, cancel context.CancelFunc)
	}
	for _, tc := range []testCase{
		{
			repo: mockRepository{
				RepoErr:  []error{},
				TimerErr: []error{mockErr, mockErr},
			},
			expectedState: TimerUpdateError,
			handler: StepResultHandler{
				TimerUpdateError: func(err error) error {
					assert.ErrorIs(err, mockErr)
					return nil
				},
			},
			action: func(ctx context.Context, cancel context.CancelFunc) {},
		},
		{
			repo: mockRepository{
				RepoErr:  []error{},
				TimerErr: []error{mockErr},
			},
			expectedState: AwaitingNext,
			handler: StepResultHandler{
				AwaitingNext: func(err error) error {
					time.Sleep(time.Millisecond)
					assert.ErrorIs(err, context.Canceled)
					return nil
				},
			},
			action: func(ctx context.Context, cancel context.CancelFunc) {
				cancel()
			},
		},
	} {
		tc.repo.InMemoryRepository = inmemory.NewInMemoryRepository()
		tc.repo.Clock = mockable.NewClockFake(time.Now())
		tc.handler = fillEmptyHandler(tc.handler)

		dispatcher := &mockDispatcher{
			Pending: make(map[string]pair),
		}

		scheduler := NewScheduler(&tc.repo, dispatcher)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var state StepState
		syncCh := make(chan struct{})
		go func() {
			<-syncCh
			state = scheduler.Step(ctx)
			syncCh <- struct{}{}
		}()
		syncCh <- struct{}{}
		tc.action(ctx, cancel)
		<-syncCh

		assert.Equal(tc.expectedState, state.State())
		assert.NoError(state.Match(tc.handler))
	}
}

func fillEmptyHandler(handler StepResultHandler) StepResultHandler {
	if handler.TimerUpdateError == nil {
		handler.TimerUpdateError = func(err error) error { panic("handler is not defined") }
	}
	if handler.AwaitingNext == nil {
		handler.AwaitingNext = func(err error) error { panic("handler is not defined") }
	}
	if handler.NextTask == nil {
		handler.NextTask = func(task def.Task, err error) error { panic("handler is not defined") }
	}
	if handler.DispatchErr == nil {
		handler.DispatchErr = func(task def.Task, err error) error {
			panic("handler is not defined")
		}
	}
	if handler.Dispatched == nil {
		handler.Dispatched = func(id string) error { panic("handler is not defined") }
	}
	if handler.TaskDone == nil {
		handler.TaskDone = func(id string, taskErr error, updateErr error) error {
			panic("handler is not defined")
		}
	}
	return handler
}
