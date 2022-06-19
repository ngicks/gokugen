package taskstorage

import (
	"fmt"
	"time"
)

type StateUpdater interface {
	// UpdateState updates task State to new if current state is old.
	// swapped is true if update is successful, false otherwise.
	UpdateState(taskId string, old, new TaskState) (swapped bool, err error)
}

type Updater interface {
	// Update updates the info with diff.
	// Implementation may limit updatable state.
	// If state is not updatable, return wrapped or unwrapped ErrNotUpdatableState.
	Update(taskId string, diff UpdateDiff) (err error)
}

type Repository interface {
	// Insert inserts the model to the repository.
	// Id and LastModified field of passed TaskInfo are ignored and will be newly created.
	Insert(TaskInfo) (taskId string, err error)
	// GetUpdatedAfter fetches all elements modified after t.
	// Implementation may limit the number of fetched elements.
	GetUpdatedAfter(t time.Time) ([]TaskInfo, error)
	// GetById fetches an element associated with the id.
	// If the id does not exist in the repository,
	// GetById reports it by returning wrapped  or unwrapped ErrNoEnt error.
	GetById(id string) (TaskInfo, error)
	// MarkAsDone marks the task as done.
	// Only Initialized or Working state must be updated to done state.
	MarkAsDone(id string) (ok bool, err error)
	// MarkAsCancelled marks the task as cancelled.
	// Only Initialized or Working state must be updated to done state.
	MarkAsCancelled(id string) (ok bool, err error)
	// MarkAsFailed marks the task as failed which means workers failed to do this task.
	// Only Initialized or Working state must be updated to done state.
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
