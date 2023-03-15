package repository

import (
	"fmt"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

var _ scheduler.TaskRepository = (*Repository[scheduler.RepositoryLike])(nil)

// Repository is a thin wrapper that connects scheduler.RepositoryLike and HookTimer.
// Every mutation is observed and that would update the timer if necessary.
type Repository[T scheduler.RepositoryLike] struct {
	Core      T
	HookTimer HookTimer
}

func New[T scheduler.RepositoryLike](core T, hookTimer HookTimer) *Repository[T] {
	r := &Repository[T]{
		Core:      core,
		HookTimer: hookTimer,
	}
	r.HookTimer.SetRepository(r.Core)
	return r
}

func (r *Repository[T]) AddTask(param scheduler.TaskParam) (scheduler.Task, error) {
	if !param.IsInitialized() {
		return scheduler.Task{}, fmt.Errorf("input param is not initialized")
	}

	t, err := r.Core.AddTask(param)
	if err != nil {
		return scheduler.Task{}, err
	}

	r.HookTimer.AddTask(param)

	return t, nil
}

func (r *Repository[T]) Cancel(id string) (cancelled bool, err error) {
	cancelled, err = r.Core.Cancel(id)
	if err == nil {
		r.HookTimer.Cancel(id)
	}
	return cancelled, err
}

func (r *Repository[T]) GetById(id string) (scheduler.Task, error) {
	return r.Core.GetById(id)
}

func (r *Repository[T]) GetNext() (scheduler.Task, error) {
	return r.Core.GetNext()
}

func (r *Repository[T]) MarkAsDispatched(id string) error {
	err := r.Core.MarkAsDispatched(id)
	if err == nil {
		r.HookTimer.MarkAsDispatched(id)
	}
	return err
}

func (r *Repository[T]) MarkAsDone(id string, err error) error {
	return r.Core.MarkAsDone(id, err)
}

func (r *Repository[T]) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	updated, err = r.Core.Update(id, param)
	if err == nil {
		r.HookTimer.Update(id, param)
	}
	return updated, err
}

func (r *Repository[T]) Find(matcher scheduler.TaskMatcher) ([]scheduler.Task, error) {
	return r.Core.Find(matcher)
}

func (r *Repository[T]) FindMetaContain(matcher []scheduler.KeyValuePairMatcher) ([]scheduler.Task, error) {
	return r.Core.FindMetaContain(matcher)
}

func (r *Repository[T]) RevertDispatched() error {
	if del, ok := any(r.Core).(scheduler.DispatchedReverter); ok {
		return del.RevertDispatched()
	}
	// Should we panic on this?
	return nil
}

func (r *Repository[T]) DeleteBefore(before time.Time, returning bool) (scheduler.Deleted, error) {
	if del, ok := any(r.Core).(scheduler.BeforeDeleter); ok {
		return del.DeleteBefore(before, returning)
	}
	// Should we panic on this?
	return scheduler.Deleted{
		Cancelled: make(map[string]scheduler.Task),
		Done:      make(map[string]scheduler.Task),
	}, nil
}

func (r *Repository[T]) StartTimer() {
	r.HookTimer.StartTimer()
}

func (r *Repository[T]) StopTimer() {
	r.HookTimer.StopTimer()
}

func (r *Repository[T]) TimerChannel() <-chan time.Time {
	return r.HookTimer.TimerChannel()
}
