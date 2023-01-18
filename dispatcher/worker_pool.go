package dispatcher

import (
	"context"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/workerpool"
)

type workFn struct {
	ctx      context.Context
	fetcher  func(ctx context.Context) (scheduler.Task, error)
	fetchErr chan error
	workErr  chan error
}

func (f workFn) close() {
	close(f.fetchErr)
	close(f.workErr)
}

func newWorkFn(ctx context.Context, fetcher func(ctx context.Context) (scheduler.Task, error)) workFn {
	return workFn{
		ctx:      ctx,
		fetcher:  fetcher,
		fetchErr: make(chan error),
		workErr:  make(chan error),
	}
}

var _ workerpool.WorkExecuter[string, workFn] = &executor{}

type executor struct {
	workRegistry scheduler.WorkRegistry
}

func (e *executor) Exec(ctx context.Context, id string, param workFn) error {
	defer param.close()

	combined, cancel := context.WithCancel(param.ctx)
	defer cancel()

	go func() {
		select {
		case <-combined.Done():
		case <-param.ctx.Done():
		case <-ctx.Done():
		}
		cancel()
	}()

	t, fetchErr := param.fetcher(combined)
	if fetchErr != nil {
		param.fetchErr <- fetchErr
		return fetchErr
	}

	param.fetchErr <- nil

	fn, ok := e.workRegistry.Load(t.WorkId)
	if !ok {
		notFoundErr := &scheduler.ErrWorkIdNotFound{Param: t.ToParam()}
		param.workErr <- notFoundErr
		return notFoundErr
	}

	select {
	case <-combined.Done():
		param.workErr <- combined.Err()
		return combined.Err()
	default:
	}

	fnErr := fn(combined, t.Param)

	param.workErr <- fnErr
	return fnErr
}

type WorkerPool interface {
	Add(delta int) (ok bool)
	Remove(delta int)
	Kill()
	Wait()
}

var _ scheduler.Dispatcher = &WorkerPoolDispatcher{}

// WorkerPoolDispatcher is an in-memory worker pool backed dispatcher.
type WorkerPoolDispatcher struct {
	WorkerPool   WorkerPool
	workerPool   *workerpool.Pool[string, workFn]
	workRegistry scheduler.WorkRegistry
}

// NewWorkerPoolDispatcher returns in-memory worker pool dispatcher.
// Initially worker pool has zero worker. You must call Add.
func NewWorkerPoolDispatcher(workRegistry scheduler.WorkRegistry) *WorkerPoolDispatcher {
	pool := workerpool.New[string, workFn](
		&executor{workRegistry: workRegistry},
		workerpool.NewUuidPool(),
	)
	return &WorkerPoolDispatcher{
		workRegistry: workRegistry,
		workerPool:   pool,
		WorkerPool:   pool,
	}
}

func (d *WorkerPoolDispatcher) Dispatch(ctx context.Context, fetcher func(ctx context.Context) (scheduler.Task, error)) (<-chan error, error) {
	w := newWorkFn(ctx, fetcher)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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
