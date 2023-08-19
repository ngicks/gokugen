package inmemory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/ngicks/genericcontainer/heapimpl"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type InMemoryRepository struct {
	mu                  sync.Mutex
	insertionOrderCount *atomic.Uint64
	heap                *heapimpl.FilterableHeap[*indexedTask]
	orderedMap          *orderedmap.OrderedMap[string, *indexedTask]
	randStrGen          def.RandStrGen
	clock               mockable.Clock
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		insertionOrderCount: new(atomic.Uint64),
		heap:                heapimpl.NewFilterableHeap[*indexedTask](),
		orderedMap:          orderedmap.New[string, *indexedTask](),
		randStrGen:          uuid.NewString,
		clock:               mockable.NewClockReal(),
	}
}

func (r *InMemoryRepository) Close() error {
	return nil
}

func (r *InMemoryRepository) AddTask(
	ctx context.Context,
	param def.TaskUpdateParam,
) (def.Task, error) {
	param = param.Normalize()

	if ctx.Err() != nil {
		return def.Task{}, ctx.Err()
	}
	t := param.ToTask(r.randStrGen(), r.clock.Now())
	if !t.IsValid() {
		return def.Task{},
			fmt.Errorf("%w. reason = %v", def.ErrInvalidTask, t.ReportInvalidity())
	}
	wrapped := wrapTask(t.Clone(), r.insertionOrderCount)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.heap.Push(wrapped)
	r.orderedMap.Set(wrapped.task.Id, wrapped)
	return t.Clone(), nil
}

func (r *InMemoryRepository) GetById(ctx context.Context, id string) (def.Task, error) {
	if ctx.Err() != nil {
		return def.Task{}, ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.orderedMap.Get(id)
	if !ok {
		return def.Task{},
			&def.RepositoryError{Kind: def.IdNotFound, Id: id}
	}
	return task.task.Clone(), nil
}

func (r *InMemoryRepository) UpdateById(
	ctx context.Context,
	id string,
	param def.TaskUpdateParam,
) error {
	param = param.Normalize()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.orderedMap.Get(id)
	if !ok {
		return &def.RepositoryError{Kind: def.IdNotFound, Id: id}
	}
	if task.task.State != def.TaskScheduled {
		return def.ErrKindUpdate(*task.task)
	}
	*(task.task) = task.task.Update(param)
	r.heap.Fix(task.index)
	return nil
}

func (r *InMemoryRepository) Cancel(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.orderedMap.Get(id)
	if !ok {
		return &def.RepositoryError{Kind: def.IdNotFound, Id: id}
	}
	if task.task.State != def.TaskScheduled {
		return def.ErrKindCancel(*task.task)
	}

	r.heap.Remove(task.index)

	task.task.State = def.TaskCancelled
	task.task.CancelledAt = option.Some(def.NormalizeTime(r.clock.Now()))

	return nil
}

func (r *InMemoryRepository) MarkAsDispatched(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.orderedMap.Get(id)
	if !ok {
		return &def.RepositoryError{Kind: def.IdNotFound, Id: id}
	}
	if task.task.State != def.TaskScheduled {
		return def.ErrKindMarkAsDispatch(*task.task)
	}
	r.heap.Remove(task.index)
	task.task.State = def.TaskDispatched
	task.task.DispatchedAt = option.Some(def.NormalizeTime(r.clock.Now()))
	return nil
}

func (r *InMemoryRepository) MarkAsDone(ctx context.Context, id string, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.orderedMap.Get(id)
	if !ok {
		return &def.RepositoryError{Kind: def.IdNotFound, Id: id}
	}
	if task.task.State != def.TaskDispatched {
		return def.ErrKindMarkAsDone(*task.task)
	}
	if err == nil {
		task.task.State = def.TaskDone
	} else {
		task.task.State = def.TaskErr
		task.task.Err = err.Error()
	}
	task.task.DoneAt = option.Some(def.NormalizeTime(r.clock.Now()))
	return nil
}

func (r *InMemoryRepository) Find(
	ctx context.Context,
	matcher def.TaskQueryParam,
	offset, limit int,
) ([]def.Task, error) {
	matcher = matcher.Normalize()

	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]def.Task, 0)
	for pair := r.orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		if matcher.Match(*pair.Value.task) {
			if offset != 0 {
				offset--
				continue
			}
			if limit == 0 {
				break
			}
			if limit > 0 {
				limit--
			}
			out = append(out, pair.Value.task.Clone())
		}
	}
	return out, nil
}

func (r *InMemoryRepository) GetNext(ctx context.Context) (def.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.heap.Len() == 0 {
		return def.Task{}, &def.RepositoryError{Kind: def.Exhausted}
	}
	return r.heap.Peek().task.Clone(), nil
}
