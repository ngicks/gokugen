package scheduler

import (
	"context"
	"time"

	"github.com/ngicks/gokugen/def"
)

var _ def.ObservableRepository = (*nopRepository)(nil)

type nopRepository struct{}

func (*nopRepository) Close() (e error) { return }
func (*nopRepository) AddTask(
	ctx context.Context,
	param def.TaskUpdateParam,
) (t def.Task, e error) {
	return
}
func (*nopRepository) GetById(ctx context.Context, id string) (t def.Task, e error) { return }
func (*nopRepository) UpdateById(
	ctx context.Context,
	id string,
	param def.TaskUpdateParam,
) (e error) {
	return
}
func (*nopRepository) Cancel(ctx context.Context, id string) (e error) {
	return
}
func (*nopRepository) MarkAsDispatched(ctx context.Context, id string) (e error) {
	return
}
func (*nopRepository) MarkAsDone(ctx context.Context, id string, err error) (e error) {
	return
}
func (*nopRepository) Find(
	ctx context.Context,
	matcher def.TaskQueryParam,
	offset, limit int,
) (t []def.Task, e error) {
	return
}
func (*nopRepository) GetNext(ctx context.Context) (t def.Task, e error) {
	return
}
func (*nopRepository) LastTimerUpdateError() (e error) { return }
func (*nopRepository) StartTimer(ctx context.Context)  {}
func (*nopRepository) StopTimer()                      {}
func (*nopRepository) TimerChannel() <-chan time.Time  { return make(<-chan time.Time) }

var _ VolatileTask = (*nopVolatileTask)(nil)

type nopVolatileTask struct{}

func (*nopVolatileTask) LastTimerUpdateError() error {
	return nil
}
func (*nopVolatileTask) StartTimer(context.Context) {}
func (*nopVolatileTask) StopTimer()                 {}
func (*nopVolatileTask) TimerChannel() <-chan time.Time {
	return make(<-chan time.Time)
}
func (*nopVolatileTask) GetNext(context.Context) (def.Task, error) {
	return def.Task{}, &def.RepositoryError{Kind: def.Exhausted}
}
func (*nopVolatileTask) Pop(ctx context.Context) (def.Task, error) {
	return def.Task{}, &def.RepositoryError{Kind: def.Exhausted}
}
