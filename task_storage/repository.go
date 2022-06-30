package taskstorage

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrNoEnt is used in Repository implementation
	// to express situation where TaskInfo for operation does not exist.
	ErrNoEnt = errors.New("no ent")
	// ErrInvalidEnt is used in Repository implementation.
	// This is indication of an invalid TaskInfo insertion or a stored info is invalid.
	ErrInvalidEnt = errors.New("invalid ent")
	// ErrNotUpdatableState is used in Repository implementation.
	// It is returned when TaskInfo for the update is not updatable.
	ErrNotUpdatableState = errors.New("not updatable")
)

type StateUpdater interface {
	// UpdateState updates task State to new if current state is old.
	// swapped is true if update is successful, false otherwise.
	UpdateState(taskId string, old, new TaskState) (swapped bool, err error)
}

type Updater interface {
	// Update updates the info with diff.
	// Implementation may limit updatable state.
	// If state is not updatable, return wrapped or direct ErrNotUpdatableState.
	Update(taskId string, diff UpdateDiff) (err error)
}

type Repository interface {
	// Insert inserts given model to the repository.
	// Id and LastModified field are ignored and will be newly created.
	Insert(TaskInfo) (taskId string, err error)
	// GetUpdatedSince fetches all elements modified after or equal to t.
	// Result must be ordered by LastModified in ascending order.
	// Implementation may limit the number of fetched elements.
	// Implementation may or may not unmarshal Param to any. Fetched Param may be []byte or naive
	GetUpdatedSince(t time.Time) ([]TaskInfo, error)
	// GetById fetches an element associated with the id.
	// If the id does not exist in the repository,
	// GetById reports it by returning wrapped  or unwrapped ErrNoEnt error.
	// Implementation may or may not unmarshal Param to any.
	GetById(id string) (TaskInfo, error)
	// MarkAsDone marks the task as done.
	// Other than Initialized or Working state is not updatable to done.
	MarkAsDone(id string) (ok bool, err error)
	// MarkAsCancelled marks the task as cancelled.
	// Other than Initialized or Working state is not updatable to cancelled.
	MarkAsCancelled(id string) (ok bool, err error)
	// MarkAsFailed marks the task as failed which means workers failed to do this task.
	// Other than Initialized or Working state is not updatable to failed.
	MarkAsFailed(id string) (ok bool, err error)
}

type TaskState int

const (
	Initialized TaskState = iota
	Working
	Done
	Cancelled
	Failed
)

func NewStateFromString(s string) TaskState {
	switch s {
	case "Initialized":
		return Initialized
	case "Working":
		return Working
	case "Done":
		return Done
	case "Cancelled":
		return Cancelled
	case "Failed":
		return Failed
	}
	return -1
}

func (ts TaskState) String() string {
	switch ts {
	case Initialized:
		return "Initialized"
	case Working:
		return "Working"
	case Done:
		return "Done"
	case Cancelled:
		return "Cancelled"
	case Failed:
		return "Failed"
	default:
		return fmt.Sprintf("unknown=%d", ts)
	}
}

type TaskInfo struct {
	Id            string
	WorkId        string
	Param         any
	ScheduledTime time.Time
	State         TaskState
	LastModified  time.Time
}

type UpdateKey struct {
	WorkId        bool
	Param         bool
	ScheduledTime bool
	State         bool
}

type UpdateDiff struct {
	UpdateKey UpdateKey
	Diff      TaskInfo
}
