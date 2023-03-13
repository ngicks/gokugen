package scheduler

import (
	"fmt"
	"time"
)

// NeverExistentId is an id that is valid,
// but must never be stored or never be generated inside the TaskRepository.
// This sits here only for testing.
var NeverExistentId = "%%%%$$$$%%%%$$$$%%%%$$$$"

// TaskRepository is a combination of Repository and Timer interfaces.
type TaskRepository interface {
	RepositoryLike
	TimerLike
}

type RepositoryLike interface {
	// AddTask adds a task, which is configured by param, to the Repository.
	// An implementation can utilize its own id creation rule for Task.Id.
	// The implementation must not leave Id field empty.
	// It finally returns a Task created inside it.
	//
	// An implementation might check param validity by calling param.IsInitialized,
	// or might silently fall back to default ScheduledAt and/or WorkId values.
	AddTask(param TaskParam) (Task, error)
	GetById(id string) (Task, error)
	// Update updates a task with param.
	// Zero fields of param are considered as `no update` for that field.
	Update(id string, param TaskParam) (updated bool, err error)
	Cancel(id string) (cancelled bool, err error)

	MarkAsDispatched(id string) error
	// MarkAsDone marks the id as done. if err is non-nil, task is marked as failed.
	// Only tasks having been marked-as-dispatched can be marked-as-done.
	MarkAsDone(id string, err error) error

	// Find finds tasks matching to matcher.
	//
	// Zero fields of matcher are considered as empty search conditions,
	// placing no restriction on the field.
	Find(matcher TaskMatcher) ([]Task, error)
	// FindMetaContain finds tasks whose meta field satisfies matcher.
	//
	// An implementation must implement HasKey and Exact matching types to be a conformant.
	// Others are optional.
	FindMetaContain(matcher []KeyValuePairMatcher) ([]Task, error)
	// GetNext returns a next scheduled Task without changing repository contents.
	// GetNext must not return a cancelled, dispatched or done task.
	GetNext() (Task, error)
}

type KeyValuePairMatcher struct {
	Key     string
	Value   string
	MatchTy matchType
}

type matchType int

const (
	HasKey matchType = iota
	Exact
	Forward
	Backward
	Partial
)

func (t matchType) String() string {
	switch t {
	case HasKey:
		return "HasKey"
	case Exact:
		return "Exact"
	case Forward:
		return "Forward"
	case Backward:
		return "Backward"
	case Partial:
		return "Partial"
	}
	return fmt.Sprintf("<unknown: %d>", t)
}

type TimerLike interface {
	// StartTimer starts its internal timer.
	// A channel returned from TimerChannel emits only if it is started.
	// In started state, the timer channel updates to a next scheduled element
	// on start and at every Repository mutations.
	StartTimer()
	// StopTimer stops its internal timer.
	// The channel returned from TimerChannel would not emit after return of StopTimer,
	// unless StartTimer is called again.
	StopTimer()
	// TimerChannel returns the internal timer channel.
	// The timer can be either of started, or stopped state.
	// It will emits if and only if in started state.
	TimerChannel() <-chan time.Time
}

// DispatchedReverter is an optional interface for RepositoryLike implementations.
// A single process scheduler might use this on process start up.
type DispatchedReverter interface {
	// RevertDispatched sets nil to DispatchedAt field of those which are not cancelled, done, yet dispatched.
	// A single process scheduler may call this to recover abandoned tasks which might have been caused by
	// interrupt/kill signals or power failures.
	RevertDispatched() error
}

// BeforeDeleter is an optional interface for Repository implementations.
// This is a suggestion for an unified interface of deletion.
type BeforeDeleter interface {
	// DeleteBefore deletes done or cancelled tasks before before.
	// It returns Deleted with non-nil fields only if returning is true, otherwise all fields are nil.
	//
	// This package does not use this. This is here only for an unified interface of soft or hard deletion.
	DeleteBefore(before time.Time, returning bool) (Deleted, error)
}

type Deleted struct {
	Cancelled map[string]Task
	Done      map[string]Task
}
