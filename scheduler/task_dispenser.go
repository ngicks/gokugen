package scheduler

import "time"

type TaskDispenser interface {
	// MarkStart marks the TaskDispenser started.
	// This must not block long.
	MarkStart()
	// Stop stops TaskDispenser.
	// This may take time until it fully stops.
	Stop()
	// TaskBefore fetches a task from the dispenser.
	TaskBefore(t time.Time) (Task, error)
	Peek() (Task, error)
	// Timer returns timer channel
	// which will be emitted when tasks are past their scheduled time.
	Timer() <-chan time.Time
	RemoveCancelled() error
}
