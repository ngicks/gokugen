package deadline

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ngicks/gokugen"
	mock_gokugen "github.com/ngicks/gokugen/__mock"
	"github.com/ngicks/gokugen/middleware/observe"
	mock_gommon "github.com/ngicks/gommon/__mock"
	"github.com/stretchr/testify/require"
)

func TestDeadline(t *testing.T) {
	_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)
	ma := gokugen.NewMiddlewareApplicator(mockSched)

	var workErr error
	observeMw := observe.New(nil, func(_ any, err error) {
		workErr = err
	})
	// never dies
	deadlineMw := New(math.MaxInt64, func(ctx gokugen.SchedulerContext) bool { return false })

	ma.Use(
		deadlineMw.Middleware,
		observeMw.Middleware,
	)

	inputCtx := gokugen.BuildContext(
		time.Now(),
		func(taskCtx context.Context, scheduled time.Time) (any, error) {
			return "baz", nil
		},
		nil,
	)

	ma.Schedule(inputCtx)
	getTrappedTask().Do(context.TODO())

	require.Equal(t, nil, workErr)

	// dies however you try
	deadlineMw.deadline = 0
	ma.Schedule(inputCtx)
	getTrappedTask().Do(context.TODO())

	dee, ok := workErr.(DeadlineExeededErr)
	require.Equal(t, true, ok)
	require.Error(t, dee)
}

func TestDeadlineMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGetNow := mock_gommon.NewMockGetNower(ctrl)
	_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

	ma := gokugen.NewMiddlewareApplicator(mockSched)

	deadlineMw := New(time.Second, func(ctx gokugen.SchedulerContext) bool { return false })
	deadlineMw.getNow = mockGetNow
	var workErr error
	observeMw := observe.New(nil, func(ret any, err error) {
		workErr = err
	})

	ma.Use(
		deadlineMw.Middleware,
		observeMw.Middleware,
	)

	psuedoNow := time.Date(2000, time.April, 21, 12, 9, 54, 1, time.UTC)
	mockGetNow.EXPECT().GetNow().Return(psuedoNow.Add(500 * time.Millisecond))

	inputCtx := gokugen.BuildContext(
		psuedoNow,
		func(taskCtx context.Context, scheduled time.Time) (any, error) {
			return "baz", nil
		},
		nil,
	)

	ma.Schedule(inputCtx)
	getTrappedTask().Do(context.TODO())

	require.Equal(t, nil, workErr)

	mockGetNow.EXPECT().GetNow().Return(psuedoNow.Add(time.Second + 500*time.Millisecond))

	ma.Schedule(inputCtx)
	getTrappedTask().Do(context.TODO())

	dee, ok := workErr.(DeadlineExeededErr)
	require.Equal(t, true, ok)
	require.Equal(t, psuedoNow, dee.ScheduledTime())
	require.Equal(t, psuedoNow.Add(time.Second+500*time.Millisecond), dee.ExecutedTime())
}
