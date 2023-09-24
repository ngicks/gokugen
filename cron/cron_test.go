package cron

import (
	"context"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/mutator"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/assert"
)

var (
	fakeCurrent = parseTime("2023-04-20T06:29:02.123000000Z")
)

func TestCron(t *testing.T) {
	assert := assert.New(t)

	raw := []RowRaw{
		{
			Param: def.TaskUpdateParam{
				WorkId: option.Some("foo"),
			},
			Schedule: "@every 4h25m",
		},
		{
			Param: def.TaskUpdateParam{
				WorkId: option.Some("bar"),
			},
			Schedule: "0,5,10 6,7 * * *",
		},
		{
			Param: def.TaskUpdateParam{
				WorkId: option.Some("baz"),
			},
			Schedule: "30 6,7 * * *",
		},
		{
			Param: def.TaskUpdateParam{
				WorkId: option.Some("qux"),
			},
			Schedule: "0 0 21 * *",
		},
	}

	rows := make([]Row, len(raw))

	for idx := range raw {
		var err error
		rows[idx], err = raw[idx].Parse()
		if err != nil {
			panic(err)
		}
	}

	ents := make([]*Entry, len(rows))
	for idx := range rows {
		ents[idx] = NewEntry(fakeCurrent, rows[idx])
	}

	table, err := NewCronTable(ents)
	if err != nil {
		panic(err)
	}
	fakeClock := mockable.NewClockFake(fakeCurrent)
	table.clock = fakeClock

	task, err := table.Peek(context.Background())
	if err != nil {
		panic(err)
	}

	assert.Equal(task.WorkId, "baz")
	assertTimeEqual(t, parseTime("2023-04-20T06:30:00Z"), task.ScheduledAt)

	table.StartTimer(context.Background())

	for _, expected := range []def.Task{
		{WorkId: "baz", ScheduledAt: parseTime("2023-04-20T06:30:00Z")},
		{WorkId: "bar", ScheduledAt: parseTime("2023-04-20T07:00:00Z")},
		{WorkId: "bar", ScheduledAt: parseTime("2023-04-20T07:05:00Z")},
		{WorkId: "bar", ScheduledAt: parseTime("2023-04-20T07:10:00Z")},
		{WorkId: "baz", ScheduledAt: parseTime("2023-04-20T07:30:00Z")},
		// it trunc milli sec or finer time.
		{WorkId: "foo", ScheduledAt: parseTime("2023-04-20T10:54:02Z")},
		{WorkId: "foo", ScheduledAt: parseTime("2023-04-20T15:19:02Z")},
		{WorkId: "foo", ScheduledAt: parseTime("2023-04-20T19:44:02Z")},
		{WorkId: "qux", ScheduledAt: parseTime("2023-04-21T00:00:00Z")},
	} {
		lastResetDur, _ := fakeClock.LastReset()
		assert.Equal(expected.ScheduledAt.Sub(fakeCurrent), lastResetDur)

		task, err := table.Pop(context.Background())
		assert.NoError(err)
		assert.Equal(expected.WorkId, task.WorkId)
		assertTimeEqual(t, expected.ScheduledAt, task.ScheduledAt)
	}
}

func parseTime(timeStr string) time.Time {
	v, _ := time.Parse(time.RFC3339Nano, timeStr)
	return v
}

func assertTimeEqual(t *testing.T, l, r time.Time) bool {
	t.Helper()

	if !l.Equal(r) {
		t.Errorf(
			"not equal. left = %s, right = %s",
			l.Format(time.RFC3339Nano), r.Format(time.RFC3339Nano),
		)
		return false
	}
	return true
}

func TestCron_default_mutator_store(t *testing.T) {
	assert := assert.New(t)

	fakeCurrent := parseTime("2023-05-17T12:00:00Z")
	row, _ := RowRaw{
		Param: def.TaskUpdateParam{
			WorkId: option.Some("foo"),
			Meta: option.Some(map[string]string{
				mutator.LabelRandomizeScheduledAtMin: "-1s",
				mutator.LabelRandomizeScheduledAtMax: "1s",
			}),
		},
		Schedule: "@every 1h",
	}.Parse()

	table, err := NewCronTable([]*Entry{NewEntry(fakeCurrent, row)})
	if err != nil {
		panic(err)
	}

	type underSec struct {
		sec, nanoSec int
	}

	nanoSecMap := make(map[underSec]struct{}, 100)

	for i := 0; i < 100; i++ {
		task, err := table.Pop(context.Background())
		assert.NoError(err)

		assertTimeEqual(
			t,
			fakeCurrent.Add(time.Duration(i+1)*time.Hour),
			task.ScheduledAt.Add(time.Second).Truncate(time.Hour),
		)

		min := task.ScheduledAt.Minute()
		if min != 0 && min != 59 {
			t.Errorf("not within range. min must be 0 or 59 but is %d", min)
		}

		sec := task.ScheduledAt.Second()
		nanoSec := task.ScheduledAt.Nanosecond()
		if sec != 0 && sec != 59 {
			t.Errorf("not within range. sec must be 0 or 59 but is %d", sec)
		}
		if nanoSec%(1e6) != 0 {
			t.Errorf("time must be normalized. but is not. nano sec = %d", nanoSec)
		}
		nanoSecMap[underSec{sec, nanoSec}] = struct{}{}
	}

	assert.Greater(len(nanoSecMap), 50)
}
