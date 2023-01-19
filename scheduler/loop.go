package scheduler

import (
	"context"
	"sync"

	"github.com/ngicks/gommon/pkg/lockmap"
)

type beingDispatchedIDs struct {
	*lockmap.LockMap[string, struct{}]
}

func newBeingDispatchedIDs() beingDispatchedIDs {
	return beingDispatchedIDs{
		LockMap: lockmap.New[string, struct{}](),
	}
}

func (ids *beingDispatchedIDs) Add(id string) {
	ids.LockMap.Set(id, struct{}{})
}

func (ids *beingDispatchedIDs) Has(id string) bool {
	_, ok := ids.LockMap.Get(id)
	return ok
}

func (ids *beingDispatchedIDs) RunWithinLock(id string, fn func(has bool)) {
	ids.LockMap.RunWithinLock(id, func(v struct{}, has bool, set func(v struct{})) { fn(has) })
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
		if !IsEmpty(err) {
			hookErr := l.hooks.OnGetNextError(err)
			if hookErr != nil {
				return hookErr
			}
			return nil
		}
		// log when IsEmpty == true ?
		return nil
	}

	l.hooks.OnGetNext(task)

	var updated Task
	resultCh, err := l.dispatcher.Dispatch(
		ctx,
		func(ctx context.Context) (Task, error) {
			l.beingDispatched.Add(task.Id)
			task, err := l.repo.GetById(task.Id)
			if err != nil {
				return Task{}, err
			}
			if task.CancelledAt != nil {
				return Task{}, &RepositoryError{Id: task.Id, Kind: AlreadyCancelled}
			}
			updated = task
			return task, nil
		},
	)

	if err != nil {
		l.beingDispatched.Delete(task.Id)
		if IsAlreadyCancelled(err) {
			return nil
		}

		hookErr := l.hooks.OnDispatchError(task, err)
		if hookErr != nil {
			return hookErr
		}
		return nil
	}

	l.hooks.OnDispatch(task)

	err = l.repo.MarkAsDispatched(task.Id)
	if err != nil {
		l.beingDispatched.Delete(task.Id)
		hookErr := l.hooks.OnUpdateError(updated, MarkAsDispatched, err)
		if hookErr != nil {
			return hookErr
		}
	}

	l.hooks.OnUpdate(updated, MarkAsDispatched)

	l.updateEventQueue.Reserve(
		task.Id,
		func() updateEvent {
			err := <-resultCh
			return updateEvent{
				id:         task.Id,
				updateType: MarkAsDone,
				task:       updated,
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
			l.beingDispatched.RunWithinLock(event.id, func(has bool) {
				if has {
					result = &RepositoryError{Id: event.id, Kind: AlreadyDispatched}
					return
				} else {
					if event.updateType == CancelTask {
						_, result = l.repo.Cancel(event.id)
					} else if event.updateType == UpdateParam {
						_, result = l.repo.Update(event.id, event.param)
					}
				}
			})
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
		return ErrNotRunning
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
		return ErrNotRunning
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
		hookErr := l.hooks.OnUpdateError(event.task, event.updateType, markErr)
		if hookErr != nil {
			l.errCh <- hookErr
		}
	} else {
		l.hooks.OnTaskDone(event.task, event.err)
	}
}
