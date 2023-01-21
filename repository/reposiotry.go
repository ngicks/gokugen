package repository

import (
	"fmt"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

// Repository is a thin wrapper that connects scheduler.Repository and HookTimer.
// Every mutation is observed and updates timer if necessary.
type Repository struct {
	Core      scheduler.RepositoryLike
	HookTimer HookTimer
}

func (r *Repository) AddTask(param scheduler.TaskParam) (scheduler.Task, error) {
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

func (r *Repository) Cancel(id string) (cancelled bool, err error) {
	cancelled, err = r.Core.Cancel(id)
	if err == nil {
		r.HookTimer.Cancel(id)
	}
	return cancelled, err
}
func (r *Repository) GetById(id string) (scheduler.Task, error) {
	t, err := r.Core.GetById(id)
	if err != nil {
		return scheduler.Task{}, err
	}
	return t, nil
}
func (r *Repository) GetNext() (scheduler.Task, error) {
	t, err := r.Core.GetNext()
	if err != nil {
		return scheduler.Task{}, err
	}
	return t, nil
}
func (r *Repository) MarkAsDispatched(id string) error {
	err := r.Core.MarkAsDispatched(id)
	if err == nil {
		r.HookTimer.MarkAsDispatched(id)
	}
	return err
}
func (r *Repository) MarkAsDone(id string, err error) error {
	return r.Core.MarkAsDone(id, err)
}
func (r *Repository) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	updated, err = r.Core.Update(id, param)
	if err == nil {
		r.HookTimer.Update(id, param)
	}
	return updated, err
}

func (r *Repository) StartTimer() {
	r.HookTimer.StartTimer()
}

func (r *Repository) StopTimer() {
	r.HookTimer.StopTimer()
}

func (r *Repository) TimerChannel() <-chan time.Time {
	return r.HookTimer.TimerChannel()
}
