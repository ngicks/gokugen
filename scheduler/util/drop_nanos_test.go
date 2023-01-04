package util_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler/util"
	"github.com/stretchr/testify/require"
)

func TestDropNanos(t *testing.T) {
	require := require.New(t)
	var random = rand.New(rand.NewSource(time.Now().UnixMilli()))

	for i := 0; i < 1000; i++ {
		randomTime := time.Unix(random.Int63(), random.Int63())

		dropped := util.DropNanos(randomTime)

		require.Equal(randomTime.UnixMilli(), dropped.UnixMilli())
	}

	assertDropNanosReturnsSameTime := func(tt time.Time) {
		if !tt.Equal(util.DropNanos(tt)) {
			t.Fatalf("not equal. expected = %v, actual = %v", tt, util.DropNanos(tt))
		}
	}
	var tt time.Time
	tt, _ = time.Parse(time.RFC3339, "2023-01-01T15:56:52Z")
	assertDropNanosReturnsSameTime(tt)
	tt, _ = time.Parse(time.RFC3339, "2023-01-01T15:56:52.123Z")
	assertDropNanosReturnsSameTime(tt)
}
