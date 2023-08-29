package inmemory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/ngicks/genericcontainer/heapimpl"
	"github.com/ngicks/gokugen/def"
	sortabletask "github.com/ngicks/gokugen/internal/sortable_task"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type InMemoryRepository struct {
	mu                  sync.Mutex
	insertionOrderCount *atomic.Uint64
	heap                *heapimpl.FilterableHeap[*sortabletask.IndexedTask]
	orderedMap          *orderedmap.OrderedMap[string, *sortabletask.IndexedTask]
	randStrGen          def.RandStrGen
	clock               mockable.Clock
}

func NewInMemoryRepository() *InMemoryRepository {
	r := &InMemoryRepository{
		randStrGen: uuid.NewString,
		clock:      mockable.NewClockReal(),
	}
	r.init()
	return r
}

func (r *InMemoryRepository) init() {
	r.insertionOrderCount = new(atomic.Uint64)
	r.heap = heapimpl.NewFilterableHeapHooks[*sortabletask.IndexedTask](
		sortabletask.Less[*sortabletask.IndexedTask],
		sortabletask.MakeHeapMethodSet[*sortabletask.IndexedTask](),
	)
	r.orderedMap = orderedmap.New[string, *sortabletask.IndexedTask]()
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
	wrapped := sortabletask.WrapTask(t.Clone(), r.insertionOrderCount)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.heap.Push(wrapped)
	r.orderedMap.Set(wrapped.Task.Id, wrapped)
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
	return task.Task.Clone(), nil
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
	if task.Task.State != def.TaskScheduled {
		return def.ErrKindUpdate(*task.Task)
	}
	*(task.Task) = task.Task.Update(param)
	r.heap.Fix(task.Index)
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
	if task.Task.State != def.TaskScheduled {
		return def.ErrKindCancel(*task.Task)
	}

	r.heap.Remove(task.Index)

	task.Task.State = def.TaskCancelled
	task.Task.CancelledAt = option.Some(def.NormalizeTime(r.clock.Now()))

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
	if task.Task.State != def.TaskScheduled {
		return def.ErrKindMarkAsDispatch(*task.Task)
	}
	r.heap.Remove(task.Index)
	task.Task.State = def.TaskDispatched
	task.Task.DispatchedAt = option.Some(def.NormalizeTime(r.clock.Now()))
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
	if task.Task.State != def.TaskDispatched {
		return def.ErrKindMarkAsDone(*task.Task)
	}
	if err == nil {
		task.Task.State = def.TaskDone
	} else {
		task.Task.State = def.TaskErr
		task.Task.Err = err.Error()
	}
	task.Task.DoneAt = option.Some(def.NormalizeTime(r.clock.Now()))
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
		if matcher.Match(*pair.Value.Task) {
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
			out = append(out, pair.Value.Task.Clone())
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
	return r.heap.Peek().Task.Clone(), nil
}
