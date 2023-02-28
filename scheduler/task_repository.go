package scheduler

import "time"

// NeverExistentId is an id that is valid,
// but must never be stored or never be generated inside the TaskRepository.
// This is mainly for testing.
var NeverExistentId = "%%%%$$$$%%%%$$$$%%%%$$$$"

// TaskRepository is a combination of Repository and Timer interfaces.
type TaskRepository interface {
	RepositoryLike
	TimerLike
}

type RepositoryLike interface {
	// AddTask adds a task based on param to the Repository, returning created Task.
	// An implementation can utilize its own id creation rule for Task.Id.
	// The implementation must not leave Id field empty.
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
