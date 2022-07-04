package common_test

import (
	"testing"
	"time"

	"github.com/ngicks/gokugen/common"
	"github.com/stretchr/testify/require"
)

func TestTimer(t *testing.T) {
	timer := common.NewTimerImpl()

	now := time.Now()
	timer.Reset(now, now)

	require.InDelta(t, now.UnixNano(), (<-timer.GetChan()).UnixNano(), float64(time.Millisecond))

	timer.Reset(now.Add(-time.Second), now)
	// emit fast.
	require.InDelta(t, now.UnixNano(), (<-timer.GetChan()).UnixNano(), float64(time.Millisecond))
}
