package dispatcher

import (
	"context"
	"fmt"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/workerpool"
)

type WorkFn = func(ctx context.Context, param []byte) error

type WorkRegistry interface {
	Load(workId string) (fn WorkFn, ok bool)
}

type ErrWorkIdNotFound struct {
	WorkId string
}

func (e *ErrWorkIdNotFound) Error() string {
	return fmt.Sprintf("work id not found. id = %s", e.WorkId)
}

type workFn struct {
	ctx      context.Context
	fetcher  func(ctx context.Context) (scheduler.Task, error)
	fetchErr chan error
	workErr  chan error
}

func newWorkFn(ctx context.Context, fetcher func(ctx context.Context) (scheduler.Task, error)) *workFn {
	return &workFn{
		ctx:      ctx,
		fetcher:  fetcher,
		fetchErr: make(chan error),
		workErr:  make(chan error),
	}
}

var _ workerpool.WorkExecuter[*workFn] = &executor{}

type executor struct {
	workRegistry WorkRegistry
}

func (e *executor) Exec(ctx context.Context, param *workFn) error {
	t, err := param.fetcher(ctx)
	if err != nil {
		param.fetchErr <- err
		return err
	}

	fn, ok := e.workRegistry.Load(t.WorkId)
	if !ok {
		err := &ErrWorkIdNotFound{WorkId: t.WorkId}
		param.fetchErr <- err
		return err
	}

	param.fetchErr <- nil

	combined, cancel := context.WithCancel(param.ctx)
	defer cancel()

	go func() {
		select {
		case <-combined.Done():
		case <-ctx.Done():
		}
		cancel()
	}()

	err = fn(combined, t.Param)
	param.workErr <- err

	return err
}

type WorkerPool interface {
	Add(delta int)
	Remove(delta int)
	Kill()
	Wait()
}

var _ scheduler.Dispatcher = &WorkerPoolDispatcher{}

type WorkerPoolDispatcher struct {
	WorkerPool   WorkerPool
	workerPool   *workerpool.Pool[*workFn]
	workRegistry WorkRegistry
}

// NewWorkerPoolDispatcher returns in-memory worker pool dispatcher.
// Initially worker pool has zero worker. You must call Add.
func NewWorkerPoolDispatcher(workRegistry WorkRegistry) *WorkerPoolDispatcher {
	pool := workerpool.New[*workFn](&executor{workRegistry: workRegistry})
	return &WorkerPoolDispatcher{
		workRegistry: workRegistry,
		workerPool:   pool,
		WorkerPool:   pool,
	}
}

func (d *WorkerPoolDispatcher) Dispatch(ctx context.Context, fetcher func(ctx context.Context) (scheduler.Task, error)) (<-chan error, error) {
	w := newWorkFn(ctx, fetcher)
	select {
	case d.workerPool.Sender() <- w:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	err := <-w.fetchErr
	if err != nil {
		return nil, err
	}
	return w.workErr, nil
}
