package scheduler

import "time"

// NeverExistenceId is a valid id
// but must never be stored or never be generated inside the TaskRepository.
// This is mainly for testing.
var NeverExistenceId = "%%%%$$$$%%%%$$$$%%%%$$$$"

type TaskRepository interface {
	RepositoryLike
	TimerLike
}

type RepositoryLike interface {
	AddTask(param TaskParam) (Task, error)
	GetById(id string) (Task, error)
	Update(id string, param TaskParam) (updated bool, err error)
	Cancel(id string) (cancelled bool, err error)

	MarkAsDispatched(id string) error
	// MarkAsDone marks the id as done. if err is non-nil, task is marked as failed.
	MarkAsDone(id string, err error) error

	// GetNext returns the next scheduled Task without changing repository contents.
	// GetNext should not return cancelled tasks.
	GetNext() (Task, error)
}

type TimerLike interface {
	// StartTimer starts the internal timer. A channel returned from TimerChannel emits only if it is started.
	// In started state, the timer channel updates to the next scheduled element at every Repository mutations.
	StartTimer()
	// StopTimer stops timer channels returned from TimerChannel.
	StopTimer()
	// TimerChannel returns the internal timer channel. The timer could be either of started, or stopped state.
	// It will emits if and only if in started state.
	TimerChannel() <-chan time.Time
}
