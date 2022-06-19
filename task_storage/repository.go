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
	// Id and LastModified field of passed TaskInfo are ignored.
	Insert(TaskInfo) (taskId string, err error)
	GetUpdatedAfter(time.Time) ([]TaskInfo, error)
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
