package taskstorage

import (
	"fmt"
	"time"
)

type StateUpdater interface {
	UpdateState(taskId string, old, new TaskState) (swapped bool, err error)
}

type Updater interface {
	Update(taskId string, diff UpdateDiff) (err error)
}

type Repository interface {
	// Insert inserts the model to the repository.
	// The id of passed TaskInfo will be ignored.
	// Insert returns a newly created id.
	Insert(TaskInfo) (taskId string, err error)
	// GetAll returns many models stored in the repository.
	// Implementation may or may not limit number of models.
	GetAll() ([]TaskInfo, error)
	GetById(id string) (TaskInfo, error)
	MarkAsDone(id string) (ok bool, err error)
	MarkAsCancelled(id string) (ok bool, err error)
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
