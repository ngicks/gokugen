package acceptancetest

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gokugen/scheduler/util"
	"github.com/ngicks/gommon/pkg/randstr"
)

var randGen *randstr.Generator = randstr.New(randstr.EncoderFactory(hex.NewEncoder))

func RandByte() []byte {
	bytes, err := randGen.Bytes()
	if err != nil {
		panic(err)
	}
	return bytes
}

func RandByteLen(length int) []byte {
	bytes, err := randGen.BytesLen(int64(length))
	if err != nil {
		panic(err)
	}
	return bytes
}

func RandStrLen(length int) string {
	str, err := randGen.StringLen(int64(length))
	if err != nil {
		panic(err)
	}
	return str
}

func IsTaskBasedOnParam(task scheduler.Task, param scheduler.TaskParam) bool {
	var p int
	if param.Priority != nil {
		p = *param.Priority
	}

	return util.TimeEqual(task.ScheduledAt, param.ScheduledAt) &&
		task.WorkId == param.WorkId &&
		bytes.Equal(task.Param, param.Param) &&
		task.Priority == p
}

func IsTaskInitial(task scheduler.Task, nearNow time.Time) bool {
	return task.Id != "" &&
		task.WorkId != "" &&
		// param can be nil or empty slice.
		// priority can be zero or whatever.
		IsTimeNearNow(task.CreatedAt, nearNow) &&
		task.CancelledAt == nil &&
		task.DispatchedAt == nil &&
		task.DoneAt == nil &&
		task.Err == ""
}

func IsTimeNearNow(t, now time.Time) bool {
	return IsTimeAfterAndWithinRange(t, now, time.Second)
}

func IsTimeAfterAndWithinRange(t time.Time, rangeStart time.Time, d time.Duration) bool {
	return t.Equal(rangeStart) || t.After(rangeStart.Add(-10*time.Millisecond)) && t.Before(rangeStart.Add(d))
}

func TruncatedNow() time.Time {
	return time.Now().Truncate(time.Millisecond)
}
