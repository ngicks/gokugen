package scheduler

import "time"

type Task interface {
	Id() string
	TaskId() string
	ScheduledAt() time.Time
	IsCancelled() bool
	// IsDone returns true if the Task is dispatched at least once,
	// false otherwise.
	IsDone() bool
}
