package workerpool

import (
	"context"
	"fmt"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/workerpool"
)

type workFn struct {
	ctx      context.Context
	fetcher  func(ctx context.Context) (def.Task, error)
	fetchErr chan error
	workErr  chan error
}

func (f workFn) close() {
	close(f.fetchErr)
	close(f.workErr)
}

func newWorkFn(ctx context.Context, fetcher func(ctx context.Context) (def.Task, error)) workFn {
	return workFn{
		ctx:      ctx,
		fetcher:  fetcher,
		fetchErr: make(chan error, 1),
		workErr:  make(chan error, 1),
	}
}

var _ workerpool.WorkExecuter[string, workFn] = &executor{}

type executor struct {
	workRegistry def.WorkRegistry
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
		notFoundErr := fmt.Errorf("%w: work_id = %s", def.ErrWorkIdNotFound, t.WorkId)
		param.workErr <- notFoundErr
		return notFoundErr
	}

	select {
	case <-combined.Done():
		param.workErr <- combined.Err()
		return combined.Err()
	default:
	}

	var workCtx context.Context = combined
	if t.Deadline.IsSome() {
		deadlineCtx, workFnCancel := context.WithDeadline(combined, t.Deadline.Value())
		defer workFnCancel()
		workCtx = deadlineCtx
	}
	fnErr := (*fn)(workCtx, t.Param)

	param.workErr <- fnErr
	return fnErr
}

type WorkerPool interface {
	Add(delta int) (ok bool)
	Remove(delta int)
	Kill()
	WaitUntil(condition func(alive int, sleeping int, active int) bool, action ...func())
	Wait()
}

var _ def.Dispatcher = &WorkerPoolDispatcher{}

// WorkerPoolDispatcher is an in-memory worker pool backed dispatcher.
type WorkerPoolDispatcher struct {
	WorkerPool   WorkerPool
	workerPool   *workerpool.Pool[string, workFn]
	workRegistry def.WorkRegistry
}

// NewWorkerPoolDispatcher returns in-memory worker pool dispatcher.
// Initially worker pool has zero worker. You must call Add.
func NewWorkerPoolDispatcher(workRegistry def.WorkRegistry) *WorkerPoolDispatcher {
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

func (d *WorkerPoolDispatcher) Dispatch(
	ctx context.Context,
	fetcher func(ctx context.Context) (def.Task, error),
) (<-chan error, error) {
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
