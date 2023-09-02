package cron

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJsonExp(t *testing.T) {
	assert := assert.New(t)
	type testCase struct {
		input  RawExpression
		expect time.Time
	}
	for _, tc := range []testCase{
		{
			input:  "@every 4h25m",
			expect: fakeCurrent.Truncate(time.Second).Add(4*time.Hour + 25*time.Minute),
		},
		{
			input:  "1 2 3 4 *",
			expect: parseTime("2024-04-03T02:01:00.000000000Z"),
		},
		{
			input:  "1 2 3 4 5 *",
			expect: parseTime("2023-05-04T03:02:01.000000000Z"),
		},
		{
			input: JsonExp{
				Second: []uint64{15, 30},
			},
			expect: parseTime("2023-04-20T06:29:15Z"),
		},
	} {
		sched, err := ParseRawExpression(tc.input)
		assert.NoError(err)
		next := sched.Next(fakeCurrent)
		assertTimeEqual(t, tc.expect, next)
	}

}
