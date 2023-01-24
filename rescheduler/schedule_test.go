package rescheduler_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ngicks/gokugen/rescheduler"
	"github.com/ngicks/type-param-common/util"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

func TestSchedule_RescheduleRule(t *testing.T) {
	assert := assert.New(t)
	r := rescheduler.RescheduleRule{}

	var err error
	err = json.Unmarshal([]byte(`{}`), &r)
	assert.NoError(err)
	assert.Equal(0, len(r))

	err = json.Unmarshal([]byte(`{
		"foo": "CRON_TZ=Asia/Tokyo 0 5 * * *",
		"bar": {"schedule":"@every 5m", "n": 123},
		"baz": {"dur": 1000000000}
	}`), &r)
	assert.NoError(err)
	assert.Equal(3, len(r))

	{
		s, ok := r["foo"].(*rescheduler.CronSchedule)
		assert.True(ok, "wrong type = %T", r["foo"])
		_, ok = s.Spec.(*cron.SpecSchedule)
		assert.True(ok, "wrong type = %T", s.Spec)
	}
	{
		s, ok := r["bar"].(*rescheduler.LimitedSchedule)
		assert.True(ok, "wrong type = %T", r["bar"])
		assert.Equal(int64(123), s.N)
		c, ok := s.Schedule.(*rescheduler.CronSchedule)
		assert.True(ok, "wrong type = %T", s.Schedule)
		_, ok = c.Spec.(cron.ConstantDelaySchedule)
		assert.True(ok, "wrong type = %T", c.Spec)
	}
	{
		s, ok := r["baz"].(*rescheduler.IntervalSchedule)
		assert.True(ok, "wrong type = %T", r["baz"])
		assert.Equal(time.Second, s.Dur)
	}
}

func assertBinIsCronScheduleParam(t *testing.T, data []byte, now time.Time) (ok bool) {
	ok = true

	var p rescheduler.CronScheduleParam
	err := p.UnmarshalBinary(data)
	if err != nil {
		ok = false
		t.Errorf("must not be err: %+v", err)
	}
	assertTime := func(input time.Time) {
		if !now.Equal(input) {
			ok = false
			t.Errorf(
				"not equal: input = %s, want = %s",
				input, now,
			)
		}
	}
	assertTime(p.Next)
	assertTime(p.Prev)
	return
}

func TestInitial(t *testing.T) {
	t.Run("CronSchedule", func(t *testing.T) {
		var s rescheduler.CronSchedule
		now := time.Now()
		bin := s.Initial(now)
		assertBinIsCronScheduleParam(t, bin, now)
	})
	t.Run("LimitedSchedule", func(t *testing.T) {
		assert := assert.New(t)

		for _, v := range []rescheduler.LimitedSchedule{
			{
				Schedule: util.Must(rescheduler.UnmarshalSchedule([]byte("\"CRON_TZ=Asia/Tokyo 0 5 * * *\""))),
				N:        1,
			}, {
				Schedule: util.Must(rescheduler.UnmarshalSchedule([]byte("\"CRON_TZ=Asia/Tokyo 0 5 * * *\""))),
				N:        10,
			}, {
				Schedule: util.Must(rescheduler.UnmarshalSchedule([]byte("\"CRON_TZ=Asia/Tokyo 0 5 * * *\""))),
				N:        500,
			},
		} {
			now := time.Now()
			bin := v.Initial(now)

			var p rescheduler.LimitedScheduleParam
			err := p.UnmarshalBinary(bin)
			assert.NoError(err)
			assert.Equal(v.N, p.N)

			assertBinIsCronScheduleParam(t, p.Rest, now)
		}
	})
	t.Run("IntervalSchedule", func(t *testing.T) {
		now := time.Now()
		var s rescheduler.IntervalSchedule
		bin := s.Initial(now)
		assertBinIsCronScheduleParam(t, bin, now)
	})
}
