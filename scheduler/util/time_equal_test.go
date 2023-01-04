package util_test

import (
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler/util"
	"github.com/stretchr/testify/assert"
)

func TestTimeEqual(t *testing.T) {
	assert := assert.New(t)

	tt, err := time.Parse(time.RFC3339Nano, "2023-01-01T05:27:30.123456789Z")
	if err != nil {
		panic(err)
	}

	u, err := time.Parse(time.RFC3339Nano, "2023-01-01T05:27:30.123Z")
	if err != nil {
		panic(err)
	}

	assert.True(util.DropNanos(tt).Equal(u), "DropNanos")
	assert.True(util.TimeEqual(tt, u), "TimeEqual")
	assert.True(util.DropNanosPointer(&tt).Equal(u), "DropNanosPointer")
	assert.True(!util.TimePointerEqual(&tt, &u, false), "TimeEqualPointer, ignoreMilli = false")
	assert.True(util.TimePointerEqual(&tt, &u, true), "TimeEqualPointer, ignoreMilli = true")
}

func TestTimeEqualPointer(t *testing.T) {
	assert := assert.New(t)

	now := time.Now()
	nowWithoutNanos := util.DropNanos(now)

	for _, testCase := range []struct {
		l           *time.Time
		r           *time.Time
		ignoreMilli bool
		expected    bool
	}{
		{nil, nil, false, true},
		{nil, &now, false, false},
		{nil, &now, true, false},
		{&now, nil, false, false},
		{&now, nil, true, false},
		{&now, &now, false, true},
		{&now, &now, true, true},
		{&nowWithoutNanos, &nowWithoutNanos, false, true},
		{&nowWithoutNanos, &nowWithoutNanos, true, true},
		{&now, &nowWithoutNanos, false, false},
		{&nowWithoutNanos, &now, false, false},
		{&now, &nowWithoutNanos, true, true},
		{&nowWithoutNanos, &now, true, true},
	} {
		assert.True(
			util.TimePointerEqual(
				testCase.l,
				testCase.r,
				testCase.ignoreMilli,
			) == testCase.expected,
			"expected to be %t, left = %v, right = %v, ignoreMilli = %t",
			testCase.expected,
			testCase.l,
			testCase.r,
			testCase.ignoreMilli,
		)
	}
}
