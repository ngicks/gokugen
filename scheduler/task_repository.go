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

	// GetNext returns the most prioritized element without changing repository contents.
	// GetNext should not return cancelled tasks.
	GetNext() (Task, error)
	TimerLike
}

type TimerLike interface {
	StartTimer()
	StopTimer()
	TimerChannel() <-chan time.Time
}
