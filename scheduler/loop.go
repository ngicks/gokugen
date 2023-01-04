package scheduler

import (
	"context"
	"sync"

	"github.com/ngicks/type-param-common/set"
)

type LoopHooks interface {
	OnPopError(err error) error
	OnDispatchError(err error) error
	OnUpdateError(updateType UpdateType, err error) error
}

type beingDispatchedIDs struct {
	*set.Set[string]
	sync.Mutex
}

func newBeingDispatchedIDs() beingDispatchedIDs {
	return beingDispatchedIDs{
		Set: set.New[string](),
	}
}

func (ids *beingDispatchedIDs) Add(id string) {
	ids.Lock()
	ids.Set.Add(id)
	ids.Unlock()
}

func (ids *beingDispatchedIDs) Delete(id string) {
	ids.Lock()
	ids.Set.Delete(id)
	ids.Unlock()
}

func (ids *beingDispatchedIDs) Has(id string) bool {
	ids.Lock()
	has := ids.Set.Has(id)
	ids.Unlock()
	return has
}

type loop struct {
	dispatcher Dispatcher
	repo       TaskRepository
	hooks      LoopHooks

	beingDispatched beingDispatchedIDs
	errCh           chan error

	updateEventQueue *updateEventQueue
}

func newLoop(dispatcher Dispatcher, repo TaskRepository, hooks LoopHooks) loop {
	return loop{
		dispatcher: dispatcher,
		repo:       repo,
		hooks:      hooks,

		beingDispatched: newBeingDispatchedIDs(),
		errCh:           make(chan error),

		updateEventQueue: newUpdateEventQueue(),
	}
}

func (l *loop) StartTimer() {
	l.repo.StartTimer()
}

func (l *loop) StopTimer() {
	l.repo.StopTimer()
}

func (l *loop) Run(ctx context.Context, startTimer, stopTimerOnClose bool) error {
	updateCtx, cancel := context.WithCancel(ctx)
	doneCh := make(chan struct{})
	go func() {
		l.runUpdateLoop(updateCtx)
		close(doneCh)
	}()

	defer func() {
		cancel()
		<-doneCh
	}()

	if startTimer {
		l.repo.StartTimer()
	}

	if stopTimerOnClose {
		defer l.repo.StopTimer()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-l.repo.TimerChannel():
			err := l.dispatch(ctx)
			if err != nil {
				return err
			}
		case err := <-l.errCh:
			return err
		}
	}
}

func (l *loop) dispatch(ctx context.Context) error {
	task, err := l.repo.GetNext()
	if err != nil {
		repoErr, ok := err.(*RepositoryError)
		if !ok || repoErr.Kind != Empty {
			hookErr := l.hooks.OnPopError(err)
			if hookErr != nil {
				return hookErr
			}
			return nil
		}
	}

	resultCh, err := l.dispatcher.Dispatch(
		ctx,
		func() (Task, error) {
			l.beingDispatched.Add(task.Id)
			task, err := l.repo.GetById(task.Id)
			if err != nil {
				return Task{}, err
			}
			if task.CancelledAt != nil {
				return Task{}, &RepositoryError{Id: task.Id, Kind: AlreadyCancelled}
			}
			return task, nil
		},
	)

	if err != nil {
		if IsAlreadyCancelled(err) {
			return nil
		}

		hookErr := l.hooks.OnDispatchError(err)
		if hookErr != nil {
			return hookErr
		}
		return nil
	}

	err = l.repo.MarkAsDispatched(task.Id)
	if err != nil {
		hookErr := l.hooks.OnUpdateError(MarkAsDispatched, err)
		if hookErr != nil {
			return hookErr
		}
	}

	l.updateEventQueue.Reserve(
		task.Id,
		func() updateEvent {
			err := <-resultCh
			return updateEvent{
				id:         task.Id,
				updateType: MarkAsDone,
				err:        err,
			}
		},
	)

	return nil
}

func (l *loop) runUpdateLoop(ctx context.Context) error {
	queueContext, cancel := context.WithCancel(ctx)

	startCh := make(chan struct{}, 1)
	doneCh := make(chan struct{})
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		l.updateEventQueue.Run(queueContext, startCh)
		close(doneCh)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		select {
		case <-ctx.Done():
		case <-doneCh:
		}
		cancel()
		wg.Done()
	}()

	defer func() {
		cancel()
		wg.Wait()
	}()

	<-startCh

	l.updateLoop()

	return nil
}

func (l *loop) updateLoop() {
	for {
		event, ok := <-l.updateEventQueue.Subscribe()
		if !ok {
			return
		}

		var result error
		switch event.updateType {
		case CancelTask, UpdateParam:
			if l.beingDispatched.Has(event.id) {
				result = &RepositoryError{Id: event.id, Kind: AlreadyDispatched}
			} else {
				if event.updateType == CancelTask {
					_, result = l.repo.Cancel(event.id)
				} else if event.updateType == UpdateParam {
					_, result = l.repo.Update(event.id, event.param)
				}
			}
		case MarkAsDone:
			l.markAsDone(event)
		}

		if event.responseCh != nil {
			event.responseCh <- result
		}
	}
}

func (l *loop) Cancel(id string) error {
	if l.updateEventQueue.IsClosed() {
		_, err := l.repo.Cancel(id)
		return err
	}

	errCh := make(chan error)
	l.updateEventQueue.Push(updateEvent{
		id:         id,
		updateType: CancelTask,
		responseCh: errCh,
	})
	return <-errCh
}

func (l *loop) Update(id string, param TaskParam) error {
	if l.updateEventQueue.IsClosed() {
		_, err := l.repo.Update(id, param)
		return err
	}

	errCh := make(chan error)
	l.updateEventQueue.Push(updateEvent{
		id:         id,
		updateType: UpdateParam,
		param:      param,
		responseCh: errCh,
	})
	return <-errCh
}

func (l *loop) markAsDone(event updateEvent) {
	markErr := l.repo.MarkAsDone(event.id, event.err)
	l.beingDispatched.Delete(event.id)
	if markErr != nil {
		hookErr := l.hooks.OnUpdateError(event.updateType, markErr)
		if hookErr != nil {
			l.errCh <- hookErr
		}
	}
}
