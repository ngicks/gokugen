package util_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def/util"
	"github.com/stretchr/testify/require"
)

func TestDropMicros(t *testing.T) {
	require := require.New(t)
	var random = rand.New(rand.NewSource(time.Now().UnixMilli()))

	for i := 0; i < 1000; i++ {
		randomTime := time.Unix(random.Int63(), random.Int63())

		dropped := util.DropMicros(randomTime)

		require.Equal(randomTime.UnixMilli(), dropped.UnixMilli())
	}

	assertDropMicrosReturnsSameTime := func(tt time.Time) {
		if !tt.Equal(util.DropMicros(tt)) {
			t.Fatalf("not equal. expected = %v, actual = %v", tt, util.DropMicros(tt))
		}
	}
	var tt time.Time
	tt, _ = time.Parse(time.RFC3339, "2023-01-01T15:56:52Z")
	assertDropMicrosReturnsSameTime(tt)
	tt, _ = time.Parse(time.RFC3339, "2023-01-01T15:56:52.123Z")
	assertDropMicrosReturnsSameTime(tt)
	assertDropMicrosReturnsSameTime(time.Time{})
}
