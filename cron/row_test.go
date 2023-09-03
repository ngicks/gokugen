package cron

import (
	"testing"
	"time"
	_ "time/tzdata"

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
			input:  "TZ=Asia/Tokyo @every 4h25m",
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
			input:  "TZ=Asia/Tokyo 1 2 3 4 5 *",
			expect: parseTime("2023-05-04T03:02:01.000000000+09:00"),
		},
		{
			input: JsonExp{
				Second: []uint64{15, 30},
			},
			expect: parseTime("2023-04-20T06:29:15Z"),
		},
		{
			input: JsonExp{
				Second:   []uint64{15, 30},
				Hour:     []uint64{6},
				Location: "Asia/Tokyo",
			},
			expect: parseTime("2023-04-21T06:00:15+09:00"),
		},
	} {
		sched, err := ParseRawExpression(tc.input)
		assert.NoError(err)
		next := sched.Next(fakeCurrent)
		assertTimeEqual(t, tc.expect.In(time.UTC), next)
	}

}
