package scheduler

import (
	"context"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/inmemory"
	"github.com/ngicks/mockable"
)

var _ def.ObservableRepository = (*mockRepository)(nil)

type mockRepository struct {
	*inmemory.InMemoryRepository
	RepoErr           []error
	TimerErr          []error
	TimerStarted      bool
	NextScheduledTime time.Time
	Clock             *mockable.ClockFake
}

func popErr(errors *[]error) error {
	if len(*errors) == 0 {
		return nil
	}
	p := (*errors)[0]
	(*errors) = (*errors)[1:]
	return p
}

func (r *mockRepository) GetById(ctx context.Context, id string) (def.Task, error) {
	if err := popErr(&r.RepoErr); err != nil {
		return def.Task{}, err
	}
	return r.InMemoryRepository.GetById(ctx, id)
}
func (r *mockRepository) GetNext(ctx context.Context) (def.Task, error) {
	if err := popErr(&r.RepoErr); err != nil {
		return def.Task{}, err
	}
	return r.InMemoryRepository.GetNext(ctx)
}
func (r *mockRepository) MarkAsDispatched(ctx context.Context, id string) error {
	if err := popErr(&r.RepoErr); err != nil {
		return err
	}
	return r.InMemoryRepository.MarkAsDispatched(ctx, id)
}
func (r *mockRepository) MarkAsDone(ctx context.Context, id string, err error) error {
	if err := popErr(&r.RepoErr); err != nil {
		return err
	}
	return r.InMemoryRepository.MarkAsDone(ctx, id, err)
}

func (r *mockRepository) LastTimerUpdateError() error {
	return popErr(&r.TimerErr)
}
func (r *mockRepository) StartTimer(ctx context.Context) {
	r.TimerStarted = true
}
func (r *mockRepository) StopTimer() {
	r.TimerStarted = false
}
func (r *mockRepository) NextScheduled() (time.Time, bool) {
	if r.TimerStarted {
		return r.NextScheduledTime, true
	} else {
		return time.Time{}, false
	}
}
func (r *mockRepository) TimerChannel() <-chan time.Time {
	return r.Clock.C()
}

var _ def.Dispatcher = (*mockDispatcher)(nil)

type pair struct {
	Task def.Task
	Ch   chan error
}

type mockDispatcher struct {
	DispatchErr error
	Pending     map[string]pair
}

func (d *mockDispatcher) Dispatch(
	ctx context.Context,
	fetcher func(ctx context.Context) (def.Task, error),
) (<-chan error, error) {
	if d.DispatchErr != nil {
		return nil, d.DispatchErr
	}
	task, err := fetcher(ctx)
	if err != nil {
		return nil, err
	}
	pending := pair{
		Task: task,
		Ch:   make(chan error),
	}
	d.Pending[task.Id] = pending
	return pending.Ch, nil
}

func (d *mockDispatcher) Unblock(id string) {
	d.Pending[id].Ch <- nil
	close(d.Pending[id].Ch)
	delete(d.Pending, id)
}

func (d *mockDispatcher) UnblockOne() (ok bool) {
	for k := range d.Pending {
		d.Unblock(k)
		return true
	}
	return false
}
