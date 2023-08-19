package repository

import (
	"context"
	"time"

	"github.com/ngicks/gokugen/def"
)

var _ def.ObservableRepository = (*Repository)(nil)

// Repository is a thin wrapper that connects def.Repository and HookTimer.
// Every mutation is observed and that would update the timer if necessary.
type Repository struct {
	Repository def.Repository
	HookTimer  HookTimer
}

func New(core def.Repository, hookTimer HookTimer) *Repository {
	r := &Repository{
		Repository: core,
		HookTimer:  hookTimer,
	}
	r.HookTimer.SetRepository(r.Repository)
	return r
}

func (r *Repository) Close() error {
	return r.Repository.Close()
}

func (r *Repository) AddTask(ctx context.Context, param def.TaskUpdateParam) (def.Task, error) {
	t, err := r.Repository.AddTask(ctx, param)
	if err != nil {
		return def.Task{}, err
	}

	r.HookTimer.AddTask(ctx, param)

	return t, nil
}

func (r *Repository) GetById(ctx context.Context, id string) (def.Task, error) {
	return r.Repository.GetById(ctx, id)
}

func (r *Repository) UpdateById(ctx context.Context, id string, param def.TaskUpdateParam) error {
	err := r.Repository.UpdateById(ctx, id, param)
	if err != nil {
		return err
	}
	r.HookTimer.UpdateById(ctx, id, param)
	return nil
}

func (r *Repository) Cancel(ctx context.Context, id string) error {
	err := r.Repository.Cancel(ctx, id)
	if err != nil {
		return err
	}
	r.HookTimer.Cancel(ctx, id)
	return nil
}

func (r *Repository) MarkAsDispatched(ctx context.Context, id string) error {
	err := r.Repository.MarkAsDispatched(ctx, id)
	if err != nil {
		return err
	}
	r.HookTimer.MarkAsDispatched(ctx, id)
	return nil
}

func (r *Repository) MarkAsDone(ctx context.Context, id string, err error) error {
	return r.Repository.MarkAsDone(ctx, id, err)
}
func (r *Repository) Find(
	ctx context.Context,
	matcher def.TaskQueryParam,
	offset, limit int,
) ([]def.Task, error) {
	return r.Repository.Find(ctx, matcher, offset, limit)
}

func (r *Repository) GetNext(ctx context.Context) (def.Task, error) {
	return r.Repository.GetNext(ctx)
}

func (r *Repository) LastTimerUpdateError() error {
	return r.HookTimer.LastTimerUpdateError()
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
