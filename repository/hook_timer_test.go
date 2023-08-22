package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/require"
)

var (
	fakeErr    = errors.New("fake")
	sampleDate = time.Date(
		2023, time.April, 18,
		21, 57, 42,
		123000000, time.UTC,
	)
	sampleTask = def.TaskUpdateParam{
		WorkId:      option.Some("foo"),
		ScheduledAt: option.Some(sampleDate),
	}.ToTask(def.NeverExistentId, time.Now())
)

func testHookTimer(t *testing.T, hookTimer HookTimer, fakeTimer *mockable.ClockFake) {
	var (
		dur time.Duration
		ctx = context.Background()
	)
	require := require.New(t)
	repo := newFakeRepository()

	hookTimer.SetRepository(repo)

	func() {
		defer func() {
			require.NotNil(recover(), "calling SetRepository twice must panic.")
		}()
		hookTimer.SetRepository(newFakeRepository())
	}()

	assertResetCalled := createAssertResetCalled(t, fakeTimer)
	assertStopCalled := createAssertStopCalled(t, fakeTimer)

	// empty
	hookTimer.StartTimer(context.Background())
	require.NoError(hookTimer.LastTimerUpdateError())
	assertResetCalled(false)
	assertStopCalled()

	hookTimer.StopTimer()
	assertStopCalled()

	repo.getErr = fakeErr
	hookTimer.StartTimer(context.Background())
	assertResetCalled(false)
	assertStopCalled()
	require.ErrorIs(
		hookTimer.LastTimerUpdateError(),
		fakeErr,
		"StartTimer must call repo.GetNext and store an error returned if it is non nil.",
	)

	repo.getErr = nil
	repo.next = sampleTask

	hookTimer.StartTimer(context.Background())
	require.NoError(hookTimer.LastTimerUpdateError())
	assertResetCalled(true)
	assertStopCalled()
	dur, _ = fakeTimer.LastReset()
	require.Equal(sampleTask.ScheduledAt.Sub(fakeTimer.Now()), dur)

	hookTimer.AddTask(ctx, def.TaskUpdateParam{
		ScheduledAt: option.Some(sampleDate.Add(time.Minute)),
	})
	assertResetCalled(false)

	hookTimer.AddTask(ctx, def.TaskUpdateParam{
		ScheduledAt: option.Some(sampleDate.Add(-time.Minute)),
	})
	assertResetCalled(true)
	assertStopCalled()

	type testSet struct {
		id         string
		param      def.TaskUpdateParam
		shouldRest bool
	}
	for _, ts := range []testSet{
		{
			id:         sampleTask.Id,
			param:      def.TaskUpdateParam{Priority: option.Some(5)},
			shouldRest: true,
		},
		{
			id:         sampleTask.Id,
			param:      def.TaskUpdateParam{Priority: option.Some(-1)},
			shouldRest: false,
		},
		{
			id: sampleTask.Id,
			param: def.TaskUpdateParam{
				ScheduledAt: option.Some(sampleTask.ScheduledAt.Add((-time.Minute))),
			},
			shouldRest: true,
		},
		{
			id: sampleTask.Id,
			param: def.TaskUpdateParam{
				ScheduledAt: option.Some(sampleTask.ScheduledAt.Add((time.Minute))),
			},
			shouldRest: false,
		},
		{
			id: sampleTask.Id + "bar",
			param: def.TaskUpdateParam{
				ScheduledAt: option.Some(sampleTask.ScheduledAt.Add((-time.Minute))),
			},
			shouldRest: true,
		},
		{
			id: sampleTask.Id + "bar",
			param: def.TaskUpdateParam{
				ScheduledAt: option.Some(sampleTask.ScheduledAt.Add((time.Minute))),
			},
			shouldRest: false,
		},
		{
			id: sampleTask.Id + "bar",
			param: def.TaskUpdateParam{
				Priority: option.Some(5),
			},
			shouldRest: true,
		},
		{
			id: sampleTask.Id + "bar",
			param: def.TaskUpdateParam{
				Priority: option.Some(-1),
			},
			shouldRest: false,
		},
	} {
		hookTimer.UpdateById(ctx, ts.id, ts.param)
		if !assertResetCalled(ts.shouldRest) {
			t.Logf("input params = %+#v", ts)
		}
	}
	hookTimer.Cancel(ctx, "bar")
	assertResetCalled(false)
	hookTimer.Cancel(ctx, sampleTask.Id)
	assertResetCalled(true)

	hookTimer.MarkAsDispatched(ctx, "bar")
	assertResetCalled(false)
	hookTimer.MarkAsDispatched(ctx, sampleTask.Id)
	assertResetCalled(true)
}

var _ def.Repository = (*fakeRepository)(nil)

type fakeRepository struct {
	next   def.Task
	getErr error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{}
}

func (r *fakeRepository) Close() error {
	return nil
}

func (r *fakeRepository) AddTask(ctx context.Context, param def.TaskUpdateParam) (def.Task, error) {
	return def.Task{}, nil
}
func (r *fakeRepository) GetById(ctx context.Context, id string) (def.Task, error) {
	return def.Task{}, nil
}
func (r *fakeRepository) UpdateById(
	ctx context.Context,
	id string,
	param def.TaskUpdateParam,
) error {
	return nil
}
func (r *fakeRepository) Cancel(ctx context.Context, id string) error {
	return nil
}
func (r *fakeRepository) MarkAsDispatched(ctx context.Context, id string) error {
	return nil
}
func (r *fakeRepository) MarkAsDone(ctx context.Context, id string, err error) error {
	return nil
}
func (r *fakeRepository) Find(
	ctx context.Context,
	matcher def.TaskQueryParam,
	offset, limit int,
) ([]def.Task, error) {
	return nil, nil
}
func (r *fakeRepository) GetNext(ctx context.Context) (def.Task, error) {
	if r.getErr != nil {
		return def.Task{}, r.getErr
	}
	if reflect.ValueOf(r.next).IsZero() {
		return def.Task{}, &def.RepositoryError{Kind: def.Exhausted}
	}
	return r.next, nil
}

func createAssertResetCalled(t *testing.T, fakeTimer *mockable.ClockFake) func(called bool) bool {
	filterNil := func(v []*time.Duration) []*time.Duration {
		modified := v[:0]
		for _, b := range v {
			if b != nil {
				modified = append(modified, b)
			}
		}
		return modified
	}

	before := filterNil(fakeTimer.CloneResetArg())

	return func(called bool) bool {
		t.Helper()
		after := filterNil(fakeTimer.CloneResetArg())
		didErr := false
		if (len(before)+1 != len(after)) == called {
			didErr = true
			if called {
				t.Errorf("must be reset.")
			} else {
				t.Error("must not be reset")
			}
		}
		before = after
		return !didErr
	}
}

func createAssertStopCalled(t *testing.T, fakeTimer *mockable.ClockFake) func() bool {
	filterNonNil := func(v []*time.Duration) []*time.Duration {
		modified := v[:0]
		for _, b := range v {
			if b == nil {
				modified = append(modified, b)
			}
		}
		return modified
	}

	before := filterNonNil(fakeTimer.CloneResetArg())

	return func() bool {
		t.Helper()
		after := filterNonNil(fakeTimer.CloneResetArg())
		didErr := false
		if !(len(before) != len(after)) {
			didErr = true
			t.Error("must be stopped")
		}
		before = after
		return !didErr
	}
}
