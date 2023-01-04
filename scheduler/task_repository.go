package scheduler

import "time"

// NeverExistenceId is a valid id
// but must never be stored or never be generated inside the TaskRepository.
// This is mainly for testing.
var NeverExistenceId = "%%%%$$$$%%%%$$$$%%%%$$$$"

type TaskRepository interface {
	AddTask(param TaskParam) (Task, error)
	GetById(id string) (Task, error)
	Update(id string, param TaskParam) (updated bool, err error)
	Cancel(id string) (cancelled bool, err error)

	MarkAsDispatched(id string) error
	// MarkAsDone marks the id as done. if err is non-nil, task is marked as failed.
	MarkAsDone(id string, err error) error

	// Peek peeks a min (most prioritized) element from TaskRepository without changing it.
	// Implementations might use a cached content for its return value. The caller will not mutate returned Task.
	// In case of error, the implementation may return nil pointer.
	Peek() *Task
	// Pop pops a min (most prioritized) element from TaskRepository.
	// The implementation may or may not remove the element from it, since the caller might fail to dispatch.
	Pop() (Task, error)
	TimerLike
}

type TimerLike interface {
	StartTimer()
	StopTimer()
	TimerChannel() <-chan time.Time
}
