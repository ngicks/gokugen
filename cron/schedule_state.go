package cron

import (
	"time"
)

type NextScheduler interface {
	NextSchedule(now time.Time) (time.Time, error)
}

// ScheduleState is a simple state storage for NextScheduler.
//
// All methods of ScheduleState are not concurrent-safe. Multiple goroutine must not call them directly.
type ScheduleState struct {
	prevTime  time.Time
	schedTime NextScheduler
	callCount int
}

// NewScheduleState creates new ScheduleState.
// schedTime is schedule-time calculator implementation. NextSchedule is called with whence or,
// for twice or later time, previous output of the method.
// whence is the time whence calcuation starts. Next sticks to its location.
func NewScheduleState(schedTime NextScheduler, whence time.Time) *ScheduleState {
	return &ScheduleState{
		schedTime: schedTime,
		prevTime:  whence,
	}
}

// Next returns next schedule time.
// If `same` is false, `next` is larger value than previously returned value.
// Or otherwise it is same time.
func (s *ScheduleState) Next() (same bool, callCount int, next time.Time) {
	next, _ = s.schedTime.NextSchedule(s.prevTime)
	if next.Equal(s.prevTime) {
		same = true
	}
	next = next.In(s.prevTime.Location())
	s.prevTime = next
	callCount = s.callCount
	s.callCount++
	return
}
