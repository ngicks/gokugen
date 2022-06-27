package cron_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ngicks/gokugen/cron"

	mock_cron "github.com/ngicks/gokugen/cron/__mock"
	"github.com/stretchr/testify/require"
)

func TestScheduleState(t *testing.T) {
	nextScheduler := cron.Duration{
		Duration: time.Second,
		Command:  []string{"foo"},
	}
	now := time.Now()
	state := cron.NewScheduleState(nextScheduler, now)

	for i := 0; i < 100; i++ {
		same, count, next := state.Next()
		require.Equal(t, false, same)
		require.Equal(t, i, count)
		require.Equal(t, now.Add(time.Duration(i+1)*time.Second).Equal(next), true)
	}

	mockNextScheduler := mock_cron.NewMockNextScheduler(gomock.NewController(t))
	mockNextScheduler.
		EXPECT().
		NextSchedule(gomock.Any()).
		DoAndReturn(func(now time.Time) (time.Time, error) {
			return time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC), nil
		}).
		AnyTimes()
	state = cron.NewScheduleState(mockNextScheduler, now)

	for i := 0; i < 100; i++ {
		same, count, next := state.Next()
		if i == 0 {
			require.Equal(t, false, same)
		} else {
			require.Equal(t, true, same)
		}
		require.Equal(t, i, count)
		require.Equal(t, time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC).Equal(next), true)
	}
}
