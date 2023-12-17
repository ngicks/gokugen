package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/inmemory"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/assert"
)

var (
	mockErr        = fmt.Errorf("foo")
	currentTime    = time.Now().Truncate(time.Millisecond)
	currentTimeOpt = option.Some(currentTime)
)

func TestScheduler(t *testing.T) {
	assert := assert.New(t)

	type testCase struct {
		caseName      string
		repo          *mockRepository
		expectedState State
		handler       StepResultHandler
		action        func(
			ctx context.Context,
			cancel context.CancelFunc,
			mockRepository *mockRepository,
		)
	}
	for _, tc := range []testCase{
		{
			caseName: "timer update error",
			repo: &mockRepository{
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
			action: func(
				ctx context.Context,
				cancel context.CancelFunc,
				mockRepository *mockRepository,
			) {
			},
		},
		{
			caseName: "recover from timer update error",
			repo: &mockRepository{
				RepoErr:  []error{},
				TimerErr: []error{mockErr},
			},
			expectedState: AwaitingNext,
			handler: StepResultHandler{
				AwaitingNext: func(err error) error {
					assert.ErrorIs(err, context.Canceled)
					return nil
				},
			},
			action: func(
				ctx context.Context,
				cancel context.CancelFunc,
				mockRepository *mockRepository,
			) {
				cancel()
			},
		},
		{
			caseName: "awaiting next",
			repo: &mockRepository{
				RepoErr:  []error{},
				TimerErr: []error{},
			},
			expectedState: AwaitingNext,
			handler: StepResultHandler{
				AwaitingNext: func(err error) error {
					assert.ErrorIs(err, context.Canceled)
					return nil
				},
			},
			action: func(
				ctx context.Context,
				cancel context.CancelFunc,
				mockRepository *mockRepository,
			) {
				cancel()
			},
		},
		{
			caseName:      "last moment cancellation",
			repo:          &mockRepository{},
			expectedState: NextTask,
			handler: StepResultHandler{
				NextTask: func(task def.Task, err error) error {
					assert.True(def.IsExhausted(err), "must be Exhausted error.")
					return nil
				},
			},
			action: func(
				ctx context.Context,
				cancel context.CancelFunc,
				mockRepository *mockRepository,
			) {
				mockRepository.StartTimer(context.Background())
				mockRepository.Clock.Reset(currentTime.Sub(currentTime))
				mockRepository.Clock.Send()
			},
		},
		{
			caseName: "last moment timer stop",
			repo: &mockRepository{
				InMemoryRepository: newInMemoryRepository(
					func(repo *inmemory.InMemoryRepository) {
						_, err := repo.AddTask(context.Background(), def.TaskUpdateParam{
							WorkId:      option.Some("foo"),
							ScheduledAt: currentTimeOpt,
						})
						if err != nil {
							panic(err)
						}
					},
				),
			},
			expectedState: NextTask,
			handler: StepResultHandler{
				NextTask: func(task def.Task, err error) error {
					assert.ErrorIs(err, ErrScheduleStoppedOrChanged)
					return nil
				},
			},
			action: func(
				ctx context.Context,
				cancel context.CancelFunc,
				mockRepository *mockRepository,
			) {
				// timer is stopped
				mockRepository.Clock.Send()
			},
		},
		{
			caseName: "last moment timer update",
			repo: &mockRepository{
				InMemoryRepository: newInMemoryRepository(
					func(repo *inmemory.InMemoryRepository) {
						_, err := repo.AddTask(context.Background(), def.TaskUpdateParam{
							WorkId:      option.Some("foo"),
							ScheduledAt: currentTimeOpt,
						})
						if err != nil {
							panic(err)
						}
					},
				),
				NextScheduledTime: currentTime.Add(1),
			},
			expectedState: NextTask,
			handler: StepResultHandler{
				NextTask: func(task def.Task, err error) error {
					assert.ErrorIs(err, ErrScheduleStoppedOrChanged)
					return nil
				},
			},
			action: func(
				ctx context.Context,
				cancel context.CancelFunc,
				mockRepository *mockRepository,
			) {
				mockRepository.StartTimer(context.Background())
				mockRepository.Clock.Send()
			},
		},
		{
			caseName: "",
			repo: &mockRepository{
				InMemoryRepository: newInMemoryRepository(
					func(repo *inmemory.InMemoryRepository) {
						_, err := repo.AddTask(context.Background(), def.TaskUpdateParam{
							WorkId:      option.Some("foo"),
							ScheduledAt: currentTimeOpt,
						})
						if err != nil {
							panic(err)
						}
					},
				),
			},
			expectedState: NextTask,
			handler: StepResultHandler{
				NextTask: func(task def.Task, err error) error {
					assert.NoError(err)
					assert.NotEmpty(task.Id)
					assert.True(
						task.ScheduledAt.Equal(currentTime),
						"not equal. expected = %s, actual = %s", currentTime, task.ScheduledAt,
					)
					return nil
				},
			},
			action: func(
				ctx context.Context,
				cancel context.CancelFunc,
				mockRepository *mockRepository,
			) {
				mockRepository.StartTimer(context.Background())
				mockRepository.Clock.Send()
			},
		},
	} {
		if tc.repo.InMemoryRepository == nil {
			tc.repo.InMemoryRepository = inmemory.NewInMemoryRepository()
		}
		tc.repo.Clock = mockable.NewClockFake(time.Now())
		tc.handler = fillEmptyHandler(tc.handler)

		dispatcher := &mockDispatcher{
			Pending: make(map[string]pair),
		}

		scheduler := NewScheduler(tc.repo, dispatcher)

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
		tc.action(ctx, cancel, tc.repo)
		<-syncCh

		assert.Equal(tc.expectedState, state.State())
		assert.NoError(state.Match(tc.handler))
	}
}

func newInMemoryRepository(
	handler func(repo *inmemory.InMemoryRepository),
) *inmemory.InMemoryRepository {
	repo := inmemory.NewInMemoryRepository()
	handler(repo)
	return repo
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
