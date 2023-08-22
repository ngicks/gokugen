package def

import (
	"context"
	"time"
)

// NeverExistentId is an id that is valid,
// but must never be stored or never be generated inside the TaskRepository.
// This sits here only for testing.
var NeverExistentId = "%%%%$$$$%%%%$$$$%%%%$$$$"

// ObservableRepository is a combination of Repository and Timer interfaces.
type ObservableRepository interface {
	Repository
	Observer
}

//nolint:lll
type Repository interface {
	// Close closes this repository.
	Close() error
	// AddTask adds a task, which is configured by param, to this Repository.
	// Implementations can utilize their own id creation rule for Task.Id.
	// Implementations must not leave Id field empty.
	// It finally returns a Task created inside it.
	//
	// Implementations must test param validity by calling param.ToTask().IsValid().
	// If param is invalid, AddTask returns ErrInvalidTask or
	// an error which includes ErrInvalidTask in its error chain.
	AddTask(ctx context.Context, param TaskUpdateParam) (Task, error)
	GetById(ctx context.Context, id string) (Task, error)
	// UpdateById updates a task specified by id with param.
	// Every None fields are considered as `no update` for that field.
	UpdateById(ctx context.Context, id string, param TaskUpdateParam) error
	// Cancel changes a task specified by id to cancelled state.
	// It only succeeds when the task is not yet dispatched nor done.
	Cancel(ctx context.Context, id string) error

	MarkAsDispatched(ctx context.Context, id string) error
	// MarkAsDone marks the id as done. if err is non-nil, task is marked as failed.
	// Only tasks having been marked-as-dispatched can be marked-as-done.
	MarkAsDone(ctx context.Context, id string, err error) error

	// Find finds tasks matching to matcher.
	//
	// Every None fields are considered as empty search conditions.
	Find(ctx context.Context, matcher TaskQueryParam, offset, limit int) ([]Task, error)
	// GetNext returns a next scheduled Task without changing repository contents.
	// GetNext must not return a cancelled, dispatched or done task.
	GetNext(ctx context.Context) (Task, error)
}

type Observer interface {
	LastTimerUpdateError() error
	// StartTimer starts its internal timer.
	// A channel returned from TimerChannel emits only if it is started.
	// In started state, the timer channel updates to a next scheduled element
	// on start and at every Repository mutations.
	StartTimer(ctx context.Context)
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
	// RevertDispatched restores states as if
	// tasks were not yet dispatched if task is not done nor cancelled.
	// A single process scheduler may call this to recover abandoned tasks,
	// which might have been caused by interrupt/kill signals or power failures.
	RevertDispatched(ctx context.Context) error
}

// BeforeDeleter is an optional interface for Repository implementations.
// This is a suggestion for an unified interface of deletion.
//
//nolint:lll
type BeforeDeleter interface {
	// DeleteBefore deletes tasks become done or cancelled before before.
	// It returns Deleted with non-nil fields only if returning is true,
	// otherwise all fields are nil.
	//
	// This package does not use this.
	// This is here only for an unified interface of soft or hard deletion.
	DeleteBefore(ctx context.Context, before time.Time, meta map[string]string, returning bool) (Deleted, error)
}

type Deleted struct {
	Cancelled map[string]Task
	Done      map[string]Task
}
